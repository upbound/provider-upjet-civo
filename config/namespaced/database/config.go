// SPDX-FileCopyrightText: 2026 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: CC0-1.0

package database

import (
	"github.com/crossplane/upjet/v2/pkg/config"
)

// Configure configures the database group
func Configure(p *config.Provider) {
	p.AddResourceConfigurator("civo_database", func(r *config.Resource) {
		// Upstream leaves the generated password unmarked; keep it out of
		// status and publish it as a connection detail instead.
		r.TerraformResource.Schema["password"].Sensitive = true
		r.References["firewall_id"] = config.Reference{
			TerraformName: "civo_vpc_firewall",
		}
		r.References["network_id"] = config.Reference{
			TerraformName: "civo_vpc_network",
		}
	})
}
