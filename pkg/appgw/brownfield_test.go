// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	mtv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressmanagedtarget/v1"
	ptv1 "github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis/azureingressprohibitedtarget/v1"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/brownfield"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/tests"
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// appgw_suite_test.go launches these Ginkgo tests

var _ = Describe("test blacklist/whitelist health probes", func() {
	prohibitedTarget1 := ptv1.AzureIngressProhibitedTarget{
		Spec: ptv1.AzureIngressProhibitedTargetSpec{
			IP:       "123",
			Hostname: tests.Host,
			Port:     443,
			Paths: []string{
				"/fox",
				"/bar",
			},
		},
	}

	managedtarget1 := mtv1.AzureIngressManagedTarget{
		Spec: mtv1.AzureIngressManagedTargetSpec{
			IP:       "123",
			Hostname: tests.Host,
			Port:     443,
			Paths: []string{
				"/foo",
				"/bar",
				"/baz",
			},
		},
	}

	krProhibited := ConfigBuilderContext{
		ProhibitedTargets: []*ptv1.AzureIngressProhibitedTarget{
			&prohibitedTarget1,
		},
	}

	Context("test permit/prohibit paths", func() {
		actual := normalizePath("*//*hello/**/*//")
		It("should have exactly 1 record", func() {
			Expect(actual).To(Equal("*//*hello"))
		})
	})

	Context("test getManagedTargetList", func() {
		kr := ConfigBuilderContext{
			ManagedTargets: []*mtv1.AzureIngressManagedTarget{
				&managedtarget1,
			},
		}
		actual := getManagedTargetList(&kr)
		It("should have produced correct Target list", func() {
			Expect(len(*actual)).To(Equal(3))
			{
				expected := brownfield.Target{
					Host: tests.Host,
					Port: 443,
					Path: to.StringPtr("/foo"),
				}
				Expect(*actual).To(ContainElement(expected))
			}
			{
				expected := brownfield.Target{
					Host: tests.Host,
					Port: 443,
					Path: to.StringPtr("/bar"),
				}
				Expect(*actual).To(ContainElement(expected))
			}
			{
				expected := brownfield.Target{
					Host: tests.Host,
					Port: 443,
					Path: to.StringPtr("/baz"),
				}
				Expect(*actual).To(ContainElement(expected))
			}
		})
	})

	Context("test getProhibitedTargetList", func() {
		actual := getProhibitedTargetList(&krProhibited)
		It("should have produced correct Target list", func() {
			Expect(len(*actual)).To(Equal(2))
			{
				expected := brownfield.Target{
					Host: tests.Host,
					Port: 443,
					Path: to.StringPtr("/fox"),
				}
				Expect(*actual).To(ContainElement(expected))
			}
			{
				expected := brownfield.Target{
					Host: tests.Host,
					Port: 443,
					Path: to.StringPtr("/bar"),
				}
				Expect(*actual).To(ContainElement(expected))
			}
		})
	})

	Context("test inProbeList", func() {
		kr := ConfigBuilderContext{
			ProhibitedTargets: []*ptv1.AzureIngressProhibitedTarget{
				&prohibitedTarget1,
			},
			ManagedTargets: []*mtv1.AzureIngressManagedTarget{
				&managedtarget1,
			},
		}

		{
			probe := tests.GetApplicationGatewayProbe(nil, nil)
			actual := inProbeList(&probe, getProhibitedTargetList(&kr))
			It("should be able to find probe in prohibited Target list", func() {
				Expect(actual).To(BeFalse())
			})
		}
		{
			probe := tests.GetApplicationGatewayProbe(nil, nil)
			actual := inProbeList(&probe, getManagedTargetList(&kr))
			It("should be able to find probe in managed Target list", func() {
				Expect(actual).To(BeTrue())
			})
		}
	})

	Context("test getManagedProbes", func() {
		kr := ConfigBuilderContext{
			ProhibitedTargets: []*ptv1.AzureIngressProhibitedTarget{
				&prohibitedTarget1,
			},
			ManagedTargets: []*mtv1.AzureIngressManagedTarget{
				&managedtarget1,
			},
		}

		whiteListedProbe := tests.GetApplicationGatewayProbe(nil, to.StringPtr("/baz")) // whitelisted
		probes := []n.ApplicationGatewayProbe{
			tests.GetApplicationGatewayProbe(nil, to.StringPtr("/fox")), // blacklisted
			tests.GetApplicationGatewayProbe(nil, to.StringPtr("/bar")), // blacklisted
			whiteListedProbe,
		}
		actual := getManagedProbes(probes, &kr)
		It("should have filtered probes based on black/white list", func() {
			Expect(len(actual)).To(Equal(1))
			Expect(actual).To(ContainElement(whiteListedProbe))
		})

	})
})
