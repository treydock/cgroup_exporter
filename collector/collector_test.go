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
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/go-kit/log"
)

var (
	mockedExitStatus = 0
	mockedStdout     string
	_, cancel        = context.WithTimeout(context.Background(), 5*time.Second)
)

func TestMain(m *testing.M) {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	fixture := filepath.Join(dir, "..", "fixtures")
	CgroupRoot = &fixture
	procFixture := filepath.Join(fixture, "proc")
	ProcRoot = &procFixture
	varTrue := true
	collectProc = &varTrue

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
	w := log.NewSyncWriter(os.Stderr)
	logger := log.NewLogfmtLogger(w)
	getProcInfo([]int{95521, 95525}, &metric, logger)
	if val, ok := metric.processExec["/bin/bash"]; !ok {
		t.Errorf("Process /bin/bash not in metrics")
		return
	} else {
		if val != 2 {
			t.Errorf("Expected 2 /bin/bash processes, got %v", val)
		}
	}
	varLen := 6
	collectProcMaxExec = &varLen
	getProcInfo([]int{95521, 95525}, &metric, logger)
	if val, ok := metric.processExec["/bi...ash"]; !ok {
		t.Errorf("Process /bin/bash not in metrics, found: %v", metric.processExec)
		return
	} else {
		if val != 2 {
			t.Errorf("Expected 2 /b...sh processes, got %v", val)
		}
	}
}

func fakeExecCommand(ctx context.Context, command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestExecCommandHelper", "--", command}
	cs = append(cs, args...)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], cs...)
	es := strconv.Itoa(mockedExitStatus)
	tmp, _ := os.MkdirTemp("", "fake")
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1",
		"GOCOVERDIR=" + tmp,
		"STDOUT=" + mockedStdout,
		"EXIT_STATUS=" + es}
	return cmd
}

func TestGetentPasswd(t *testing.T) {
	timeout := 5 * time.Second
	userLookupTimeout = &timeout
	execCommand = fakeExecCommand
	mockedExitStatus = 0
	mockedStdout = "adm:x:3:4:adm:/var/adm:/sbin/nologin"
	defer func() { execCommand = exec.CommandContext }()
	user, err := getentPasswd("3")
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
	}
	if user != "adm" {
		t.Errorf("Unexpected user: %s", user)
	}
}
