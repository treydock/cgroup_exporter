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
	"testing"

	"github.com/prometheus/common/promslog"
)

func TestCollectUserSlice(t *testing.T) {
	varFalse := false
	collectProc = &varFalse
	level := promslog.NewLevel()
	level.Set("debug")
	logger := promslog.New(&promslog.Config{Level: level})
	exporter := NewExporter([]string{"/user.slice"}, logger, false)
	metrics, err := exporter.collectv1()
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
	if val := metrics[0].memoryUsed; val != 27115520 {
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
	level := promslog.NewLevel()
	level.Set("debug")
	logger := promslog.New(&promslog.Config{Level: level})
	exporter := NewExporter([]string{"/slurm"}, logger, false)
	metrics, err := exporter.collectv1()
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
		return
	}
	if val := len(metrics); val != 2 {
		t.Errorf("Unexpected number of metrics, got %d expected 2", val)
		return
	}
	var m CgroupMetric
	for _, metric := range metrics {
		if metric.jobid == "10" {
			m = metric
		}
	}
	if m.jobid == "" {
		t.Errorf("Metrics with jobid=10 not found")
		return
	}
	if val := m.cpuUser; val != 0 {
		t.Errorf("Unexpected value for cpuUser, got %v", val)
	}
	if val := m.cpuSystem; val != 0 {
		t.Errorf("Unexpected value for cpuSystem, got %v", val)
	}
	if val := m.cpuTotal; val != 0.007710215 {
		t.Errorf("Unexpected value for cpuTotal, got %v", val)
	}
	if val := m.cpus; val != 2 {
		t.Errorf("Unexpected value for cpus, got %v", val)
	}
	if val := m.memoryRSS; val != 311296 {
		t.Errorf("Unexpected value for memoryRSS, got %v", val)
	}
	if val := m.memoryCache; val != 4096 {
		t.Errorf("Unexpected value for memoryCache, got %v", val)
	}
	if val := m.memoryUsed; val != 356352 {
		t.Errorf("Unexpected value for memoryUsed, got %v", val)
	}
	if val := m.memoryTotal; val != 2147483648 {
		t.Errorf("Unexpected value for memoryTotal, got %v", val)
	}
	if val := m.memoryFailCount; val != 0 {
		t.Errorf("Unexpected value for memoryFailCount, got %v", val)
	}
	if val := m.memswUsed; val != 356352 {
		t.Errorf("Unexpected value for swapUsed, got %v", val)
	}
	if val := m.memswTotal; val != 2147483648 {
		t.Errorf("Unexpected value for swapTotal, got %v", val)
	}
	if val := m.memswFailCount; val != 0 {
		t.Errorf("Unexpected value for swapFailCount, got %v", val)
	}
	if val := m.uid; val != "20821" {
		t.Errorf("Unexpected value for uid, got %v", val)
	}
	if val := m.jobid; val != "10" {
		t.Errorf("Unexpected value for jobid, got %v", val)
	}
	if val, ok := m.processExec["/bin/bash"]; !ok {
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
	level := promslog.NewLevel()
	level.Set("debug")
	logger := promslog.New(&promslog.Config{Level: level})
	exporter := NewExporter([]string{"/torque"}, logger, false)
	metrics, err := exporter.collectv1()
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
