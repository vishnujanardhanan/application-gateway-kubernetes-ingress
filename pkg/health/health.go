// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package health

// Probe is a type alias for a function.
type Probe func() bool

// Probes is the interface for liveness and readiness probes
type Probes interface {
	Liveness() bool
	Readiness() bool
}
