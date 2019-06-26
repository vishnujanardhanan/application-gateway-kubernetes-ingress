package brownfield

import (
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
)

func GetSettingToTargetMapping(rr []n.ApplicationGatewayRequestRoutingRule, listeners ListenersByName, pathMap URLPathMapByName) NameToTarget {

	settingToTarget := make(map[string]Target)

	for _, rule := range rr {
		listenerName := utils.GetLastChunkOfSlashed(*rule.HTTPListener.ID)
		port := int32(80)
		if listeners[listenerName].Protocol == n.HTTPS {
			port = 443
		}
		if rule.URLPathMap == nil {
			t := Target{
				Host: *listeners[listenerName].HostName,
				Port: port,
			}
			settingToTarget[utils.GetLastChunkOfSlashed(*rule.BackendHTTPSettings.ID)] = t
		} else {
			pathMapName := utils.GetLastChunkOfSlashed(*rule.URLPathMap.ID)
			for _, pathRule := range *pathMap[pathMapName].PathRules {
				for _, path := range *pathRule.Paths {
					t := Target{
						Host: *listeners[listenerName].HostName,
						Port: port,
						Path: &path,
					}
					settingToTarget[utils.GetLastChunkOfSlashed(*pathRule.BackendHTTPSettings.ID)] = t
				}
			}
		}
	}
	return settingToTarget
}
