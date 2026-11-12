// SPDX-FileCopyrightText: 2026 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: CC0-1.0

package kubernetes

import (
	"github.com/crossplane/upjet/v2/pkg/config"
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
	})
}
