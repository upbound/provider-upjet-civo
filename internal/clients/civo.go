package clients

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/civo/terraform-provider-civo/civo"
	"github.com/crossplane/crossplane-runtime/v2/pkg/errors"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	terraformsdk "github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/upjet/v2/pkg/terraform"

	clusterv1beta1 "github.com/upbound/provider-civo/apis/cluster/v1beta1"
	namespacedv1beta1 "github.com/upbound/provider-civo/apis/namespaced/v1beta1"
)

const (
	// error messages
	errNoProviderConfig     = "no providerConfigRef provided"
	errGetProviderConfig    = "cannot get referenced ProviderConfig"
	errTrackUsage           = "cannot track ProviderConfig usage"
	errExtractCredentials   = "cannot extract credentials"
	errUnmarshalCredentials = "cannot unmarshal civo credentials as JSON"

	keyToken           = "token"
	keyCredentialsFile = "credentials_file"
	keyAPIEndpoint     = "api_endpoint"

	// EnvToken is the environment variable the upstream provider reads before
	// the credentials file. Set on the provider process it would win over
	// every ProviderConfig, so the provider refuses to start with it.
	EnvToken = "CIVO_TOKEN"

	// DefaultCredentialsDir is where the short-lived credentials files are
	// written unless the --credentials-dir flag says otherwise.
	DefaultCredentialsDir = "/tmp/provider-civo"

	// credentialsAPIKeyName is the key the token is stored under in the
	// credentials file. The upstream parser reads whichever key
	// meta.current_apikey names, so the name only has to match itself.
	credentialsAPIKeyName = "provider-civo"
	credentialsFilePrefix = "civo-credentials-"
	credentialsFileSuffix = ".json"
	credentialsDirMode    = 0o700
)

// credentialsFileTTL bounds how long a credentials file stays on disk. The
// upstream provider reads the file before it validates the token with an API
// call that has no timeout, so a stalled call would otherwise keep the file
// around for as long as the connection hangs. A configure call that has not
// read the file by then fails and is retried on the next reconcile.
var credentialsFileTTL = 30 * time.Second

// credentialsFile is the format the upstream provider's credentials_file
// parser expects, the one the Civo CLI writes to ~/.civo.json.
type credentialsFile struct {
	APIKeys map[string]string `json:"apikeys"`
	Meta    struct {
		CurrentAPIKey string `json:"current_apikey"`
	} `json:"meta"`
}

// TerraformSetupBuilder builds a terraform.SetupFn that configures the
// in-process Civo Terraform provider (no-fork, no terraform CLI). The API
// token reaches the upstream provider through a credentials file written to
// credentialsDir for the duration of one configure call.
func TerraformSetupBuilder(credentialsDir string) terraform.SetupFn {
	return func(ctx context.Context, client client.Client, mg resource.Managed) (terraform.Setup, error) {
		pcSpec, err := resolveProviderConfig(ctx, client, mg)
		if err != nil {
			return terraform.Setup{}, errors.Wrap(err, "cannot resolve provider config")
		}

		data, err := resource.CommonCredentialExtractor(ctx, pcSpec.Credentials.Source, client, pcSpec.Credentials.CommonCredentialSelectors)
		if err != nil {
			return terraform.Setup{}, errors.Wrap(err, errExtractCredentials)
		}

		creds := map[string]string{}
		if err := json.Unmarshal(data, &creds); err != nil {
			return terraform.Setup{}, errors.Wrap(err, errUnmarshalCredentials)
		}

		return configure(ctx, credentialsDir, string(mg.GetUID()), creds[keyToken], pcSpec.APIEndpoint)
	}
}

// configure hands the token to the upstream provider through a credentials
// file, configures a provider instance with it and removes the file again
// when Configure returns, or after credentialsFileTTL at the latest. Nothing
// reads the file after Configure: the token lives in the Civo API client
// stored as provider meta from then on.
func configure(ctx context.Context, credentialsDir, uid, token string, apiEndpoint *string) (terraform.Setup, error) {
	path, err := writeCredentialsFile(credentialsDir, uid, token)
	if err != nil {
		return terraform.Setup{}, errors.Wrap(err, "cannot write the credentials file")
	}
	defer func() { _ = os.Remove(path) }()
	watchdog := time.AfterFunc(credentialsFileTTL, func() { _ = os.Remove(path) })
	defer watchdog.Stop()

	ps := terraform.Setup{Configuration: buildConfiguration(path, apiEndpoint)}
	return ps, errors.Wrap(configureProviderMeta(ctx, &ps), "cannot configure the civo provider client")
}

