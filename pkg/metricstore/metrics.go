// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package metricstore

import (
	"fmt"
)

// MetricType is enum corresponding to various metrics
type MetricType int

const (
	// MetricStringFormat is a string storing metric string format
	MetricStringFormat = "Metric: %s | Average: %f | Last: %f | Counter: %f"
)

const (
	// LatencyMetric is metric corresponding to AppGw update times.
	UpdateLatencyMetricInSecs MetricType = iota + 1
)

var metricNameMap = map[MetricType]string{
	UpdateLatencyMetricInSecs: "UpdateLatencyMetricInSecs",
}

// Metric is struct storing the last and average for a metric
type Metric struct {
	Name    string
	Last    float64
	Average float64
	Counter float64
}

// NewMetric creates new metric
func NewMetric(metricType MetricType) *Metric {
	return &Metric{
		Name:    metricNameMap[metricType],
		Last:    0,
		Average: 0,
		Counter: 0,
	}
}

// Update a metric
func (m *Metric) Update(current float64) {
	m.Average = m.Average + (current-m.Average)/(m.Counter+1)
	m.Last = current
	m.Counter = m.Counter + 1
}

func (m *Metric) String() string {
	return fmt.Sprintf(MetricStringFormat, m.Name, m.Average, m.Last, m.Counter)
}
