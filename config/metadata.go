// SPDX-FileCopyrightText: 2026 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: CC0-1.0

package config

import (
	"github.com/crossplane/crossplane-runtime/v2/pkg/errors"
	ujconfig "github.com/crossplane/upjet/v2/pkg/config"
	"github.com/crossplane/upjet/v2/pkg/registry"
)

// legacyMetadataNames maps the civo_vpc_* resource names, which upstream
// registers as aliases of the same schema.Resource, to the legacy names whose
// registry doc pages carry the description, argument docs and examples. The
// alias doc pages have no example section, so the upjet scraper keys them by
// page title and NewProvider attaches no metadata to them.
var legacyMetadataNames = map[string]string{
	"civo_vpc_firewall":               "civo_firewall",
	"civo_vpc_network":                "civo_network",
	"civo_vpc_reserved_ip":            "civo_reserved_ip",
	"civo_vpc_reserved_ip_assignment": "civo_instance_reserved_ip_assignment",
}

// attachLegacyMetadata gives every generated civo_vpc_* resource the registry
// metadata scraped for its legacy name, so CRD descriptions, field docs and
// generated examples do not come out empty for the modern names.
func attachLegacyMetadata(pc *ujconfig.Provider) error {
	meta, err := registry.NewProviderMetadataFromFile([]byte(providerMetadata))
	if err != nil {
		return errors.Wrap(err, "cannot parse provider metadata")
	}
	for name, legacy := range legacyMetadataNames {
		r, ok := pc.Resources[name]
		if !ok || r.MetaResource != nil {
			continue
		}
		m, ok := meta.Resources[legacy]
		if !ok {
			return errors.Errorf("no scraped metadata for %s, needed by %s", legacy, name)
		}
		r.MetaResource = m
	}
	return nil
}
