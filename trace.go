/*
 * Copyright 2022 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package prometheus

import (
	"github.com/cloudwego/hertz/pkg/common/tracer"

	"github.com/prometheus/client_golang/prometheus/collectors"

	"github.com/cloudwego-contrib/cwgo-pkg/telemetry/provider/promprovider"

	"github.com/cloudwego-contrib/cwgo-pkg/telemetry/instrumentation/otelhertz"
)

// NewServerTracer provides tracer for server access, addr and path is the scrape_configs for prometheus server.
func NewServerTracer(addr, path string, opts ...Option) tracer.Tracer {
	cfg := defaultConfig()

	for _, opts := range opts {
		opts.apply(cfg)
	}
	if cfg.enableGoCollector {
		cfg.registry.MustRegister(collectors.NewGoCollector(collectors.WithGoCollectorRuntimeMetrics(cfg.runtimeMetricRules...)))
	}
	p := promprovider.NewPromProvider(
		promprovider.WithRegistry(cfg.registry),
		promprovider.WithHistogramBuckets(cfg.buckets),
		promprovider.WithHttpServer(),
	)

	if !cfg.disableServer {
		p.Serve(addr, path)
	}

	return otelhertz.NewServerTracer()
}
