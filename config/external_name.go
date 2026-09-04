package config

import (
	"github.com/crossplane/upjet/v2/pkg/config"
)

// ExternalNameConfigs contains all external name configurations for this
// provider.
var ExternalNameConfigs = map[string]config.ExternalName{
	// compute
	"civo_instance": config.IdentifierFromProvider,
	"civo_ssh_key":  config.IdentifierFromProvider,

	// database
	"civo_database": config.IdentifierFromProvider,

	// dns
	"civo_dns_domain_name":   config.IdentifierFromProvider,
	"civo_dns_domain_record": config.IdentifierFromProvider,

	// kubernetes
	"civo_kubernetes_cluster":   config.IdentifierFromProvider,
	"civo_kubernetes_node_pool": config.IdentifierFromProvider,

	// vpc
	"civo_vpc_firewall":               config.IdentifierFromProvider,
	"civo_vpc_network":                config.IdentifierFromProvider,
	"civo_vpc_reserved_ip":            config.IdentifierFromProvider,
	"civo_vpc_reserved_ip_assignment": config.IdentifierFromProvider,
	"civo_vpc_subnet":                 config.IdentifierFromProvider,

	// storage
	"civo_object_store":            config.IdentifierFromProvider,
	"civo_object_store_credential": config.IdentifierFromProvider,
	"civo_volume":                  config.IdentifierFromProvider,
	"civo_volume_attachment":       config.IdentifierFromProvider,
}

// ExternalNameConfigurations applies all external name configs listed in the
// table ExternalNameConfigs and sets the version of those resources to v1beta1
// assuming they will be tested.
func ExternalNameConfigurations() config.ResourceOption {
	return func(r *config.Resource) {
		e, configured := ExternalNameConfigs[r.Name]
		if !configured {
			return
		}
		r.ExternalName = e
		r.Version = versionV1Beta1
	}
}

// ExternalNameConfigured returns the list of all resources whose external name
// is configured manually.
func ExternalNameConfigured() []string {
	l := make([]string, len(ExternalNameConfigs))
	i := 0
	for name := range ExternalNameConfigs {
		// $ is added to match the exact string since the format is regex.
		l[i] = name + "$"
		i++
	}
	return l
}
