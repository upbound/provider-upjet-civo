// SPDX-FileCopyrightText: 2026 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: CC0-1.0

package config

import (
	"strings"

	"github.com/crossplane/upjet/v2/pkg/config"
	"github.com/crossplane/upjet/v2/pkg/types/name"
)

// GroupKindOverrides overrides the group and kind of the resource if it matches
// any entry in the GroupMap.
func GroupKindOverrides() config.ResourceOption {
	return func(r *config.Resource) {
		if f, ok := GroupMap[r.Name]; ok {
			r.ShortGroup, r.Kind = f(r.Name)
		}
	}
}

// GroupKindCalculator returns the correct group and kind name for given TF
// resource.
type GroupKindCalculator func(resource string) (string, string)

// ReplaceGroupWords uses given group as the group of the resource and removes
// a number of words in resource name before calculating the kind of the resource.
func ReplaceGroupWords(group string, count int) GroupKindCalculator {
	return func(resource string) (string, string) {
		// "civo_kubernetes_cluster" -> (kubernetes, Cluster)
		words := strings.Split(strings.TrimPrefix(resource, "civo_"), "_")
		snakeKind := strings.Join(words[count:], "_")
		return group, name.NewFromSnake(snakeKind).Camel
	}
}

// KnownGroupKind returns a GroupKindCalculator that assigns the given static
// group and kind regardless of the resource name.
func KnownGroupKind(group, kind string) GroupKindCalculator {
	return func(string) (string, string) { return group, kind }
}

// GroupMap assigns Civo resources to API groups and kinds. Resources are
// added here together with their entry in ExternalNameConfigs.
var GroupMap = map[string]GroupKindCalculator{
	// dns
	"civo_dns_domain_name":   KnownGroupKind("dns", "Domain"),
	"civo_dns_domain_record": KnownGroupKind("dns", "Record"),

	// vpc
	"civo_vpc_firewall":               KnownGroupKind("vpc", "Firewall"),
	"civo_vpc_network":                KnownGroupKind("vpc", "Network"),
	"civo_vpc_reserved_ip":            KnownGroupKind("vpc", "ReservedIP"),
	"civo_vpc_reserved_ip_assignment": KnownGroupKind("vpc", "ReservedIPAssignment"),
	"civo_vpc_subnet":                 KnownGroupKind("vpc", "Subnet"),
}
