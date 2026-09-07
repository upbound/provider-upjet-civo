// SPDX-FileCopyrightText: 2026 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: CC0-1.0

package kubernetes

import (
	"github.com/crossplane/upjet/v2/pkg/config"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Configure configures the kubernetes group
func Configure(p *config.Provider) {
	p.AddResourceConfigurator("civo_kubernetes_cluster", func(r *config.Resource) {
		r.References["firewall_id"] = config.Reference{
			TerraformName: "civo_vpc_firewall",
		}
		r.References["network_id"] = config.Reference{
			TerraformName: "civo_vpc_network",
		}
		// Publish the kubeconfig under the key provider-kubernetes expects in
		// addition to upjet's attribute.kubeconfig. Upstream only fills the
		// attribute when write_kubeconfig is true.
		r.Sensitive.AdditionalConnectionDetailsFn = func(attr map[string]any) (map[string][]byte, error) {
			kubeconfig, ok := attr["kubeconfig"].(string)
			if !ok || kubeconfig == "" {
				return nil, nil
			}
			return map[string][]byte{"kubeconfig": []byte(kubeconfig)}, nil
		}
	})

	p.AddResourceConfigurator("civo_kubernetes_node_pool", func(r *config.Resource) {
		r.References["cluster_id"] = config.Reference{
			TerraformName: "civo_kubernetes_cluster",
		}
		// The upstream resource has no region attribute: its create, update
		// and delete fall back to the provider-level region and its read uses
		// the provider client as is, so a pool outside the account's default
		// region cannot be managed through the Terraform schema alone. This
		// synthetic field is never sent to Terraform; the provider's SetupFn
		// reads it and scopes the API client to that region for every
		// reconcile of the resource.
		r.TerraformResource.Schema["region"] = &schema.Schema{
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "The region of the cluster the node pool belongs to. Defaults to the account's default region. It scopes every API call made for this resource, because the upstream resource has no region attribute of its own.",
		}
		r.TerraformConfigurationInjector = func(_ map[string]any, params map[string]any) error {
			delete(params, "region")
			return nil
		}
	})
}
