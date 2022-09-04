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
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stretchr/testify/assert"
)

func TestMetrics(t *testing.T) {
	registry := prom.NewRegistry()

	http.Handle("/metrics-demo", promhttp.HandlerFor(registry, promhttp.HandlerOpts{ErrorHandling: promhttp.ContinueOnError}))

	go func() {
		if err := http.ListenAndServe(":9090", nil); err != nil {
			hlog.Fatalf("HERTZ: Unable to start a http server, err: %s", err.Error())
		}
	}()

	counter := prom.NewCounterVec(
		prom.CounterOpts{
			Name:        "test_counter",
			ConstLabels: prom.Labels{"service": "prometheus-test"},
		},
		[]string{"test1", "test2"},
	)

	registry.MustRegister(counter)

	histogram := prom.NewHistogramVec(
		prom.HistogramOpts{
			Name:        "test_histogram",
			ConstLabels: prom.Labels{"service": "prometheus-test"},
			Buckets:     defaultBuckets,
		},
		[]string{"test1", "test2"},
	)

	registry.MustRegister(histogram)

	labels := prom.Labels{
		"test1": "abc",
		"test2": "def",
	}

	assert.Nil(t, counterAdd(counter, 6, labels))
	assert.Nil(t, histogramObserve(histogram, 100*time.Millisecond, labels))

	res, err := http.Get("http://localhost:9090/metrics-demo")

	assert.Nil(t, err)

	defer res.Body.Close()

	bodyBytes, err := io.ReadAll(res.Body)

	assert.Nil(t, err)

	bodyStr := string(bodyBytes)

	assert.True(t, strings.Contains(bodyStr, `test_counter{service="prometheus-test",test1="abc",test2="def"} 6`))
	assert.True(t, strings.Contains(bodyStr, `test_histogram_bucket{service="prometheus-test",test1="abc",test2="def",le="50000"} 0`))
	assert.True(t, strings.Contains(bodyStr, `test_histogram_bucket{service="prometheus-test",test1="abc",test2="def",le="100000"} 1`))
}
