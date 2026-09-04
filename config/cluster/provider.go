// SPDX-FileCopyrightText: 2026 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: CC0-1.0

package cluster

import (
	"github.com/upbound/provider-civo/config/cluster/compute"
	"github.com/upbound/provider-civo/config/cluster/dns"
	"github.com/upbound/provider-civo/config/cluster/storage"
	"github.com/upbound/provider-civo/config/cluster/vpc"
)

func init() {
	ProviderConfiguration.AddConfig(compute.Configure)
	ProviderConfiguration.AddConfig(dns.Configure)
	ProviderConfiguration.AddConfig(storage.Configure)
	ProviderConfiguration.AddConfig(vpc.Configure)
}
