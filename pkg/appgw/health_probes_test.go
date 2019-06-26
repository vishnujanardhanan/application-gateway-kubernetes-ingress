// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"fmt"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"

	ptv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressprohibitedtarget/v1"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
)

// appgw_suite_test.go launches these Ginkgo tests

var _ = Describe("configure App Gateway health probes", func() {
	ingressList := []*v1beta1.Ingress{tests.NewIngressFixture()}
	serviceList := []*v1.Service{tests.NewServiceFixture()}

	Context("create probes", func() {
		cb := newConfigBuilderFixture(nil)

		// This is a probe that (we pretend to) have created manually and we expect that with proper configuration
		// ingress controller will not delete it.
		manuallyCreatedProbe := n.ApplicationGatewayProbe{
			Name: to.StringPtr("a-probe-for-a-prohibited-Target"),
			ApplicationGatewayProbePropertiesFormat: &n.ApplicationGatewayProbePropertiesFormat{
				Protocol:                            n.HTTP,
				Host:                                to.StringPtr("www.prohibited.com"),
				Path:                                to.StringPtr("/"),
				Interval:                            to.Int32Ptr(30),
				Timeout:                             to.Int32Ptr(30),
				UnhealthyThreshold:                  to.Int32Ptr(3),
				PickHostNameFromBackendHTTPSettings: nil,
				MinServers:                          nil,
				Match:                               nil,
				ProvisioningState:                   nil,
			},
		}
		cb.appGwConfig.Probes = &[]n.ApplicationGatewayProbe{
			manuallyCreatedProbe,
		}

		endpoints := tests.NewEndpointsFixture()
		_ = cb.k8sContext.Caches.Endpoints.Add(endpoints)

		service := tests.NewServiceFixture(*tests.NewServicePortsFixture()...)
		_ = cb.k8sContext.Caches.Service.Add(service)

		pod := tests.NewPodFixture(tests.ServiceName, tests.Namespace, tests.ContainerName, tests.ContainerPort)
		_ = cb.k8sContext.Caches.Pods.Add(pod)

		// This ProhibitedTarget informs Ingress Controller NOT to mutate/delete settings for the given host/port
		prohibitedTarget1 := ptv1.AzureIngressProhibitedTarget{
			Spec: ptv1.AzureIngressProhibitedTargetSpec{
				Hostname: "www.prohibited.com",
				Port:     80,
			},
		}

		cbCtx := ConfigBuilderContext{
			IngressList: ingressList,
			ServiceList: serviceList,
			ProhibitedTargets: []*ptv1.AzureIngressProhibitedTarget{
				&prohibitedTarget1,
			},
		}

		// !! Action !!
		_ = cb.HealthProbesCollection(&cbCtx)

		actual := cb.appGwConfig.Probes

		// We expect our health probe configurator to have arrived at this final setup
		defaultProbe := n.ApplicationGatewayProbe{

			ApplicationGatewayProbePropertiesFormat: &n.ApplicationGatewayProbePropertiesFormat{
				Protocol:                            n.HTTP,
				Host:                                to.StringPtr("localhost"),
				Path:                                to.StringPtr("/"),
				Interval:                            to.Int32Ptr(30),
				Timeout:                             to.Int32Ptr(30),
				UnhealthyThreshold:                  to.Int32Ptr(3),
				PickHostNameFromBackendHTTPSettings: nil,
				MinServers:                          nil,
				Match:                               nil,
				ProvisioningState:                   nil,
			},
			Name: to.StringPtr(agPrefix + "defaultprobe"),
			Etag: nil,
			Type: nil,
			ID:   nil,
		}
		probeForHost := n.ApplicationGatewayProbe{
			ApplicationGatewayProbePropertiesFormat: &n.ApplicationGatewayProbePropertiesFormat{
				Protocol:                            n.HTTP,
				Host:                                to.StringPtr(tests.Host),
				Path:                                to.StringPtr(tests.URLPath),
				Interval:                            to.Int32Ptr(30),
				Timeout:                             to.Int32Ptr(30),
				UnhealthyThreshold:                  to.Int32Ptr(3),
				PickHostNameFromBackendHTTPSettings: nil,
				MinServers:                          nil,
				Match:                               nil,
				ProvisioningState:                   nil,
			},
			Name: to.StringPtr(agPrefix + "pb-" + tests.Namespace + "-" + tests.ServiceName + "-443---name--"),
			Etag: nil,
			Type: nil,
			ID:   nil,
		}

		probeForOtherHost := n.ApplicationGatewayProbe{
			ApplicationGatewayProbePropertiesFormat: &n.ApplicationGatewayProbePropertiesFormat{
				Protocol:                            n.HTTP,
				Host:                                to.StringPtr(tests.Host),
				Path:                                to.StringPtr(tests.URLPath),
				Interval:                            to.Int32Ptr(20),
				Timeout:                             to.Int32Ptr(5),
				UnhealthyThreshold:                  to.Int32Ptr(3),
				PickHostNameFromBackendHTTPSettings: nil,
				MinServers:                          nil,
				Match:                               nil,
				ProvisioningState:                   nil,
			},
			Name: to.StringPtr(agPrefix + "pb-" + tests.Namespace + "-" + tests.ServiceName + "-80---name--"),
			Etag: nil,
			Type: nil,
			ID:   nil,
		}

		It("should have exactly 3 records", func() {
			Expect(len(*actual)).To(Equal(4))
		})

		It("should have created 1 default probe", func() {
			Expect(*actual).To(ContainElement(defaultProbe))
		})

		It("should have created 1 probe for Host", func() {
			Expect(*actual).To(ContainElement(probeForHost))
		})

		It("should have created 1 probe for OtherHost", func() {
			Expect(*actual).To(ContainElement(probeForOtherHost))
		})

		It("should have kept the 1 probe that was manually created", func() {
			Expect(*actual).To(ContainElement(manuallyCreatedProbe))
		})
	})

	Context("use default probe when service doesn't exists", func() {
		cb := newConfigBuilderFixture(nil)

		pod := tests.NewPodFixture(tests.ServiceName, tests.Namespace, tests.ContainerName, tests.ContainerPort)
		_ = cb.k8sContext.Caches.Pods.Add(pod)

		cbCtx := &ConfigBuilderContext{
			IngressList: ingressList,
			ServiceList: serviceList,
		}

		// !! Action !!
		_ = cb.HealthProbesCollection(cbCtx)
		actual := cb.appGwConfig.Probes

		// We expect our health probe configurator to have arrived at this final setup
		defaultProbe := n.ApplicationGatewayProbe{

			ApplicationGatewayProbePropertiesFormat: &n.ApplicationGatewayProbePropertiesFormat{
				Protocol:                            n.HTTP,
				Host:                                to.StringPtr("localhost"),
				Path:                                to.StringPtr("/"),
				Interval:                            to.Int32Ptr(30),
				Timeout:                             to.Int32Ptr(30),
				UnhealthyThreshold:                  to.Int32Ptr(3),
				PickHostNameFromBackendHTTPSettings: nil,
				MinServers:                          nil,
				Match:                               nil,
				ProvisioningState:                   nil,
			},
			Name: to.StringPtr(agPrefix + "defaultprobe"),
			Etag: nil,
			Type: nil,
			ID:   nil,
		}

		It("should have exactly 1 record", func() {
			Expect(len(*actual)).To(Equal(1), fmt.Sprintf("Actual probes: %+v", *actual))
		})

		It("should have created 1 default probe", func() {
			Expect(*actual).To(ContainElement(defaultProbe))
		})
	})
})