// writeCredentialsFile writes the token, in the upstream credentials file
// format, to a new file in dir that only the provider's user can read, and
// returns its path. The name carries the managed resource's UID and a random
// part, so every call gets a file of its own.
func writeCredentialsFile(dir, uid, token string) (string, error) {
	if token == "" {
		return "", errors.New(`credentials secret has no "token" key`)
	}
	cf := credentialsFile{APIKeys: map[string]string{credentialsAPIKeyName: token}}
	cf.Meta.CurrentAPIKey = credentialsAPIKeyName
	content, err := json.Marshal(cf)
	if err != nil {
		return "", errors.Wrap(err, "cannot encode the credentials file")
	}
	f, err := os.CreateTemp(dir, credentialsFilePrefix+uid+"-*"+credentialsFileSuffix)
	if err != nil {
		return "", errors.Wrap(err, "cannot create the credentials file")
	}
	if _, err := f.Write(content); err != nil {
		_ = f.Close()
		_ = os.Remove(f.Name())
		return "", errors.Wrap(err, "cannot write the credentials file")
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(f.Name())
		return "", errors.Wrap(err, "cannot close the credentials file")
	}
	return f.Name(), nil
}

// PrepareCredentialsDir creates the credentials directory, readable by the
// provider's user only, and removes credentials files an earlier process left
// behind, for example after a crash between writing one and configuring the
// provider.
func PrepareCredentialsDir(dir string) error {
	if err := os.MkdirAll(dir, credentialsDirMode); err != nil {
		return errors.Wrapf(err, "cannot create the credentials directory %s", dir)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return errors.Wrapf(err, "cannot list the credentials directory %s", dir)
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasPrefix(name, credentialsFilePrefix) || !strings.HasSuffix(name, credentialsFileSuffix) {
			continue
		}
		if err := os.Remove(filepath.Join(dir, name)); err != nil && !os.IsNotExist(err) {
			return errors.Wrapf(err, "cannot remove the stale credentials file %s", name)
		}
	}
	return nil
}

// buildConfiguration maps the credentials file and the ProviderConfig spec
// onto the Civo provider configuration block. The token is handed over
// through "credentials_file" rather than the deprecated "token" attribute, and
// through a file of its own per call rather than the process-global
// CIVO_TOKEN environment variable, so that every ProviderConfig gets its own
// client. The client is never scoped to a region, because every regional
// resource scopes its own API calls upstream, and the API endpoint is only set
// when the ProviderConfig declares one, so the public https://api.civo.com
// applies otherwise.
func buildConfiguration(credentialsFile string, apiEndpoint *string) map[string]any {
	cfg := map[string]any{keyCredentialsFile: credentialsFile}
	if apiEndpoint != nil && *apiEndpoint != "" {
		cfg[keyAPIEndpoint] = *apiEndpoint
	}
	return cfg
}

// configureProviderMeta configures a fresh in-process SDKv2 provider instance
// with the resolved configuration and stores the resulting provider meta (the
// Civo API client) in the Setup, as required by upjet's TerraformPluginSDK
// connectors. A fresh instance is used per call because provider meta is
// per-ProviderConfig state. Note that the upstream configure function
// validates the token with a ListRegions API call on every invocation.
func configureProviderMeta(ctx context.Context, ps *terraform.Setup) error {
	p := civo.Provider()
	diag := p.Configure(context.WithoutCancel(ctx), &terraformsdk.ResourceConfig{Config: ps.Configuration})
	if diag != nil && diag.HasError() {
		return errors.Errorf("failed to configure the civo provider: %v", diag)
	}
	ps.Meta = p.Meta()
	return nil
}

