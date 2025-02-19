// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package azure

import (
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestUtils(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Azure Suite")
}

var _ = Describe("Azure", func() {
	Describe("Testing `azure` helpers", func() {

		Context("ensure ParseResourceID works as expected", func() {
			It("should parse appgw resourceId correctly", func() {
				subID := SubscriptionID("xxxx")
				resGp := ResourceGroup("yyyy")
				resName := ResourceName("zzzz")
				resourceID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/publicIPAddresses/%s", subID, resGp, resName)
				outSubID, outResGp, outResName := ParseResourceID(resourceID)
				Expect(outSubID).To(Equal(subID))
				Expect(resGp).To(Equal(outResGp))
				Expect(resName).To(Equal(outResName))
			})
		})

		Context("ensure ConvertToClusterResourceGroup works as expected", func() {
			It("should parse empty infra resourse group correctly", func() {
				subID := SubscriptionID("xxxx")
				resGp := ResourceGroup("")
				_, err := ConvertToClusterResourceGroup(subID, resGp, nil)
				Ω(err).To(HaveOccurred(), "this call should have failed in parsing the resource group")
			})
			It("should parse valid infra resourse group correctly", func() {
				subID := SubscriptionID("xxxx")
				resGp := ResourceGroup("MC_resgp_resName_location")
				Expect(ConvertToClusterResourceGroup(subID, resGp, nil)).To(Equal("/subscriptions/xxxx/resourcegroups/resgp/providers/Microsoft.ContainerService/managedClusters/resName"))

				subID = SubscriptionID("xxxx")
				resGp = ResourceGroup("mc_resgp_resName_location")
				Expect(ConvertToClusterResourceGroup(subID, resGp, nil)).To(Equal("/subscriptions/xxxx/resourcegroups/resgp/providers/Microsoft.ContainerService/managedClusters/resName"))
			})
		})
	})
})
