// SPDX-FileCopyrightText: 2026 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/crossplane/upjet/v2/pkg/controller"

	domain "github.com/upbound/provider-civo/internal/controller/cluster/dns/domain"
	record "github.com/upbound/provider-civo/internal/controller/cluster/dns/record"
	providerconfig "github.com/upbound/provider-civo/internal/controller/cluster/providerconfig"
	firewall "github.com/upbound/provider-civo/internal/controller/cluster/vpc/firewall"
	network "github.com/upbound/provider-civo/internal/controller/cluster/vpc/network"
	reservedip "github.com/upbound/provider-civo/internal/controller/cluster/vpc/reservedip"
	reservedipassignment "github.com/upbound/provider-civo/internal/controller/cluster/vpc/reservedipassignment"
	subnet "github.com/upbound/provider-civo/internal/controller/cluster/vpc/subnet"
)

// Setup creates all controllers with the supplied logger and adds them to
// the supplied manager.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	for _, setup := range []func(ctrl.Manager, controller.Options) error{
		domain.Setup,
		record.Setup,
		providerconfig.Setup,
		firewall.Setup,
		network.Setup,
		reservedip.Setup,
		reservedipassignment.Setup,
		subnet.Setup,
	} {
		if err := setup(mgr, o); err != nil {
			return err
		}
	}
	return nil
}

// SetupGated creates all controllers with the supplied logger and adds them to
// the supplied manager gated.
func SetupGated(mgr ctrl.Manager, o controller.Options) error {
	for _, setup := range []func(ctrl.Manager, controller.Options) error{
		domain.SetupGated,
		record.SetupGated,
		providerconfig.SetupGated,
		firewall.SetupGated,
		network.SetupGated,
		reservedip.SetupGated,
		reservedipassignment.SetupGated,
		subnet.SetupGated,
	} {
		if err := setup(mgr, o); err != nil {
			return err
		}
	}
	return nil
}

// SetupWebhookWithManager registers conversion webhooks for all resource kinds in the group.
func SetupWebhookWithManager(mgr ctrl.Manager) error {
	for _, setup := range []func(ctrl.Manager) error{
		domain.SetupWebhookWithManager,
		record.SetupWebhookWithManager,
		providerconfig.SetupWebhookWithManager,
		firewall.SetupWebhookWithManager,
		network.SetupWebhookWithManager,
		reservedip.SetupWebhookWithManager,
		reservedipassignment.SetupWebhookWithManager,
		subnet.SetupWebhookWithManager,
	} {
		if err := setup(mgr); err != nil {
			return err
		}
	}
	return nil
}
