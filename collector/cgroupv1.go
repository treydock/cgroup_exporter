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
	"os/user"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/containerd/cgroups/v3/cgroup1"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
)

func NewCgroupV1Collector(paths []string, logger log.Logger) Collector {
	return NewExporter(paths, logger, false)
}

func subsystem() ([]cgroup1.Subsystem, error) {
	s := []cgroup1.Subsystem{
		cgroup1.NewCpuacct(*CgroupRoot),
		cgroup1.NewMemory(*CgroupRoot),
	}
	return s, nil
}

func pidSubsystem() ([]cgroup1.Subsystem, error) {
	s := []cgroup1.Subsystem{
		cgroup1.NewPids(*CgroupRoot),
	}
	return s, nil
}

func getInfov1(name string, metric *CgroupMetric, logger log.Logger) {
	pathBase := filepath.Base(name)
	userSlicePattern := regexp.MustCompile("^user-([0-9]+).slice$")
	userSliceMatch := userSlicePattern.FindStringSubmatch(pathBase)
	if len(userSliceMatch) == 2 {
		metric.userslice = true
		metric.uid = userSliceMatch[1]
		user, err := user.LookupId(metric.uid)
		if err != nil {
			level.Error(logger).Log("msg", "Error looking up user slice uid", "uid", metric.uid, "err", err)
		} else {
			metric.username = user.Username
		}
		return
	}
	slurmPattern := regexp.MustCompile("^/slurm/uid_([0-9]+)/job_([0-9]+)$")
	slurmMatch := slurmPattern.FindStringSubmatch(name)
	if len(slurmMatch) == 3 {
		metric.job = true
		metric.uid = slurmMatch[1]
		metric.jobid = slurmMatch[2]
		user, err := user.LookupId(metric.uid)
		if err != nil {
			level.Error(logger).Log("msg", "Error looking up slurm uid", "uid", metric.uid, "err", err)
		} else {
			metric.username = user.Username
		}
		return
	}
	if strings.HasPrefix(name, "/torque") {
		metric.job = true
		pathBaseSplit := strings.Split(pathBase, ".")
		metric.jobid = pathBaseSplit[0]
		return
	}
}

func getNamev1(p cgroup1.Process, path string, logger log.Logger) (string, error) {
	cpuacctPath := filepath.Join(*CgroupRoot, "cpuacct")
	name := strings.TrimPrefix(p.Path, cpuacctPath)
	name = strings.TrimSuffix(name, "/")
	dirs := strings.Split(name, "/")
	level.Debug(logger).Log("msg", "cgroup name", "dirs", fmt.Sprintf("%v", dirs))
	// Handle user.slice, system.slice and torque
	if len(dirs) == 3 {
		return name, nil
	}
	// Handle deeper cgroup where we want higher level, mainly SLURM
	var keepDirs []string
	for i, d := range dirs {
		if strings.HasPrefix(d, "job_") {
			keepDirs = dirs[0 : i+1]
			break
		}
	}
	if keepDirs == nil {
		return name, nil
	}
	name = strings.Join(keepDirs, "/")
	return name, nil
}

