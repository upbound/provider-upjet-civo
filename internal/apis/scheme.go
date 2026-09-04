// SPDX-FileCopyrightText: 2026 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: Apache-2.0

// Package apis implements the scheme-based managed resource resolver used by
// the transformed cross-resource reference resolvers. Resolving source
// resources through a runtime scheme instead of static types removes the
// cross API-group imports that would otherwise form import cycles.
package apis

import (
	xpresource "github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var s = runtime.NewScheme()

// GetManagedResource returns a new managed resource and its list type with
// the given API group, version, kind and list kind, initialized via the
// runtime scheme the provider's APIs are registered with.
func GetManagedResource(group, version, kind, listKind string) (xpresource.Managed, xpresource.ManagedList, error) {
	gv := schema.GroupVersion{
		Group:   group,
		Version: version,
	}
	kindGVK := gv.WithKind(kind)
	m, err := s.New(kindGVK)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to get a new API object of GVK %q from the runtime scheme", kindGVK)
	}

	listGVK := gv.WithKind(listKind)
	l, err := s.New(listGVK)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to get a new API object list of GVK %q from the runtime scheme", listGVK)
	}
	return m.(xpresource.Managed), l.(xpresource.ManagedList), nil
}

// BuildScheme registers the GVKs of the given scheme builder with the runtime
// scheme used to resolve managed resources.
func BuildScheme(sb runtime.SchemeBuilder) error {
	return errors.Wrap(sb.AddToScheme(s), "failed to register the GVKs with the runtime scheme")
}
