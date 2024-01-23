// Copyright 2020 Trey Dockendorf
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	kingpin "github.com/alecthomas/kingpin/v2"
	"github.com/go-kit/log"
	"github.com/treydock/cgroup_exporter/collector"
)

const (
	address = "localhost:19306"
)

func TestMain(m *testing.M) {
	if _, err := kingpin.CommandLine.Parse([]string{"--config.paths=/user.slice"}); err != nil {
		os.Exit(1)
	}
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	fixture := filepath.Join(dir, "fixtures")
	collector.CgroupRoot = &fixture
	procFixture := filepath.Join(fixture, "proc")
	collector.ProcRoot = &procFixture
	varTrue := true
	disableExporterMetrics = &varTrue
	w := log.NewSyncWriter(os.Stderr)
	logger := log.NewLogfmtLogger(w)
	go func() {
		http.Handle("/metrics", metricsHandler(logger))
		err := http.ListenAndServe(address, nil)
		if err != nil {
			os.Exit(1)
		}
	}()
	time.Sleep(1 * time.Second)

	exitVal := m.Run()

	os.Exit(exitVal)
}

func TestMetricsHandler(t *testing.T) {
	body, err := queryExporter()
	if err != nil {
		t.Fatalf("Unexpected error GET /metrics: %s", err.Error())
	}
	if !strings.Contains(body, "cgroup_memory_used_bytes{cgroup=\"/user.slice/user-20821.slice\"} 8.081408e+06") {
		t.Errorf("Unexpected value for cgroup_memory_used_bytes")
	}
}

func TestMetricsHandlerBadPath(t *testing.T) {
	cPath := "/dne"
	configPaths = &cPath
	body, err := queryExporter()
	if err != nil {
		t.Fatalf("Unexpected error GET /metrics: %s", err.Error())
	}
	if !strings.Contains(body, "cgroup_exporter_collect_error{cgroup=\"/dne\"} 1") {
		t.Errorf("Unexpected value for cgroup_memory_used_bytes")
	}
}

func queryExporter() (string, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s/metrics", address))
	if err != nil {
		return "", err
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if err := resp.Body.Close(); err != nil {
		return "", err
	}
	if want, have := http.StatusOK, resp.StatusCode; want != have {
		return "", fmt.Errorf("want /metrics status code %d, have %d. Body:\n%s", want, have, b)
	}
	return string(b), nil
}