func (e *Exporter) getMetricsv1(name string, pids map[string][]int) (CgroupMetric, error) {
	metric := CgroupMetric{name: name}
	level.Debug(e.logger).Log("msg", "Loading cgroup", "root", *CgroupRoot, "path", name)

	ctrl, err := cgroup1.Load(cgroup1.StaticPath(name), cgroup1.WithHiearchy(subsystem))
	/*
		ctrl, err := cgroups.Load(subsystem, func(subsystem cgroups.Name) (string, error) {
			return name, nil
		})
	*/
	if err != nil {
		level.Error(e.logger).Log("msg", "Failed to load cgroups", "path", name, "err", err)
		metric.err = true
		return metric, err
	}
	stats, err := ctrl.Stat(cgroup1.IgnoreNotExist)
	if err != nil {
		level.Error(e.logger).Log("msg", "Failed to stat cgroups", "path", name, "err", err)
		metric.err = true
		return metric, err
	}
	if stats == nil {
		level.Error(e.logger).Log("msg", "Cgroup stats are nil", "path", name)
		metric.err = true
		return metric, err
	}
	if stats.CPU != nil {
		if stats.CPU.Usage != nil {
			metric.cpuUser = float64(stats.CPU.Usage.User) / 1000000000.0
			metric.cpuSystem = float64(stats.CPU.Usage.Kernel) / 1000000000.0
			metric.cpuTotal = float64(stats.CPU.Usage.Total) / 1000000000.0
		}
	}
	if stats.Memory != nil {
		metric.memoryRSS = float64(stats.Memory.TotalRSS)
		metric.memoryCache = float64(stats.Memory.TotalCache)
		if stats.Memory.Usage != nil {
			metric.memoryUsed = float64(stats.Memory.Usage.Usage)
			metric.memoryTotal = float64(stats.Memory.Usage.Limit)
			metric.memoryFailCount = float64(stats.Memory.Usage.Failcnt)
		}
		if stats.Memory.Swap != nil {
			metric.memswUsed = float64(stats.Memory.Swap.Usage)
			metric.memswTotal = float64(stats.Memory.Swap.Limit)
			metric.memswFailCount = float64(stats.Memory.Swap.Failcnt)
		}
	}
	cpusPath := fmt.Sprintf("%s/cpuset%s/cpuset.cpus", *CgroupRoot, name)
	if cpus, err := getCPUs(cpusPath, e.logger); err == nil {
		metric.cpus = len(cpus)
		metric.cpu_list = strings.Join(cpus, ",")
	}
	getInfov1(name, &metric, e.logger)
	if *collectProc {
		if val, ok := pids[name]; ok {
			level.Debug(e.logger).Log("msg", "Get process info", "pids", fmt.Sprintf("%v", val))
			getProcInfo(val, &metric, e.logger)
		} else {
			level.Error(e.logger).Log("msg", "Unable to get PIDs", "path", name)
			metric.err = true
		}
	}
	return metric, nil
}

func (e *Exporter) collectv1() ([]CgroupMetric, error) {
	var names []string
	var metrics []CgroupMetric
	for _, path := range e.paths {
		level.Debug(e.logger).Log("msg", "Loading cgroup", "root", *CgroupRoot, "path", path)
		control, err := cgroup1.Load(cgroup1.StaticPath(path), cgroup1.WithHiearchy(subsystem))
		if err != nil {
			level.Error(e.logger).Log("msg", "Error loading cgroup subsystem", "root", *CgroupRoot, "path", path, "err", err)
			metric := CgroupMetric{name: path, err: true}
			metrics = append(metrics, metric)
			continue
		}
		processes, err := control.Processes(cgroup1.Cpuacct, true)
		if err != nil {
			level.Error(e.logger).Log("msg", "Error loading cgroup processes", "path", path, "err", err)
			metric := CgroupMetric{name: path, err: true}
			metrics = append(metrics, metric)
			continue
		}
		level.Debug(e.logger).Log("msg", "Found processes", "processes", len(processes))
		pids := make(map[string][]int)
		for _, p := range processes {
			level.Debug(e.logger).Log("msg", "Get Name", "process", p.Path, "pid", p.Pid, "path", path)
			name, err := getNamev1(p, path, e.logger)
			if err != nil {
				level.Error(e.logger).Log("msg", "Error getting cgroup name for process", "process", p.Path, "path", path, "err", err)
				continue
			}
			if !sliceContains(names, name) {
				names = append(names, name)
			}
			if val, ok := pids[name]; ok {
				if !sliceContains(val, p.Pid) {
					val = append(val, p.Pid)
				}
				pids[name] = val
			} else {
				pids[name] = []int{p.Pid}
			}
		}
		wg := &sync.WaitGroup{}
		wg.Add(len(names))
		for _, name := range names {
			go func(n string, p map[string][]int) {
				defer wg.Done()
				metric, _ := e.getMetricsv1(n, p)
				metricLock.Lock()
				metrics = append(metrics, metric)
				metricLock.Unlock()
			}(name, pids)
		}
		wg.Wait()
	}
	return metrics, nil
}
