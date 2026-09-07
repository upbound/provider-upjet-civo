// SPDX-FileCopyrightText: 2026 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: CC0-1.0

package storage

import (
	"github.com/crossplane/upjet/v2/pkg/config"
)

// Configure configures the storage group
func Configure(p *config.Provider) {
	p.AddResourceConfigurator("civo_object_store", func(r *config.Resource) {
		r.References["access_key_id"] = config.Reference{
			TerraformName: "civo_object_store_credential",
			Extractor:     `github.com/crossplane/upjet/v2/pkg/resource.ExtractParamPath("access_key_id",true)`,
		}
	})

	p.AddResourceConfigurator("civo_object_store_credential", func(r *config.Resource) {
		// Upstream leaves the generated secret unmarked; keep it out of
		// status and publish it as a connection detail instead.
		r.TerraformResource.Schema["secret_access_key"].Sensitive = true
	})

	p.AddResourceConfigurator("civo_volume", func(r *config.Resource) {
		r.References["network_id"] = config.Reference{
			TerraformName: "civo_vpc_network",
		}
	})

	p.AddResourceConfigurator("civo_volume_attachment", func(r *config.Resource) {
		r.References["instance_id"] = config.Reference{
			TerraformName: "civo_instance",
		}
		r.References["volume_id"] = config.Reference{
			TerraformName: "civo_volume",
		}
	})
}
