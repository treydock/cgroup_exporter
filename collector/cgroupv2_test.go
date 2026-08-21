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

package collector

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/prometheus/common/promslog"
)

func TestParsePressureData(t *testing.T) {
	input := `some avg10=0.00 avg60=0.27 avg300=0.44 total=25314114
full avg10=0.00 avg60=0.00 avg300=0.00 total=2578704`

	metrics, err := parsePressureData(input)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}

	if len(metrics) != 2 {
		t.Fatalf("Expected 2 metrics, got %d", len(metrics))
	}

	// Check "some" pressure
	if metrics[0].Type != "some" {
		t.Errorf("Expected Type 'some', got '%s'", metrics[0].Type)
	}
	if metrics[0].Avg10 != 0.00 {
		t.Errorf("Expected Avg10 0.00, got %f", metrics[0].Avg10)
	}
	if metrics[0].Avg60 != 0.27 {
		t.Errorf("Expected Avg60 0.27, got %f", metrics[0].Avg60)
	}
	if metrics[0].Avg300 != 0.44 {
		t.Errorf("Expected Avg300 0.44, got %f", metrics[0].Avg300)
	}
	// total should be converted from microseconds to seconds
	expectedSomeTotal := 25314114.0 / 1000000.0
	if metrics[0].Total != expectedSomeTotal {
		t.Errorf("Expected Total %f, got %f", expectedSomeTotal, metrics[0].Total)
	}

	// Check "full" pressure
	if metrics[1].Type != "full" {
		t.Errorf("Expected Type 'full', got '%s'", metrics[1].Type)
	}
	if metrics[1].Avg10 != 0.00 {
		t.Errorf("Expected Avg10 0.00, got %f", metrics[1].Avg10)
	}
	if metrics[1].Avg60 != 0.00 {
		t.Errorf("Expected Avg60 0.00, got %f", metrics[1].Avg60)
	}
	if metrics[1].Avg300 != 0.00 {
		t.Errorf("Expected Avg300 0.00, got %f", metrics[1].Avg300)
	}
	expectedFullTotal := 2578704.0 / 1000000.0
	if metrics[1].Total != expectedFullTotal {
		t.Errorf("Expected Total %f, got %f", expectedFullTotal, metrics[1].Total)
	}
}

func TestParsePressureDataEmpty(t *testing.T) {
	metrics, err := parsePressureData("")
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}
	if len(metrics) != 0 {
		t.Errorf("Expected 0 metrics for empty input, got %d", len(metrics))
	}
}

func TestParsePressureDataSingleLine(t *testing.T) {
	input := `some avg10=1.50 avg60=2.25 avg300=3.75 total=1000000`

	metrics, err := parsePressureData(input)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}

	if len(metrics) != 1 {
		t.Fatalf("Expected 1 metric, got %d", len(metrics))
	}

	if metrics[0].Type != "some" {
		t.Errorf("Expected Type 'some', got '%s'", metrics[0].Type)
	}
	if metrics[0].Avg10 != 1.50 {
		t.Errorf("Expected Avg10 1.50, got %f", metrics[0].Avg10)
	}
	if metrics[0].Avg60 != 2.25 {
		t.Errorf("Expected Avg60 2.25, got %f", metrics[0].Avg60)
	}
	if metrics[0].Avg300 != 3.75 {
		t.Errorf("Expected Avg300 3.75, got %f", metrics[0].Avg300)
	}
	if metrics[0].Total != 1.0 {
		t.Errorf("Expected Total 1.0, got %f", metrics[0].Total)
	}
}

