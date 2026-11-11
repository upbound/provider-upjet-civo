package clients

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/civo/civogo"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/google/go-cmp/cmp"

	clusterkubernetesv1beta1 "github.com/upbound/provider-civo/apis/cluster/kubernetes/v1beta1"
	clustervpcv1beta1 "github.com/upbound/provider-civo/apis/cluster/vpc/v1beta1"
	namespacedkubernetesv1beta1 "github.com/upbound/provider-civo/apis/namespaced/kubernetes/v1beta1"
	namespacedv1beta1 "github.com/upbound/provider-civo/apis/namespaced/v1beta1"
)

func TestResolveNamespacedSpec(t *testing.T) {
	type args struct {
		spec      namespacedv1beta1.NamespacedProviderConfigSpec
		namespace string
	}

	cases := map[string]struct {
		args args
		want namespacedv1beta1.ProviderConfigSpec
	}{
		"SecretRefResolvesToReferencerNamespace": {
			args: args{
				namespace: "team-a",
				spec: namespacedv1beta1.NamespacedProviderConfigSpec{
					Credentials: namespacedv1beta1.NamespacedProviderCredentials{
						Source: xpv2.CredentialsSourceSecret,
						SecretRef: &xpv2.LocalSecretKeySelector{
							LocalSecretReference: xpv2.LocalSecretReference{Name: "creds"},
							Key:                  "credentials",
						},
					},
				},
			},
			want: namespacedv1beta1.ProviderConfigSpec{
				Credentials: namespacedv1beta1.ProviderCredentials{
					Source: xpv2.CredentialsSourceSecret,
					CommonCredentialSelectors: xpv2.CommonCredentialSelectors{
						SecretRef: &xpv2.SecretKeySelector{
							SecretReference: xpv2.SecretReference{Name: "creds", Namespace: "team-a"},
							Key:             "credentials",
						},
					},
				},
			},
		},
		"NilSecretRefStaysNil": {
			args: args{
				namespace: "team-a",
				spec: namespacedv1beta1.NamespacedProviderConfigSpec{
					Credentials: namespacedv1beta1.NamespacedProviderCredentials{
						Source: xpv2.CredentialsSourceInjectedIdentity,
					},
				},
			},
			want: namespacedv1beta1.ProviderConfigSpec{
				Credentials: namespacedv1beta1.ProviderCredentials{
					Source: xpv2.CredentialsSourceInjectedIdentity,
				},
			},
		},
		"FsEnvAndScalarFieldsPassThrough": {
			args: args{
				namespace: "team-a",
				spec: namespacedv1beta1.NamespacedProviderConfigSpec{
					APIEndpoint: new("https://api.example.civo.internal"),
					Credentials: namespacedv1beta1.NamespacedProviderCredentials{
						Source: xpv2.CredentialsSourceFilesystem,
						Fs:     &xpv2.FsSelector{Path: "/creds"},
						Env:    &xpv2.EnvSelector{Name: "CIVO_CREDS"},
					},
				},
			},
			want: namespacedv1beta1.ProviderConfigSpec{
				APIEndpoint: new("https://api.example.civo.internal"),
				Credentials: namespacedv1beta1.ProviderCredentials{
					Source: xpv2.CredentialsSourceFilesystem,
					CommonCredentialSelectors: xpv2.CommonCredentialSelectors{
						Fs:  &xpv2.FsSelector{Path: "/creds"},
						Env: &xpv2.EnvSelector{Name: "CIVO_CREDS"},
					},
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := resolveNamespacedSpec(tc.args.spec, tc.args.namespace)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("resolveNamespacedSpec() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestBuildConfiguration(t *testing.T) {
	type args struct {
		credentialsFile string
		region          *string
		apiEndpoint     *string
	}

	cases := map[string]struct {
		args args
		want map[string]any
	}{
		"FileOnly": {
			args: args{credentialsFile: "/tmp/provider-civo/civo-credentials-uid-1.json"},
			want: map[string]any{"credentials_file": "/tmp/provider-civo/civo-credentials-uid-1.json"},
		},
		"FileAndRegion": {
			args: args{credentialsFile: "/tmp/provider-civo/civo-credentials-uid-1.json", region: new("LON1")},
			want: map[string]any{"credentials_file": "/tmp/provider-civo/civo-credentials-uid-1.json", "region": "LON1"},
		},
		"EmptyRegionOmitted": {
			args: args{credentialsFile: "/tmp/provider-civo/civo-credentials-uid-1.json", region: new("")},
			want: map[string]any{"credentials_file": "/tmp/provider-civo/civo-credentials-uid-1.json"},
		},
		"FileAndAPIEndpoint": {
			args: args{credentialsFile: "/tmp/provider-civo/civo-credentials-uid-1.json", apiEndpoint: new("https://api.example.civo.internal")},
			want: map[string]any{"credentials_file": "/tmp/provider-civo/civo-credentials-uid-1.json", "api_endpoint": "https://api.example.civo.internal"},
		},
		"EmptyAPIEndpointOmitted": {
			args: args{credentialsFile: "/tmp/provider-civo/civo-credentials-uid-1.json", apiEndpoint: new("")},
			want: map[string]any{"credentials_file": "/tmp/provider-civo/civo-credentials-uid-1.json"},
		},
		"AllFields": {
			args: args{credentialsFile: "/tmp/provider-civo/civo-credentials-uid-1.json", region: new("NYC1"), apiEndpoint: new("https://api.example.civo.internal")},
			want: map[string]any{"credentials_file": "/tmp/provider-civo/civo-credentials-uid-1.json", "region": "NYC1", "api_endpoint": "https://api.example.civo.internal"},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := buildConfiguration(tc.args.credentialsFile, tc.args.region, tc.args.apiEndpoint)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("buildConfiguration() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// credentialsFiles lists the credentials files in dir.
func credentialsFiles(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("cannot list %s: %v", dir, err)
	}
	var names []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "civo-credentials-") {
			names = append(names, e.Name())
		}
	}
	return names
}

func TestWriteCredentialsFile(t *testing.T) {
	const uid = "0f6e2d1a-6b2c-4a19-9b7e-2f5c1d3e4a5b"

	t.Run("WritesTheUpstreamFormat", func(t *testing.T) {
		dir := t.TempDir()
		path, err := writeCredentialsFile(dir, uid, "civo-token")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if filepath.Dir(path) != dir {
			t.Errorf("file %s is not in %s", path, dir)
		}
		name := filepath.Base(path)
		if !strings.HasPrefix(name, "civo-credentials-"+uid+"-") || !strings.HasSuffix(name, ".json") {
			t.Errorf("unexpected file name %s", name)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("cannot stat %s: %v", path, err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Errorf("file mode is %o, want 600", info.Mode().Perm())
		}
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("cannot read %s: %v", path, err)
		}
		want := `{"apikeys":{"provider-civo":"civo-token"},"meta":{"current_apikey":"provider-civo"}}`
		if diff := cmp.Diff(want, string(content)); diff != "" {
			t.Errorf("file content mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("EveryCallGetsItsOwnFile", func(t *testing.T) {
		dir := t.TempDir()
		first, err := writeCredentialsFile(dir, uid, "civo-token")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		second, err := writeCredentialsFile(dir, uid, "civo-token")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if first == second {
			t.Errorf("both calls returned %s", first)
		}
		if got := credentialsFiles(t, dir); len(got) != 2 {
			t.Errorf("got files %v, want two", got)
		}
	})

	t.Run("EmptyToken", func(t *testing.T) {
		dir := t.TempDir()
		_, err := writeCredentialsFile(dir, uid, "")
		if err == nil || !strings.Contains(err.Error(), `credentials secret has no "token" key`) {
			t.Fatalf("got error %v, want the missing token error", err)
		}
		if got := credentialsFiles(t, dir); len(got) != 0 {
			t.Errorf("got files %v, want none", got)
		}
	})

	t.Run("MissingDirectory", func(t *testing.T) {
		_, err := writeCredentialsFile(filepath.Join(t.TempDir(), "missing"), uid, "civo-token")
		if err == nil || !strings.Contains(err.Error(), "cannot create the credentials file") {
			t.Fatalf("got error %v, want a create error", err)
		}
	})
}

func TestPrepareCredentialsDir(t *testing.T) {
	t.Run("CreatesTheDirectory", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "provider-civo")
		if err := PrepareCredentialsDir(dir); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("cannot stat %s: %v", dir, err)
		}
		if !info.IsDir() || info.Mode().Perm() != 0o700 {
			t.Errorf("got mode %o, want a directory with mode 700", info.Mode().Perm())
		}
	})

	t.Run("RemovesStaleCredentialsFilesOnly", func(t *testing.T) {
		dir := t.TempDir()
		for _, name := range []string{"civo-credentials-uid-1.json", "civo-credentials-uid-2.json", "other.json"} {
			if err := os.WriteFile(filepath.Join(dir, name), []byte("{}"), 0o600); err != nil {
				t.Fatalf("cannot write %s: %v", name, err)
			}
		}
		if err := os.Mkdir(filepath.Join(dir, "civo-credentials-dir.json"), 0o700); err != nil {
			t.Fatalf("cannot create the directory: %v", err)
		}
		if err := PrepareCredentialsDir(dir); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("cannot list %s: %v", dir, err)
		}
		got := make([]string, 0, len(entries))
		for _, e := range entries {
			got = append(got, e.Name())
		}
		want := []string{"civo-credentials-dir.json", "other.json"}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("directory content mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestNodePoolRegion(t *testing.T) {
	type args struct {
		mg resource.Managed
	}

	cases := map[string]struct {
		args args
		want *string
	}{
		"NodePoolRegionScopesTheClient": {
			args: args{
				mg: &clusterkubernetesv1beta1.NodePool{Spec: clusterkubernetesv1beta1.NodePoolSpec{
					ForProvider: clusterkubernetesv1beta1.NodePoolParameters{Region: new("LON1")},
				}},
			},
			want: new("LON1"),
		},
		"NamespacedNodePoolRegionScopesTheClient": {
			args: args{
				mg: &namespacedkubernetesv1beta1.NodePool{Spec: namespacedkubernetesv1beta1.NodePoolSpec{
					ForProvider: namespacedkubernetesv1beta1.NodePoolParameters{Region: new("FRA1")},
				}},
			},
			want: new("FRA1"),
		},
		"NodePoolWithoutRegionLeavesTheClientUnscoped": {
			args: args{mg: &clusterkubernetesv1beta1.NodePool{}},
			want: nil,
		},
		"NodePoolEmptyRegionLeavesTheClientUnscoped": {
			args: args{
				mg: &clusterkubernetesv1beta1.NodePool{Spec: clusterkubernetesv1beta1.NodePoolSpec{
					ForProvider: clusterkubernetesv1beta1.NodePoolParameters{Region: new("")},
				}},
			},
			want: nil,
		},
		"OtherKindsLeaveTheClientUnscoped": {
			args: args{
				mg: &clustervpcv1beta1.Network{Spec: clustervpcv1beta1.NetworkSpec{
					ForProvider: clustervpcv1beta1.NetworkParameters{Region: new("LON1")},
				}},
			},
			want: nil,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := nodePoolRegion(tc.args.mg)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("nodePoolRegion() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// TestConfigure drives the real upstream provider configure function against
// a local HTTP server, with the token handed over through the credentials
// file, so it catches an upstream change to the credentials file parser or to
// how the region and the API endpoint reach the Civo API client.
func TestConfigure(t *testing.T) {
	type request struct {
		Path          string
		Authorization string
	}
	type client struct {
		BaseURL string
		Region  string
		APIKey  string
		Request request
	}

	var seen request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = request{Path: r.URL.Path, Authorization: r.Header.Get("Authorization")}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("[]"))
	}))
	defer server.Close()
	unreachable := httptest.NewServer(http.NotFoundHandler())
	unreachable.Close()

	type args struct {
		token       string
		region      *string
		apiEndpoint *string
	}
	type want struct {
		client      client
		errContains string
	}

	cases := map[string]struct {
		args args
		want want
	}{
		"TokenRegionAndEndpointReachTheClient": {
			args: args{token: "civo-token", region: new("LON1"), apiEndpoint: new(server.URL)},
			want: want{client: client{
				BaseURL: server.URL,
				Region:  "LON1",
				APIKey:  "civo-token",
				Request: request{Path: "/v2/regions", Authorization: "bearer civo-token"},
			}},
		},
		"NoRegionLeavesTheClientUnscoped": {
			args: args{token: "civo-token", apiEndpoint: new(server.URL)},
			want: want{client: client{
				BaseURL: server.URL,
				APIKey:  "civo-token",
				Request: request{Path: "/v2/regions", Authorization: "bearer civo-token"},
			}},
		},
		"UnreachableEndpoint": {
			args: args{token: "civo-token", apiEndpoint: new(unreachable.URL)},
			want: want{errContains: "connecting to Civo's API"},
		},
		"EmptyToken": {
			args: args{apiEndpoint: new(server.URL)},
			want: want{errContains: `credentials secret has no "token" key`},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			for _, key := range []string{"CIVO_TOKEN", "CIVO_REGION", "CIVO_API_URL", "CIVO_CREDENTIAL_FILE"} {
				t.Setenv(key, "")
			}
			seen = request{}
			dir := t.TempDir()
			ps, err := configure(context.Background(), dir, "uid-1", tc.args.token, tc.args.region, tc.args.apiEndpoint)
			if got := credentialsFiles(t, dir); len(got) != 0 {
				t.Errorf("credentials files left behind: %v", got)
			}
			if tc.want.errContains != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.want.errContains)
				}
				if !strings.Contains(err.Error(), tc.want.errContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tc.want.errContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if _, ok := ps.Configuration["token"]; ok {
				t.Errorf("the configuration still carries the token attribute")
			}
			if path, ok := ps.Configuration["credentials_file"].(string); !ok || filepath.Dir(path) != dir {
				t.Errorf("credentials_file is %v, want a file in %s", ps.Configuration["credentials_file"], dir)
			}
			apiClient, ok := ps.Meta.(*civogo.Client)
			if !ok {
				t.Fatalf("provider meta is %T, want *civogo.Client", ps.Meta)
			}
			got := client{BaseURL: apiClient.BaseURL.String(), Region: apiClient.Region, APIKey: apiClient.APIKey, Request: seen}
			if diff := cmp.Diff(tc.want.client, got); diff != "" {
				t.Errorf("configure() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
