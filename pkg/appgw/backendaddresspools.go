// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"fmt"
	"sort"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/brownfield"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/events"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/sorter"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"
)

func (c *appGwConfigBuilder) BackendAddressPools(cbCtx *ConfigBuilderContext) error {
	defaultPool := defaultBackendAddressPool()
	addressPools := map[string]*n.ApplicationGatewayBackendAddressPool{
		*defaultPool.Name: defaultPool,
	}

	_, _, serviceBackendPairMap, _ := c.getBackendsAndSettingsMap(cbCtx)

	for backendID, serviceBackendPair := range serviceBackendPairMap {
		if pool := c.getBackendAddressPool(backendID, serviceBackendPair, addressPools); pool != nil {
			addressPools[*pool.Name] = pool
		}
	}

	pools := getBackendPoolMapValues(&addressPools)
	newManaged := c.getNewManagedPools(*pools, cbCtx)

	var existingPools []n.ApplicationGatewayBackendAddressPool
	if c.appGwConfig.BackendAddressPools != nil {
		existingPools = *c.appGwConfig.BackendAddressPools
	}

	existingUnmanaged := c.pruneManagedPools(existingPools, cbCtx)
	mergedPools := mergePools(existingUnmanaged, newManaged)

	sort.Sort(sorter.ByBackendPoolName(mergedPools))

	c.appGwConfig.BackendAddressPools = &mergedPools
	return nil
}

func (c appGwConfigBuilder) getPoolToTargetMapping(cbCtx *ConfigBuilderContext) map[string]brownfield.Target {
	listeners := make(map[string]*n.ApplicationGatewayHTTPListener)
	_, listenerMap := c.getListeners(cbCtx)
	for _, listener := range listenerMap {
		listeners[*listener.Name] = listener
	}

	poolToTarget := make(map[string]brownfield.Target)
	requestRoutingRules, paths := c.getRules(cbCtx)

	pathMap := make(map[string]n.ApplicationGatewayURLPathMap)
	for _, path := range paths {
		pathMap[*path.Name] = path
	}

	for _, rule := range requestRoutingRules {
		listenerName := utils.GetLastChunkOfSlashed(*rule.HTTPListener.ID)
		port := int32(80)
		if listeners[listenerName].Protocol == n.HTTPS {
			port = 443
		}
		if rule.URLPathMap == nil {
			t := brownfield.Target{
				Host: *listeners[listenerName].HostName,
				Port: port,
			}
			poolToTarget[utils.GetLastChunkOfSlashed(*rule.BackendAddressPool.ID)] = t
		} else {
			pathMapName := utils.GetLastChunkOfSlashed(*rule.URLPathMap.ID)
			for _, pathRule := range *pathMap[pathMapName].PathRules {
				for _, path := range *pathRule.Paths {
					t := brownfield.Target{
						Host: *listeners[listenerName].HostName,
						Port: port,
						Path: &path,
					}
					poolToTarget[utils.GetLastChunkOfSlashed(*pathRule.BackendAddressPool.ID)] = t
				}
			}
		}
	}
	return poolToTarget
}

func (c *appGwConfigBuilder) newBackendPoolMap(cbCtx *ConfigBuilderContext) map[backendIdentifier]*n.ApplicationGatewayBackendAddressPool {
	defaultPool := defaultBackendAddressPool()
	addressPools := map[string]*n.ApplicationGatewayBackendAddressPool{
		*defaultPool.Name: defaultPool,
	}

	backendPoolMap := make(map[backendIdentifier]*n.ApplicationGatewayBackendAddressPool)

	_, _, serviceBackendPairMap, _ := c.getBackendsAndSettingsMap(cbCtx)
	for backendID, serviceBackendPair := range serviceBackendPairMap {
		glog.V(5).Info("Constructing backend pool for service:", backendID.serviceKey())
		backendPoolMap[backendID] = defaultPool

		if pool := c.getBackendAddressPool(backendID, serviceBackendPair, addressPools); pool != nil {
			backendPoolMap[backendID] = pool
		}
	}
	return backendPoolMap
}

func mergePools(probesBuckets ...[]n.ApplicationGatewayBackendAddressPool) []n.ApplicationGatewayBackendAddressPool {
	uniqProbes := make(map[string]n.ApplicationGatewayBackendAddressPool)
	for _, bucket := range probesBuckets {
		for _, p := range bucket {
			uniqProbes[*p.Name] = p
		}
	}
	var merged []n.ApplicationGatewayBackendAddressPool
	for _, probe := range uniqProbes {
		merged = append(merged, probe)
	}
	return merged
}

