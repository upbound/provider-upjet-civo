package clients

import (
	"context"
	"net/http"
	"net/http/httptest"
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

	"github.com/crossplane/upjet/v2/pkg/terraform"
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
		creds       map[string]string
		region      *string
		apiEndpoint *string
	}

	cases := map[string]struct {
		args            args
		want            map[string]any
		wantErrContains string
	}{
		"TokenOnly": {
			args: args{creds: map[string]string{"token": "civo-token"}},
			want: map[string]any{"token": "civo-token"},
		},
		"TokenAndRegion": {
			args: args{creds: map[string]string{"token": "civo-token"}, region: new("LON1")},
			want: map[string]any{"token": "civo-token", "region": "LON1"},
		},
		"EmptyRegionOmitted": {
			args: args{creds: map[string]string{"token": "civo-token"}, region: new("")},
			want: map[string]any{"token": "civo-token"},
		},
		"TokenAndAPIEndpoint": {
			args: args{creds: map[string]string{"token": "civo-token"}, apiEndpoint: new("https://api.example.civo.internal")},
			want: map[string]any{"token": "civo-token", "api_endpoint": "https://api.example.civo.internal"},
		},
		"EmptyAPIEndpointOmitted": {
			args: args{creds: map[string]string{"token": "civo-token"}, apiEndpoint: new("")},
			want: map[string]any{"token": "civo-token"},
		},
		"AllFields": {
			args: args{creds: map[string]string{"token": "civo-token"}, region: new("NYC1"), apiEndpoint: new("https://api.example.civo.internal")},
			want: map[string]any{"token": "civo-token", "region": "NYC1", "api_endpoint": "https://api.example.civo.internal"},
		},
		"ExtraKeysIgnored": {
			args: args{creds: map[string]string{"token": "civo-token", "unrelated": "x"}},
			want: map[string]any{"token": "civo-token"},
		},
		"EmptyToken": {
			args:            args{creds: map[string]string{"token": ""}},
			wantErrContains: `credentials secret has no "token" key`,
		},
		"MissingToken": {
			args:            args{creds: map[string]string{}},
			wantErrContains: `credentials secret has no "token" key`,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := buildConfiguration(tc.args.creds, tc.args.region, tc.args.apiEndpoint)
			if tc.wantErrContains != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.wantErrContains)
				}
				if !strings.Contains(err.Error(), tc.wantErrContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tc.wantErrContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("buildConfiguration() mismatch (-want +got):\n%s", diff)
			}
		})
	}
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

// TestConfigureProviderMeta drives the real upstream provider configure
// function against a local HTTP server, so it catches an upstream change to
// how the token, the region or the API endpoint reach the Civo API client.
func TestConfigureProviderMeta(t *testing.T) {
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
		configuration map[string]any
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
			args: args{configuration: map[string]any{"token": "civo-token", "region": "LON1", "api_endpoint": server.URL}},
			want: want{client: client{
				BaseURL: server.URL,
				Region:  "LON1",
				APIKey:  "civo-token",
				Request: request{Path: "/v2/regions", Authorization: "bearer civo-token"},
			}},
		},
		"NoRegionLeavesTheClientUnscoped": {
			args: args{configuration: map[string]any{"token": "civo-token", "api_endpoint": server.URL}},
			want: want{client: client{
				BaseURL: server.URL,
				APIKey:  "civo-token",
				Request: request{Path: "/v2/regions", Authorization: "bearer civo-token"},
			}},
		},
		"UnreachableEndpoint": {
			args: args{configuration: map[string]any{"token": "civo-token", "api_endpoint": unreachable.URL}},
			want: want{errContains: "connecting to Civo's API"},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			for _, key := range []string{"CIVO_TOKEN", "CIVO_REGION", "CIVO_API_URL", "CIVO_CREDENTIAL_FILE"} {
				t.Setenv(key, "")
			}
			seen = request{}
			ps := terraform.Setup{Configuration: tc.args.configuration}
			err := configureProviderMeta(context.Background(), &ps)
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
			apiClient, ok := ps.Meta.(*civogo.Client)
			if !ok {
				t.Fatalf("provider meta is %T, want *civogo.Client", ps.Meta)
			}
			got := client{BaseURL: apiClient.BaseURL.String(), Region: apiClient.Region, APIKey: apiClient.APIKey, Request: seen}
			if diff := cmp.Diff(tc.want.client, got); diff != "" {
				t.Errorf("configureProviderMeta() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
