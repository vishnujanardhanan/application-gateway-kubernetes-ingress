// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package httpserver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/health"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/metricstore"
)

// HTTPServer serving probes and metrics
type HTTPServer interface {
	Start()
	Stop()
}

type httpServer struct {
	server *http.Server
}

func makeHandler(router *http.ServeMux, url string, probe health.Probe) {
	router.Handle(url, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(map[bool]int{
			true:  http.StatusOK,
			false: http.StatusServiceUnavailable,
		}[probe()])
	}))
}

// NewHealthMux makes a new *http.ServeMux
func NewHealthMux(probes health.Probes, metricStore metricstore.MetricStore) *http.ServeMux {
	router := http.NewServeMux()

	// Plumb handler for metrics
	reg := metricStore.Registry()
	router.Handle("/metrics", promhttp.InstrumentMetricHandler(
		reg,
		promhttp.HandlerFor(reg, promhttp.HandlerOpts{}),
	))

	// Plumb handlers for health probes
	var handlers = map[string]health.Probe{
		"/health/ready": probes.Readiness,
		"/health/alive": probes.Liveness,
	}
	for url, probe := range handlers {
		makeHandler(router, url, probe)
	}

	return router
}

// NewHTTPServer creates a new api server
func NewHTTPServer(probes health.Probes, metricStore metricstore.MetricStore, envVariable environment.EnvVariables) HTTPServer {
	return &httpServer{
		server: &http.Server{
			Addr:    fmt.Sprintf(":%s", envVariable.HTTPServicePort),
			Handler: NewHealthMux(probes, metricStore),
		},
	}
}

func (s *httpServer) Start() {
	go func() {
		glog.Infof("Starting API Server on %s", s.server.Addr)
		if err := s.server.ListenAndServe(); err != nil {
			glog.Fatal("Failed starting API", err)
		}
	}()
}

func (s *httpServer) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.server.Shutdown(ctx); err != nil {
		glog.Error("Unable to shutdown API server gracefully", err)
	}
}
