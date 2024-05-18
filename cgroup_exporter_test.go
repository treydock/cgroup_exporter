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
	"github.com/go-kit/log/level"
	"github.com/treydock/cgroup_exporter/collector"
)

const (
	address = "localhost:19306"
)

func TestMain(m *testing.M) {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	fixture := filepath.Join(dir, "fixtures")
	procFixture := filepath.Join(fixture, "proc")
	collector.PidGroupPath = func(pid int) (string, error) {
		if pid == 67998 {
			return "/user.slice/user-20821.slice/session-157.scope", nil
		}
		return "", fmt.Errorf("Could not find cgroup path for %d", pid)
	}
	args := []string{
		"--config.paths=/user.slice",
		fmt.Sprintf("--path.cgroup.root=%s", fixture),
		fmt.Sprintf("--path.proc.root=%s", procFixture),
		"--web.disable-exporter-metrics",
	}
	if _, err := kingpin.CommandLine.Parse(args); err != nil {
		fmt.Printf("Error: %s", err.Error())
		os.Exit(1)
	}
	w := log.NewSyncWriter(os.Stderr)
	logger := log.NewLogfmtLogger(w)
	logger = level.NewFilter(logger, level.AllowDebug())
	go func() {
		http.Handle("/metrics", metricsHandler(logger))
		err := http.ListenAndServe(address, nil)
		if err != nil {
			fmt.Printf("Error: %s", err.Error())
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
	if !strings.Contains(body, "cgroup_memory_used_bytes{cgroup=\"/user.slice/user-20821.slice\"} 2.711552e+07") {
		t.Errorf("Unexpected value for cgroup_memory_used_bytes: %s", body)
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