func (c appGwConfigBuilder) pruneManagedPools(pools []n.ApplicationGatewayBackendAddressPool, kr *ConfigBuilderContext) []n.ApplicationGatewayBackendAddressPool {
	managedPool := c.getNewManagedPools(pools, kr)
	if managedPool == nil {
		return pools
	}
	indexed := make(map[string]n.ApplicationGatewayBackendAddressPool)
	for _, pool := range managedPool {
		indexed[*pool.Name] = pool
	}
	var unmanagedPools []n.ApplicationGatewayBackendAddressPool
	for _, probe := range pools {
		if _, isManaged := indexed[*probe.Name]; !isManaged {
			unmanagedPools = append(unmanagedPools, probe)
		}
	}
	return unmanagedPools
}

func getBackendPoolMapValues(m *map[string]*n.ApplicationGatewayBackendAddressPool) *[]n.ApplicationGatewayBackendAddressPool {
	var backendAddressPools []n.ApplicationGatewayBackendAddressPool
	for _, addr := range *m {
		backendAddressPools = append(backendAddressPools, *addr)
	}
	return &backendAddressPools
}

func (c *appGwConfigBuilder) getBackendAddressPool(backendID backendIdentifier, serviceBackendPair serviceBackendPortPair, addressPools map[string]*n.ApplicationGatewayBackendAddressPool) *n.ApplicationGatewayBackendAddressPool {
	endpoints, err := c.k8sContext.GetEndpointsByService(backendID.serviceKey())
	if err != nil {
		logLine := fmt.Sprintf("Failed fetching endpoints for service: %s", backendID.serviceKey())
		glog.Errorf(logLine)
		c.recorder.Event(backendID.Ingress, v1.EventTypeWarning, events.ReasonEndpointsEmpty, logLine)
		return nil
	}

	for _, subset := range endpoints.Subsets {
		if _, portExists := getUniqueTCPPorts(subset)[serviceBackendPair.BackendPort]; portExists {
			poolName := generateAddressPoolName(backendID.serviceFullName(), backendID.Backend.ServicePort.String(), serviceBackendPair.BackendPort)
			// The same service might be referenced in multiple ingress resources, this might result in multiple `serviceBackendPairMap` having the same service key but different
			// ingress resource. Thus, while generating the backend address pool, we should make sure that we are generating unique backend address pools.
			if pool, ok := addressPools[poolName]; ok {
				return pool
			}
			return newPool(poolName, subset)
		}
		logLine := fmt.Sprintf("Backend target port %d does not have matching endpoint port", serviceBackendPair.BackendPort)
		glog.Error(logLine)
		c.recorder.Event(backendID.Ingress, v1.EventTypeWarning, events.ReasonBackendPortTargetMatch, logLine)
	}
	return nil
}

func getUniqueTCPPorts(subset v1.EndpointSubset) map[int32]interface{} {
	ports := make(map[int32]interface{})
	for _, endpointsPort := range subset.Ports {
		if endpointsPort.Protocol == v1.ProtocolTCP {
			ports[endpointsPort.Port] = nil
		}
	}
	return ports
}

func newPool(poolName string, subset v1.EndpointSubset) *n.ApplicationGatewayBackendAddressPool {
	return &n.ApplicationGatewayBackendAddressPool{
		Etag: to.StringPtr("*"),
		Name: &poolName,
		ApplicationGatewayBackendAddressPoolPropertiesFormat: &n.ApplicationGatewayBackendAddressPoolPropertiesFormat{
			BackendAddresses: getAddressesForSubset(subset),
		},
	}
}

func getAddressesForSubset(subset v1.EndpointSubset) *[]n.ApplicationGatewayBackendAddress {
	// We make separate maps for IP and FQDN to ensure uniqueness within the 2 groups
	// We cannot use ApplicationGatewayBackendAddress as it contains pointer to strings and the same IP string
	// at a different address would be 2 unique keys.
	addrSet := make(map[n.ApplicationGatewayBackendAddress]interface{})
	ips := make(map[string]interface{})
	fqdns := make(map[string]interface{})
	for _, address := range subset.Addresses {
		// prefer IP address
		if len(address.IP) != 0 {
			// address specified by ip
			ips[address.IP] = nil
		} else if len(address.Hostname) != 0 {
			// address specified by hostname
			fqdns[address.Hostname] = nil
		}
	}

	for ip := range ips {
		addrSet[n.ApplicationGatewayBackendAddress{IPAddress: to.StringPtr(ip)}] = nil
	}
	for fqdn := range fqdns {
		addrSet[n.ApplicationGatewayBackendAddress{Fqdn: to.StringPtr(fqdn)}] = nil
	}
	return getBackendAddressMapKeys(&addrSet)
}

func getBackendAddressMapKeys(m *map[n.ApplicationGatewayBackendAddress]interface{}) *[]n.ApplicationGatewayBackendAddress {
	var addresses []n.ApplicationGatewayBackendAddress
	for addr := range *m {
		addresses = append(addresses, addr)
	}
	sort.Sort(sorter.ByIPFQDN(addresses))
	return &addresses
}
