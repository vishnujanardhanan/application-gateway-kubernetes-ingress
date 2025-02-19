// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package controller

import "errors"

var (
	// ErrFetchingAppGatewayConfig is an error.
	ErrFetchingAppGatewayConfig = errors.New("unable to get specified AppGateway")

	// ErrDeployingAppGatewayConfig is an error.
	ErrDeployingAppGatewayConfig = errors.New("unable to deploy App Gateway config")
)
