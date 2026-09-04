// SPDX-FileCopyrightText: 2026 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: CC0-1.0

package namespaced

import (
	"github.com/upbound/provider-civo/config/namespaced/compute"
	"github.com/upbound/provider-civo/config/namespaced/database"
	"github.com/upbound/provider-civo/config/namespaced/dns"
	"github.com/upbound/provider-civo/config/namespaced/kubernetes"
	"github.com/upbound/provider-civo/config/namespaced/storage"
	"github.com/upbound/provider-civo/config/namespaced/vpc"
)

func init() {
	ProviderConfiguration.AddConfig(compute.Configure)
	ProviderConfiguration.AddConfig(database.Configure)
	ProviderConfiguration.AddConfig(dns.Configure)
	ProviderConfiguration.AddConfig(kubernetes.Configure)
	ProviderConfiguration.AddConfig(storage.Configure)
	ProviderConfiguration.AddConfig(vpc.Configure)
}
