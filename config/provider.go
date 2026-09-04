// SPDX-FileCopyrightText: 2026 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: CC0-1.0

package config

import (
	// Note(turkenh): we are importing this to embed provider schema document
	"context"
	_ "embed"

	"github.com/civo/terraform-provider-civo/civo"
	ujconfig "github.com/crossplane/upjet/v2/pkg/config"

	"github.com/upbound/provider-civo/config/cluster"
	"github.com/upbound/provider-civo/config/templates"
)

const (
	resourcePrefix = "civo"
	modulePath     = "github.com/upbound/provider-civo"
	versionV1Beta1 = "v1beta1"
)

//go:embed schema.json
var providerSchema string

//go:embed provider-metadata.yaml
var providerMetadata string

// GetProvider returns provider configuration
func GetProvider(_ context.Context) (*ujconfig.Provider, error) {
	sdkProvider := civo.Provider()

	defaultResourceOptions := []ujconfig.ResourceOption{
		GroupKindOverrides(),
		ExternalNameConfigurations(),
	}

	pc := ujconfig.NewProvider(
		[]byte(providerSchema), resourcePrefix, modulePath, []byte(providerMetadata),
		ujconfig.WithRootGroup("civo.upbound.io"),
		ujconfig.WithIncludeList([]string{}),
		ujconfig.WithControllerTemplate(templates.ControllerTemplate),
		ujconfig.WithTerraformPluginSDKIncludeList(ExternalNameConfigured()),
		ujconfig.WithFeaturesPackage("internal/features"),
		ujconfig.WithTerraformProvider(sdkProvider),
		ujconfig.WithSchemaTraversers(&ujconfig.SingletonListEmbedder{}),
		ujconfig.WithDefaultResourceOptions(defaultResourceOptions...),
	)

	// add custom config functions
	for _, configure := range cluster.ProviderConfiguration {
		configure(pc)
	}

	pc.ConfigureResources()

	registerTFConversions(pc)

	return pc, nil
}
