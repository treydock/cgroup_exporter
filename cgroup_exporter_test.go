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
	"github.com/prometheus/common/log"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

const (
	address = "localhost:19306"
)

func TestMain(m *testing.M) {
	if _, err := kingpin.CommandLine.Parse([]string{"--config.paths=/user.slice"}); err != nil {
		log.Fatal(err)
	}
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	fixture := filepath.Join(dir, "fixtures")
	cgroupRoot = &fixture
	procFixture := filepath.Join(fixture, "proc")
	procRoot = &procFixture
	varTrue := true
	disableExporterMetrics = &varTrue
	collectProc = &varTrue
	_ = log.Base().SetLevel("debug")
	go func() {
		http.Handle("/metrics", metricsHandler())
		log.Fatal(http.ListenAndServe(address, nil))
	}()
	time.Sleep(1 * time.Second)

	exitVal := m.Run()

	os.Exit(exitVal)
}

func TestParseCpuSet(t *testing.T) {
	expected := []string{"0", "1", "2"}
	if cpus, err := parseCpuSet("0-2"); err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
	} else if !reflect.DeepEqual(cpus, expected) {
		t.Errorf("Unexpected cpus, expected %v got %v", expected, cpus)
	}
	expected = []string{"0", "1", "4", "5", "8", "9"}
	if cpus, err := parseCpuSet("0-1,4-5,8-9"); err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
	} else if !reflect.DeepEqual(cpus, expected) {
		t.Errorf("Unexpected cpus, expected %v got %v", expected, cpus)
	}
	expected = []string{"1", "3", "5", "7"}
	if cpus, err := parseCpuSet("1,3,5,7"); err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
	} else if !reflect.DeepEqual(cpus, expected) {
		t.Errorf("Unexpected cpus, expected %v got %v", expected, cpus)
	}
}

func TestGetProcInfo(t *testing.T) {
	metric := CgroupMetric{}
	getProcInfo([]int{95521, 95525}, &metric)
	if val, ok := metric.processExec["/bin/bash"]; !ok {
		t.Errorf("Process /bin/bash not in metrics")
		return
	} else {
		if val != 2 {
			t.Errorf("Expected 2 /bin/bash processes, got %v", val)
		}
	}
	varLen := 4
	collectProcMaxExec = &varLen
	getProcInfo([]int{95521, 95525}, &metric)
	if val, ok := metric.processExec["...bash"]; !ok {
		t.Errorf("Process /bin/bash not in metrics, found: %v", metric.processExec)
		return
	} else {
		if val != 2 {
			t.Errorf("Expected 2 .../bash processes, got %v", val)
		}
	}
}

func TestCollectUserSlice(t *testing.T) {
	varFalse := false
	collectProc = &varFalse
	exporter := NewExporter([]string{"/user.slice"})
	metrics, err := exporter.collect()
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
		return
	}
	if val := len(metrics); val != 1 {
		t.Errorf("Unexpected number of metrics, got %d expected 1", val)
		return
	}
	if val := metrics[0].cpuUser; val != 0.41 {
		t.Errorf("Unexpected value for cpuUser, got %v", val)
	}
	if val := metrics[0].cpuSystem; val != 0.39 {
		t.Errorf("Unexpected value for cpuSystem, got %v", val)
	}
	if val := metrics[0].cpuTotal; val != 0.831825022 {
		t.Errorf("Unexpected value for cpuTotal, got %v", val)
	}
	if val := metrics[0].cpus; val != 0 {
		t.Errorf("Unexpected value for cpus, got %v", val)
	}
	if val := metrics[0].memoryRSS; val != 5378048 {
		t.Errorf("Unexpected value for memoryRSS, got %v", val)
	}
	if val := metrics[0].memoryCache; val != 2322432 {
		t.Errorf("Unexpected value for memoryCache, got %v", val)
	}
	if val := metrics[0].memoryUsed; val != 8081408 {
		t.Errorf("Unexpected value for memoryUsed, got %v", val)
	}
	if val := metrics[0].memoryTotal; val != 68719476736 {
		t.Errorf("Unexpected value for memoryTotal, got %v", val)
	}
	if val := metrics[0].memoryFailCount; val != 0 {
		t.Errorf("Unexpected value for memoryFailCount, got %v", val)
	}
	if val := metrics[0].memswUsed; val != 8081408 {
		t.Errorf("Unexpected value for swapUsed, got %v", val)
	}
	if val := metrics[0].memswTotal; val != 9.223372036854772e+18 {
		t.Errorf("Unexpected value for swapTotal, got %v", val)
	}
	if val := metrics[0].memswFailCount; val != 0 {
		t.Errorf("Unexpected value for swapFailCount, got %v", val)
	}
	if val := metrics[0].uid; val != "20821" {
		t.Errorf("Unexpected value for uid, got %v", val)
	}
}

