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
	"github.com/prometheus/common/log"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
	"path/filepath"
	"runtime"
	"testing"
)

func TestParseCpuSet(t *testing.T) {
	if cpus, err := parseCpuSet("0-2"); err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
	} else if cpus != 3 {
		t.Errorf("Unexpected cpus, expected 3 got %d", cpus)
	}
	if cpus, err := parseCpuSet("0-1,4-5,8-9"); err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
	} else if cpus != 6 {
		t.Errorf("Unexpected cpus, expected 6 got %d", cpus)
	}
	if cpus, err := parseCpuSet("1,3,5,7"); err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
	} else if cpus != 4 {
		t.Errorf("Unexpected cpus, expected 4 got %d", cpus)
	}
}

func TestCollectUserSlice(t *testing.T) {
	if _, err := kingpin.CommandLine.Parse([]string{"--config.paths=/user.slice"}); err != nil {
		t.Fatal(err)
	}
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	fixture := filepath.Join(dir, "test")
	cgroupRoot = &fixture

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
	if val := metrics[0].memoryUsed; val != 8081408 {
		t.Errorf("Unexpected value for memoryUsed, got %v", val)
	}
	if val := metrics[0].memoryTotal; val != 68719476736 {
		t.Errorf("Unexpected value for memoryTotal, got %v", val)
	}
	if val := metrics[0].swapUsed; val != 0 {
		t.Errorf("Unexpected value for swapUsed, got %v", val)
	}
	if val := metrics[0].swapTotal; val != 9223371968135295000 {
		t.Errorf("Unexpected value for swapTotal, got %v", val)
	}
	if val := metrics[0].uid; val != "20821" {
		t.Errorf("Unexpected value for uid, got %v", val)
	}
}

func TestCollectSLURM(t *testing.T) {
	if _, err := kingpin.CommandLine.Parse([]string{"--config.paths=/slurm"}); err != nil {
		t.Fatal(err)
	}
	_ = log.Base().SetLevel("debug")
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	fixture := filepath.Join(dir, "test")
	cgroupRoot = &fixture

	exporter := NewExporter([]string{"/slurm"})
	metrics, err := exporter.collect()
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
		return
	}
	if val := len(metrics); val != 1 {
		t.Errorf("Unexpected number of metrics, got %d expected 1", val)
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
	if val := metrics[0].memoryUsed; val != 356352 {
		t.Errorf("Unexpected value for memoryUsed, got %v", val)
	}
	if val := metrics[0].memoryTotal; val != 2147483648 {
		t.Errorf("Unexpected value for memoryTotal, got %v", val)
	}
	if val := metrics[0].swapUsed; val != 0 {
		t.Errorf("Unexpected value for swapUsed, got %v", val)
	}
	if val := metrics[0].swapTotal; val != 0 {
		t.Errorf("Unexpected value for swapTotal, got %v", val)
	}
	if val := metrics[0].uid; val != "20821" {
		t.Errorf("Unexpected value for uid, got %v", val)
	}
	if val := metrics[0].jobid; val != "10" {
		t.Errorf("Unexpected value for jobid, got %v", val)
	}
}

func TestCollectTorque(t *testing.T) {
	if _, err := kingpin.CommandLine.Parse([]string{"--config.paths=/torque"}); err != nil {
		t.Fatal(err)
	}
	_ = log.Base().SetLevel("debug")
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	fixture := filepath.Join(dir, "test")
	cgroupRoot = &fixture

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
	if val := metrics[0].memoryUsed; val != 82553999360 {
		t.Errorf("Unexpected value for memoryUsed, got %v", val)
	}
	if val := metrics[0].memoryTotal; val != 196755132416 {
		t.Errorf("Unexpected value for memoryTotal, got %v", val)
	}
	if val := metrics[0].swapUsed; val != 0 {
		t.Errorf("Unexpected value for swapUsed, got %v", val)
	}
	if val := metrics[0].swapTotal; val != 0 {
		t.Errorf("Unexpected value for swapTotal, got %v", val)
	}
	if val := metrics[0].uid; val != "" {
		t.Errorf("Unexpected value for uid, got %v", val)
	}
	if val := metrics[0].jobid; val != "1182724" {
		t.Errorf("Unexpected value for jobid, got %v", val)
	}
}
