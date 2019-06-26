package brownfield

import (
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
)

type NameToTarget map[string]Target

type ListenersByName map[string]*n.ApplicationGatewayHTTPListener

type URLPathMapByName map[string]n.ApplicationGatewayURLPathMap

type Target struct {
	Host string
	Port int32
	Path *string
}
