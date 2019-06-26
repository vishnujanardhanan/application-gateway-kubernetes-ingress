// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/brownfield"
	"reflect"
	"strings"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
)

func (c appGwConfigBuilder) getNewManagedPools(pools []n.ApplicationGatewayBackendAddressPool, cbCtx *ConfigBuilderContext) []n.ApplicationGatewayBackendAddressPool {
	blacklist := getProhibitedTargetList(cbCtx)
	whitelist := getManagedTargetList(cbCtx)

	if len(*blacklist) == 0 && len(*whitelist) == 0 {
		return pools
	}

	var managedPools []n.ApplicationGatewayBackendAddressPool
	poolToTarget := c.getPoolToTargetMapping(cbCtx)

	// Blacklist takes priority
	if len(*blacklist) > 0 {
		// Apply blacklist
		for _, pool := range pools {
			target := poolToTarget[*pool.Name]
			if isTargetInList(target, blacklist) {
				continue
			}
			managedPools = append(managedPools, pool)
		}
		return managedPools
	}

	// Is it whitelisted
	for _, pool := range pools {
		target := poolToTarget[*pool.Name]
		if isTargetInList(target, whitelist) {
			managedPools = append(managedPools, pool)
		}
	}

	for _, pool := range pools {
		target := poolToTarget[*pool.Name]
		if isTargetInList(target, blacklist) {
			managedPools = append(managedPools, pool)
		}
	}
	return managedPools
}

func getManagedProbes(probes []n.ApplicationGatewayProbe, cbCtx *ConfigBuilderContext) []n.ApplicationGatewayProbe {
	var managedProbes []n.ApplicationGatewayProbe

	blacklist := getProhibitedTargetList(cbCtx)
	whitelist := getManagedTargetList(cbCtx)

	if len(*blacklist) == 0 && len(*whitelist) == 0 {
		return probes
	}

	// Blacklist takes priority
	if len(*blacklist) > 0 {
		// Apply blacklist
		for _, probe := range probes {
			if inProbeList(&probe, blacklist) {
				continue
			}
			managedProbes = append(managedProbes, probe)
		}
		return managedProbes
	}

	// Is it Whitelisted
	for _, probe := range probes {
		if inProbeList(&probe, whitelist) {
			managedProbes = append(managedProbes, probe)
		}
	}

	for _, probe := range probes {
		if inProbeList(&probe, blacklist) {
			managedProbes = append(managedProbes, probe)
		}
	}
	return managedProbes
}

func getProhibitedTargetList(cbCtx *ConfigBuilderContext) *[]brownfield.Target {
	var tl []brownfield.Target
	for _, pt := range cbCtx.ProhibitedTargets {
		if len(pt.Spec.Paths) < 1 {
			tl = append(tl, brownfield.Target{
				Host: pt.Spec.Hostname,
				Port: pt.Spec.Port,
				Path: nil,
			})
		}
		for _, path := range pt.Spec.Paths {
			tl = append(tl, brownfield.Target{
				Host: pt.Spec.Hostname,
				Port: pt.Spec.Port,
				Path: to.StringPtr(path),
			})
		}
	}
	return &tl
}

func getManagedTargetList(cbCtx *ConfigBuilderContext) *[]brownfield.Target {
	var tl []brownfield.Target
	for _, mt := range cbCtx.ManagedTargets {
		if len(mt.Spec.Paths) < 1 {
			tl = append(tl, brownfield.Target{
				Host: mt.Spec.Hostname,
				Port: mt.Spec.Port,
				Path: nil,
			})
		}
		for _, path := range mt.Spec.Paths {
			tl = append(tl, brownfield.Target{
				Host: mt.Spec.Hostname,
				Port: mt.Spec.Port,
				Path: to.StringPtr(path),
			})
		}
	}
	return &tl
}

func isTargetInList(tgt brownfield.Target, targetList *[]brownfield.Target) bool {
	for _, t := range *targetList {
		if reflect.DeepEqual(tgt, t) {
			// Found it
			return true
		}
	}

	// Did not find it
	return false
}

func inProbeList(probe *n.ApplicationGatewayProbe, targetList *[]brownfield.Target) bool {
	for _, t := range *targetList {
		if t.Host == *probe.Host {
			if t.Path == nil {
				// Host matches; No paths - found it
				return true
			} else if normalizePath(*t.Path) == normalizePath(*probe.Path) {
				// Matches a path - found it
				return true
			}
		}
	}

	// Did not find it
	return false
}

func normalizePath(path string) string {
	trimmed, prevTrimmed := "", path
	cutset := "*/"
	for trimmed != prevTrimmed {
		prevTrimmed = trimmed
		trimmed = strings.TrimRight(path, cutset)
	}
	return trimmed
}
