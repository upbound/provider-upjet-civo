// SPDX-FileCopyrightText: 2026 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: CC0-1.0

package namespaced

import (
	"github.com/upbound/provider-civo/config/namespaced/dns"
	"github.com/upbound/provider-civo/config/namespaced/vpc"
)

func init() {
	ProviderConfiguration.AddConfig(dns.Configure)
	ProviderConfiguration.AddConfig(vpc.Configure)
}
