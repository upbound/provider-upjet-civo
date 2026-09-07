// SPDX-FileCopyrightText: 2026 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: CC0-1.0

package vpc

import (
	"github.com/crossplane/upjet/v2/pkg/config"
)

// Configure configures the vpc group
func Configure(p *config.Provider) {
	p.AddResourceConfigurator("civo_vpc_firewall", func(r *config.Resource) {
		r.References["network_id"] = config.Reference{
			TerraformName: "civo_vpc_network",
		}
		// The API reports the rules it creates for create_default_rules (and
		// the ones Civo adds for attached clusters) back on read. Copying them
		// into the spec would trip the upstream CustomizeDiff, which rejects
		// create_default_rules together with explicit rules, and block every
		// later observe including the one before delete.
		r.LateInitializer = config.LateInitializer{
			IgnoredFields: []string{"ingress_rule", "egress_rule"},
		}
	})

	p.AddResourceConfigurator("civo_vpc_reserved_ip_assignment", func(r *config.Resource) {
		r.References["instance_id"] = config.Reference{
			TerraformName: "civo_instance",
		}
		r.References["reserved_ip_id"] = config.Reference{
			TerraformName: "civo_vpc_reserved_ip",
		}
	})

	p.AddResourceConfigurator("civo_vpc_subnet", func(r *config.Resource) {
		r.References["network_id"] = config.Reference{
			TerraformName: "civo_vpc_network",
		}
	})
}
