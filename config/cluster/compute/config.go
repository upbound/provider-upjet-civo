// SPDX-FileCopyrightText: 2026 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: CC0-1.0

package compute

import (
	"github.com/crossplane/upjet/v2/pkg/config"
)

// Configure configures the compute group
func Configure(p *config.Provider) {
	p.AddResourceConfigurator("civo_instance", func(r *config.Resource) {
		r.References["firewall_id"] = config.Reference{
			TerraformName: "civo_vpc_firewall",
		}
		r.References["network_id"] = config.Reference{
			TerraformName: "civo_vpc_network",
		}
		r.References["reserved_ipv4"] = config.Reference{
			TerraformName: "civo_vpc_reserved_ip",
		}
		r.References["sshkey_id"] = config.Reference{
			TerraformName: "civo_ssh_key",
		}
	})
}