func TestParsePressureDataWithEmptyLines(t *testing.T) {
	input := `
some avg10=0.50 avg60=1.00 avg300=1.50 total=5000000

full avg10=0.25 avg60=0.50 avg300=0.75 total=2500000
`

	metrics, err := parsePressureData(input)
	if err != nil {
		t.Fatalf("Unexpected error: %s", err)
	}

	if len(metrics) != 2 {
		t.Fatalf("Expected 2 metrics, got %d", len(metrics))
	}

	if metrics[0].Type != "some" {
		t.Errorf("Expected Type 'some', got '%s'", metrics[0].Type)
	}
	if metrics[1].Type != "full" {
		t.Errorf("Expected Type 'full', got '%s'", metrics[1].Type)
	}
}

func TestGetStatv2(t *testing.T) {
	_, err := getStatv2("swapcached", "/dne")
	if err == nil {
		t.Errorf("Expected error with /dne but none given")
	}
	path := filepath.Join(*CgroupRoot, "system.slice")
	_, err = getStatv2("swapcached", path)
	if err == nil {
		t.Errorf("Expected error with /dne but none given")
	}
	path = filepath.Join(*CgroupRoot, "user.slice/user-20821.slice/memory.max")
	_, err = getStatv2("swapcached", path)
	if err == nil {
		t.Errorf("Expected error with single value file but none given")
	}
	path = filepath.Join(*CgroupRoot, "stat.invalid")
	_, err = getStatv2("nan", path)
	if err == nil {
		t.Errorf("Expected error with stat.invalid but none given")
	}
	path = filepath.Join(*CgroupRoot, "user.slice/user-20821.slice/memory.stat")
	_, err = getStatv2("dne", path)
	if err == nil {
		t.Errorf("Expected error when stat key missing but none given")
	}
	stat, err := getStatv2("swapcached", path)
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
	if stat != 0 {
		t.Errorf("Unexpectd value: %v", stat)
	}
}

func TestCollectv2Error(t *testing.T) {
	level := promslog.NewLevel()
	level.Set("debug")
	logger := promslog.New(&promslog.Config{Level: level})
	exporter := NewExporter([]string{"/dne"}, logger, true)
	metrics, err := exporter.collectv2()
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
		return
	}
	if val := len(metrics); val != 1 {
		t.Errorf("Unexpected number of metrics, got %d expected 1", val)
		return
	}
	if val := metrics[0].err; val != true {
		t.Errorf("Unexpected value for err, got %v", val)
	}
}

func TestCollectv2UserSlice(t *testing.T) {
	varFalse := false
	collectProc = &varFalse
	PidGroupPath = func(pid int) (string, error) {
		if pid == 67998 {
			return "/user.slice/user-20821.slice/session-157.scope", nil
		}
		return "", fmt.Errorf("Could not find cgroup path for %d", pid)
	}
	level := promslog.NewLevel()
	level.Set("debug")
	logger := promslog.New(&promslog.Config{Level: level})
	exporter := NewExporter([]string{"/user.slice"}, logger, true)
	metrics, err := exporter.collectv2()
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
		return
	}
	if val := len(metrics); val != 1 {
		t.Errorf("Unexpected number of metrics, got %d expected 1", val)
		return
	}
	if val := metrics[0].name; val != "/user.slice/user-20821.slice" {
		t.Errorf("Unexpected value for name, got %v", val)
	}
	if val := metrics[0].cpuUser; val != 15.270449 {
		t.Errorf("Unexpected value for cpuUser, got %v", val)
	}
	if val := metrics[0].cpuSystem; val != 2.705424 {
		t.Errorf("Unexpected value for cpuSystem, got %v", val)
	}
	if val := metrics[0].cpuTotal; val != 17.975873 {
		t.Errorf("Unexpected value for cpuTotal, got %v", val)
	}
	if val := metrics[0].cpus; val != 0 {
		t.Errorf("Unexpected value for cpus, got %v", val)
	}
	if val := metrics[0].memoryRSS; val != 22626304 {
		t.Errorf("Unexpected value for memoryRSS, got %v", val)
	}
	if val := metrics[0].memoryUsed; val != 27115520 {
		t.Errorf("Unexpected value for memoryUsed, got %v", val)
	}
	if val := metrics[0].memoryTotal; val != 2147483648 {
		t.Errorf("Unexpected value for memoryTotal, got %v", val)
	}
	if val := metrics[0].memoryFailCount; val != 0 {
		t.Errorf("Unexpected value for memoryFailCount, got %v", val)
	}
	if val := metrics[0].memswUsed; val != 0 {
		t.Errorf("Unexpected value for swapUsed, got %v", val)
	}
	if val := metrics[0].memswTotal; val != 1.8446744073709552e+19 {
		t.Errorf("Unexpected value for swapTotal, got %v", val)
	}
	if val := metrics[0].uid; val != "20821" {
		t.Errorf("Unexpected value for uid, got %v", val)
	}
}

