// SPDX-FileCopyrightText: 2026 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: CC0-1.0

package dns

import (
	"github.com/crossplane/upjet/v2/pkg/config"
)

// Configure configures the dns group
func Configure(p *config.Provider) {
	p.AddResourceConfigurator("civo_dns_domain_record", func(r *config.Resource) {
		r.References["domain_id"] = config.Reference{
			TerraformName: "civo_dns_domain_name",
		}
	})
}