func toSharedPCSpec(pc *clusterv1beta1.ProviderConfig) (*namespacedv1beta1.ProviderConfigSpec, error) {
	if pc == nil {
		return nil, nil
	}
	data, err := json.Marshal(pc.Spec)
	if err != nil {
		return nil, err
	}

	var mSpec namespacedv1beta1.ProviderConfigSpec
	err = json.Unmarshal(data, &mSpec)
	return &mSpec, err
}

func resolveProviderConfig(ctx context.Context, crClient client.Client, mg resource.Managed) (*namespacedv1beta1.ProviderConfigSpec, error) {
	switch managed := mg.(type) {
	case resource.LegacyManaged:
		return resolveLegacy(ctx, crClient, managed)
	case resource.ModernManaged:
		return resolveModern(ctx, crClient, managed)
	default:
		return nil, errors.New("resource is not a managed resource")
	}
}

func resolveLegacy(ctx context.Context, client client.Client, mg resource.LegacyManaged) (*namespacedv1beta1.ProviderConfigSpec, error) {
	configRef := mg.GetProviderConfigReference()
	if configRef == nil {
		return nil, errors.New(errNoProviderConfig)
	}
	pc := &clusterv1beta1.ProviderConfig{}
	if err := client.Get(ctx, types.NamespacedName{Name: configRef.Name}, pc); err != nil {
		return nil, errors.Wrap(err, errGetProviderConfig)
	}

	t := resource.NewLegacyProviderConfigUsageTracker(client, &clusterv1beta1.ProviderConfigUsage{})
	if err := t.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackUsage)
	}

	return toSharedPCSpec(pc)
}

// resolveNamespacedSpec converts a namespaced ProviderConfig spec into the
// internal, fully-resolved ProviderConfigSpec. The namespaced spec omits the
// secret namespace, so it is resolved to the namespace of the referencing
// managed resource here.
func resolveNamespacedSpec(spec namespacedv1beta1.NamespacedProviderConfigSpec, namespace string) namespacedv1beta1.ProviderConfigSpec {
	resolved := namespacedv1beta1.ProviderConfigSpec{
		ReconciliationPolicy: spec.ReconciliationPolicy,
		APIEndpoint:          spec.APIEndpoint,
		Credentials: namespacedv1beta1.ProviderCredentials{
			Source: spec.Credentials.Source,
			CommonCredentialSelectors: xpv2.CommonCredentialSelectors{
				Fs:  spec.Credentials.Fs,
				Env: spec.Credentials.Env,
			},
		},
	}
	if spec.Credentials.SecretRef != nil {
		resolved.Credentials.SecretRef = spec.Credentials.SecretRef.ToSecretKeySelector(namespace)
	}
	return resolved
}

func resolveModern(ctx context.Context, crClient client.Client, mg resource.ModernManaged) (*namespacedv1beta1.ProviderConfigSpec, error) {
	configRef := mg.GetProviderConfigReference()
	if configRef == nil {
		return nil, errors.New(errNoProviderConfig)
	}

	pcRuntimeObj, err := crClient.Scheme().New(namespacedv1beta1.SchemeGroupVersion.WithKind(configRef.Kind))
	if err != nil {
		return nil, errors.Wrap(err, "unknown GVK for ProviderConfig")
	}
	pcObj, ok := pcRuntimeObj.(client.Object)
	if !ok {
		// This indicates a programming error, types are not properly generated
		return nil, errors.New("runtime object is not a client.Object")
	}

	// Namespace will be ignored if the PC is a cluster-scoped type
	if err := crClient.Get(ctx, types.NamespacedName{Name: configRef.Name, Namespace: mg.GetNamespace()}, pcObj); err != nil {
		return nil, errors.Wrap(err, errGetProviderConfig)
	}

	var pcSpec namespacedv1beta1.ProviderConfigSpec
	pcu := &namespacedv1beta1.ProviderConfigUsage{}
	switch pc := pcObj.(type) {
	case *namespacedv1beta1.ProviderConfig:
		pcSpec = resolveNamespacedSpec(pc.Spec, mg.GetNamespace())
	case *namespacedv1beta1.ClusterProviderConfig:
		pcSpec = pc.Spec
	default:
		return nil, errors.New("unknown provider config type")
	}
	t := resource.NewProviderConfigUsageTracker(crClient, pcu)
	if err := t.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackUsage)
	}
	return &pcSpec, nil
}