func TestCollectSLURM(t *testing.T) {
	varTrue := true
	collectProc = &varTrue
	varLen := 100
	collectProcMaxExec = &varLen
	exporter := NewExporter([]string{"/slurm"})
	metrics, err := exporter.collect()
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
		return
	}
	if val := len(metrics); val != 2 {
		t.Errorf("Unexpected number of metrics, got %d expected 2", val)
		return
	}
	if val := metrics[0].cpuUser; val != 0 {
		t.Errorf("Unexpected value for cpuUser, got %v", val)
	}
	if val := metrics[0].cpuSystem; val != 0 {
		t.Errorf("Unexpected value for cpuSystem, got %v", val)
	}
	if val := metrics[0].cpuTotal; val != 0.007710215 {
		t.Errorf("Unexpected value for cpuTotal, got %v", val)
	}
	if val := metrics[0].cpus; val != 2 {
		t.Errorf("Unexpected value for cpus, got %v", val)
	}
	if val := metrics[0].memoryRSS; val != 311296 {
		t.Errorf("Unexpected value for memoryRSS, got %v", val)
	}
	if val := metrics[0].memoryCache; val != 4096 {
		t.Errorf("Unexpected value for memoryCache, got %v", val)
	}
	if val := metrics[0].memoryUsed; val != 356352 {
		t.Errorf("Unexpected value for memoryUsed, got %v", val)
	}
	if val := metrics[0].memoryTotal; val != 2147483648 {
		t.Errorf("Unexpected value for memoryTotal, got %v", val)
	}
	if val := metrics[0].memoryFailCount; val != 0 {
		t.Errorf("Unexpected value for memoryFailCount, got %v", val)
	}
	if val := metrics[0].memswUsed; val != 356352 {
		t.Errorf("Unexpected value for swapUsed, got %v", val)
	}
	if val := metrics[0].memswTotal; val != 2147483648 {
		t.Errorf("Unexpected value for swapTotal, got %v", val)
	}
	if val := metrics[0].memswFailCount; val != 0 {
		t.Errorf("Unexpected value for swapFailCount, got %v", val)
	}
	if val := metrics[0].uid; val != "20821" {
		t.Errorf("Unexpected value for uid, got %v", val)
	}
	if val := metrics[0].jobid; val != "10" {
		t.Errorf("Unexpected value for jobid, got %v", val)
	}
	if val, ok := metrics[0].processExec["/bin/bash"]; !ok {
		t.Errorf("processExec does not contain /bin/bash")
	} else {
		if val != 2 {
			t.Errorf("Unexpected 2 values for processExec /bin/bash, got %v", val)
		}
	}
}

func TestCollectTorque(t *testing.T) {
	varFalse := false
	collectProc = &varFalse
	exporter := NewExporter([]string{"/torque"})
	metrics, err := exporter.collect()
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
		return
	}
	if val := len(metrics); val != 1 {
		t.Errorf("Unexpected number of metrics, got %d expected 1", val)
		return
	}
	if val := metrics[0].cpuUser; val != 153146.31 {
		t.Errorf("Unexpected value for cpuUser, got %v", val)
	}
	if val := metrics[0].cpuSystem; val != 260.77 {
		t.Errorf("Unexpected value for cpuSystem, got %v", val)
	}
	if val := metrics[0].cpuTotal; val != 152995.785583781 {
		t.Errorf("Unexpected value for cpuTotal, got %v", val)
	}
	if val := metrics[0].cpus; val != 40 {
		t.Errorf("Unexpected value for cpus, got %v", val)
	}
	if val := metrics[0].memoryRSS; val != 82444320768 {
		t.Errorf("Unexpected value for memoryRSS, got %v", val)
	}
	if val := metrics[0].memoryCache; val != 109678592 {
		t.Errorf("Unexpected value for memoryCache, got %v", val)
	}
	if val := metrics[0].memoryUsed; val != 82553999360 {
		t.Errorf("Unexpected value for memoryUsed, got %v", val)
	}
	if val := metrics[0].memoryTotal; val != 196755132416 {
		t.Errorf("Unexpected value for memoryTotal, got %v", val)
	}
	if val := metrics[0].memoryFailCount; val != 0 {
		t.Errorf("Unexpected value for memoryFailCount, got %v", val)
	}
	if val := metrics[0].memswUsed; val != 82553999360 {
		t.Errorf("Unexpected value for swapUsed, got %v", val)
	}
	if val := metrics[0].memswTotal; val != 196755132416 {
		t.Errorf("Unexpected value for swapTotal, got %v", val)
	}
	if val := metrics[0].memswFailCount; val != 0 {
		t.Errorf("Unexpected value for swapFailCount, got %v", val)
	}
	if val := metrics[0].uid; val != "" {
		t.Errorf("Unexpected value for uid, got %v", val)
	}
	if val := metrics[0].jobid; val != "1182724" {
		t.Errorf("Unexpected value for jobid, got %v", val)
	}
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
	b, err := ioutil.ReadAll(resp.Body)
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