func TestCollectv2SLURM(t *testing.T) {
	varTrue := true
	collectProc = &varTrue
	varLen := 100
	collectProcMaxExec = &varLen
	PidGroupPath = func(pid int) (string, error) {
		if pid == 49276 {
			return "/system.slice/slurmstepd.scope/job_4/step_0/user/task_0", nil
		}
		if pid == 43310 {
			return "/system.slice/slurmstepd.scope/system", nil
		}
		return "", fmt.Errorf("Could not find cgroup path for %d", pid)
	}
	level := promslog.NewLevel()
	level.Set("debug")
	logger := promslog.New(&promslog.Config{Level: level})
	exporter := NewExporter([]string{"/slurm"}, logger, true)
	metrics, err := exporter.collectv2()
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
		return
	}
	if val := len(metrics); val != 1 {
		t.Errorf("Unexpected number of metrics, got %d expected 1", val)
		return
	}
	var m CgroupMetric
	for _, metric := range metrics {
		if metric.jobid == "4" {
			m = metric
		}
	}
	if m.jobid == "" {
		t.Errorf("Metrics with jobid=4 not found")
		return
	}
	if val := m.name; val != "/system.slice/slurmstepd.scope/job_4" {
		t.Errorf("Unexpected value for name, got %v", val)
	}
	if val := m.cpuUser; val != 0.049043 {
		t.Errorf("Unexpected value for cpuUser, got %v", val)
	}
	if val := m.cpuSystem; val != 0.077642 {
		t.Errorf("Unexpected value for cpuSystem, got %v", val)
	}
	if val := m.cpuTotal; val != 0.126686 {
		t.Errorf("Unexpected value for cpuTotal, got %v", val)
	}
	if val := m.cpus; val != 1 {
		t.Errorf("Unexpected value for cpus, got %v", val)
	}
	if val := m.memoryRSS; val != 2777088 {
		t.Errorf("Unexpected value for memoryRSS, got %v", val)
	}
	if val := m.memoryUsed; val != 5660672 {
		t.Errorf("Unexpected value for memoryUsed, got %v", val)
	}
	if val := m.memoryTotal; val != 1835008000 {
		t.Errorf("Unexpected value for memoryTotal, got %v", val)
	}
	if val := m.memoryFailCount; val != 0 {
		t.Errorf("Unexpected value for memoryFailCount, got %v", val)
	}
	if val := m.memswUsed; val != 0 {
		t.Errorf("Unexpected value for swapUsed, got %v", val)
	}
	if val := m.memswTotal; val != 1835008000 {
		t.Errorf("Unexpected value for swapTotal, got %v", val)
	}
	if val := m.uid; val != "20821" {
		t.Errorf("Unexpected value for uid, got %v", val)
	}
	if val := m.jobid; val != "4" {
		t.Errorf("Unexpected value for jobid, got %v", val)
	}
	if val, ok := m.processExec["/usr/bin/bash"]; !ok {
		t.Errorf("processExec does not contain /bin/bash")
	} else {
		if val != 1 {
			t.Errorf("Unexpected 1 values for processExec /usr/bin/bash, got %v", val)
		}
	}
}
