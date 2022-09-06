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
	"context"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
)

// TestServerTracer test server tracer work with hertz.
func TestServerTracerWorkWithHertz(t *testing.T) {
	h := server.Default(server.WithHostPorts("127.0.0.1:8888"), server.WithTracer(NewServerTracer(":8889", "/metrics")))

	h.GET("/metricGet", func(c context.Context, ctx *app.RequestContext) {
		ctx.String(200, "hello get")
	})

	h.POST("/metricPost", func(c context.Context, ctx *app.RequestContext) {
		rand.Seed(time.Now().UnixMilli())
		// make sure the response time is greater than 50 milliseconds and less than around 151 milliseconds
		time.Sleep(time.Duration(rand.Intn(100)+51) * time.Millisecond)
		ctx.String(200, "hello post")
	})

	go h.Spin()

	time.Sleep(time.Second) // wait server start

	for i := 0; i < 10; i++ {
		_, err := http.Get("http://127.0.0.1:8888/metricGet")
		assert.Nil(t, err)
		_, err = http.Post("http://127.0.0.1:8888/metricPost", "application/json", strings.NewReader(""))
		assert.Nil(t, err)
	}

	metricsRes, err := http.Get("http://127.0.0.1:8889/metrics")

	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, metricsRes.StatusCode)

	defer metricsRes.Body.Close()

	metricsResBytes, err := io.ReadAll(metricsRes.Body)

	assert.Nil(t, err)

	metricsResStr := string(metricsResBytes)

	assert.True(t, strings.Contains(metricsResStr, `hertz_server_latency_us_bucket{method="GET",statusCode="200",le="+Inf"} 10`))
	assert.True(t, strings.Contains(metricsResStr, `hertz_server_latency_us_count{method="GET",statusCode="200"} 10`))
	assert.True(t, strings.Contains(metricsResStr, `hertz_server_latency_us_bucket{method="POST",statusCode="200",le="250000"} 10`))
	// response time is always greater than 50000 microseconds
	assert.True(t, strings.Contains(metricsResStr, `hertz_server_latency_us_bucket{method="POST",statusCode="200",le="50000"} 0`))
	assert.True(t, strings.Contains(metricsResStr, `hertz_server_latency_us_bucket{method="POST",statusCode="200",le="5000"} 0`))
	assert.True(t, strings.Contains(metricsResStr, `hertz_server_latency_us_count{method="POST",statusCode="200"} 10`))

	assert.True(t, strings.Contains(metricsResStr, `hertz_server_throughput{method="GET",statusCode="200"} 10`))
	assert.True(t, strings.Contains(metricsResStr, `hertz_server_throughput{method="POST",statusCode="200"} 10`))
}

// TestWithOption test server tracer with options
func TestWithOption(t *testing.T) {
	registry := prom.NewRegistry()

	// define your own vec
	testCounter := prom.NewCounterVec(prom.CounterOpts{
		Name: "test_with_option",
		Help: "Use for test with option",
	}, []string{
		"test1", "test2",
	})

	registry.MustRegister(testCounter)
	_ = counterAdd(testCounter, 1, prom.Labels{
		"test1": "test1",
		"test2": "test2",
	})

	h := server.Default(
		server.WithHostPorts("127.0.0.1:8891"), server.WithTracer(
			NewServerTracer(":8892", "/metrics-option",
				WithRegistry(registry),
				WithEnableGoCollector(true),
			),
		),
	)

	go h.Spin()

	time.Sleep(time.Second) // wait server start

	metricsRes, err := http.Get("http://127.0.0.1:8892/metrics-option")

	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, metricsRes.StatusCode)

	defer metricsRes.Body.Close()
	metricsResBytes, err := io.ReadAll(metricsRes.Body)

	assert.Nil(t, err)

	metricsResStr := string(metricsResBytes)

	assert.True(t, strings.Contains(metricsResStr, "test_with_option{test1=\"test1\",test2=\"test2\"} 1"))
	assert.True(t, strings.Contains(metricsResStr, "# TYPE go_gc_duration_seconds summary"))
}
