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
	"log/slog"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs"
)

var (
	collectFullSlurm   = kingpin.Flag("collect.fullslurm", "Boolean that sets if to collect all slurm steps and tasks").Default("false").Bool()
	collectProc        = kingpin.Flag("collect.proc", "Boolean that sets if to collect proc information").Default("false").Bool()
	CgroupRoot         = kingpin.Flag("path.cgroup.root", "Root path to cgroup fs").Default(defCgroupRoot).String()
	collectProcMaxExec = kingpin.Flag("collect.proc.max-exec", "Max length of process executable to record").Default("100").Int()
	ProcRoot           = kingpin.Flag("path.proc.root", "Root path to proc fs").Default(defProcRoot).String()
	metricLock         = sync.RWMutex{}
)

const (
	Namespace     = "cgroup"
	defCgroupRoot = "/sys/fs/cgroup"
	defProcRoot   = "/proc"
)

type Collector interface {
	// Get new metrics and expose them via prometheus registry.
	Describe(ch chan<- *prometheus.Desc)
	Collect(ch chan<- prometheus.Metric)
}

type ExporterDescs struct {
	cpuUser         *prometheus.Desc
	cpuSystem       *prometheus.Desc
	cpuTotal        *prometheus.Desc
	cpus            *prometheus.Desc
	cpu_info        *prometheus.Desc
	memoryRSS       *prometheus.Desc
	memoryCache     *prometheus.Desc
	memoryUsed      *prometheus.Desc
	memoryTotal     *prometheus.Desc
	memoryFailCount *prometheus.Desc
	memswUsed       *prometheus.Desc
	memswTotal      *prometheus.Desc
	memswFailCount  *prometheus.Desc
	processExec     *prometheus.Desc
}

type Exporter struct {
	paths        []string
	collectError *prometheus.Desc
	info         *prometheus.Desc
	descsRegular *ExporterDescs
	descsJob     *ExporterDescs
	logger       *slog.Logger
	cgroupv2     bool
}

type CgroupMetric struct {
	name            string
	cpuUser         float64
	cpuSystem       float64
	cpuTotal        float64
	cpus            int
	cpu_list        string
	memoryRSS       float64
	memoryCache     float64
	memoryUsed      float64
	memoryTotal     float64
	memoryFailCount float64
	memswUsed       float64
	memswTotal      float64
	memswFailCount  float64
	userslice       bool
	job             bool
	uid             string
	username        string
	jobid           string
	step		string
	task		string
	processExec     map[string]float64
	err             bool
}

func NewCgroupCollector(cgroupV2 bool, paths []string, logger *slog.Logger) Collector {
	var collector Collector
	if cgroupV2 {
		collector = NewCgroupV2Collector(paths, logger)
	} else {
		collector = NewCgroupV1Collector(paths, logger)
	}
	return collector
}

func NewDescs(labels []string) *ExporterDescs{
	return &ExporterDescs{
		cpuUser: prometheus.NewDesc(prometheus.BuildFQName(Namespace, "cpu", "user_seconds"),
			"Cumalitive CPU user seconds for cgroup", labels, nil),
		cpuSystem: prometheus.NewDesc(prometheus.BuildFQName(Namespace, "cpu", "system_seconds"),
			"Cumalitive CPU system seconds for cgroup", labels, nil),
		cpuTotal: prometheus.NewDesc(prometheus.BuildFQName(Namespace, "cpu", "total_seconds"),
			"Cumalitive CPU total seconds for cgroup", labels, nil),
		cpus: prometheus.NewDesc(prometheus.BuildFQName(Namespace, "", "cpus"),
			"Number of CPUs in the cgroup", labels, nil),
		cpu_info: prometheus.NewDesc(prometheus.BuildFQName(Namespace, "", "cpu_info"),
			"Information about the cgroup CPUs", slices.Concat(labels, []string{"cpus"}), nil),
		memoryRSS: prometheus.NewDesc(prometheus.BuildFQName(Namespace, "memory", "rss_bytes"),
			"Memory RSS used in bytes", labels, nil),
		memoryCache: prometheus.NewDesc(prometheus.BuildFQName(Namespace, "memory", "cache_bytes"),
			"Memory cache used in bytes", labels, nil),
		memoryUsed: prometheus.NewDesc(prometheus.BuildFQName(Namespace, "memory", "used_bytes"),
			"Memory used in bytes", labels, nil),
		memoryTotal: prometheus.NewDesc(prometheus.BuildFQName(Namespace, "memory", "total_bytes"),
			"Memory total given to cgroup in bytes", labels, nil),
		memoryFailCount: prometheus.NewDesc(prometheus.BuildFQName(Namespace, "memory", "fail_count"),
			"Memory fail count", labels, nil),
		memswUsed: prometheus.NewDesc(prometheus.BuildFQName(Namespace, "memsw", "used_bytes"),
			"Swap used in bytes", labels, nil),
		memswTotal: prometheus.NewDesc(prometheus.BuildFQName(Namespace, "memsw", "total_bytes"),
			"Swap total given to cgroup in bytes", labels, nil),
		memswFailCount: prometheus.NewDesc(prometheus.BuildFQName(Namespace, "memsw", "fail_count"),
			"Swap fail count", labels, nil),
		processExec: prometheus.NewDesc(prometheus.BuildFQName(Namespace, "", "process_exec_count"),
			"Count of instances of a given process", []string{"cgroup", "exec"}, nil),
	}
}

func NewExporter(paths []string, logger *slog.Logger, cgroupv2 bool) *Exporter {
	return &Exporter{
		paths:        paths,
		descsRegular: NewDescs([]string{"cgroup"}),
		descsJob:     NewDescs([]string{"jobid", "step", "task"}),
		info: prometheus.NewDesc(prometheus.BuildFQName(Namespace, "", "info"),
			"User slice information", []string{"cgroup", "username", "uid", "jobid"}, nil),
		collectError: prometheus.NewDesc(prometheus.BuildFQName(Namespace, "exporter", "collect_error"),
			"Indicates collection error, 0=no error, 1=error", []string{"cgroup"}, nil),
		logger:   logger,
		cgroupv2: cgroupv2,
	}
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.descsRegular.cpuUser
	ch <- e.descsRegular.cpuSystem
	ch <- e.descsRegular.cpuTotal
	ch <- e.descsRegular.cpus
	ch <- e.descsRegular.cpu_info
	ch <- e.descsRegular.memoryRSS
	ch <- e.descsRegular.memoryCache
	ch <- e.descsRegular.memoryUsed
	ch <- e.descsRegular.memoryTotal
	ch <- e.descsRegular.memoryFailCount
	ch <- e.descsRegular.memswUsed
	ch <- e.descsRegular.memswTotal
	ch <- e.descsRegular.memswFailCount
	ch <- e.info
	if *collectProc {
		ch <- e.descsRegular.processExec
	}
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	var metrics []CgroupMetric
	var descs *ExporterDescs
	if e.cgroupv2 {
		metrics, _ = e.collectv2()
	} else {
		metrics, _ = e.collectv1()
	}

	for _, m := range metrics {
		var labels []string
		if m.err {
			ch <- prometheus.MustNewConstMetric(e.collectError, prometheus.GaugeValue, 1, m.name)
		}
		if m.job {
			descs = e.descsJob
			labels = []string{m.jobid, m.step, m.task}
		} else {
			descs = e.descsRegular
			labels = []string{m.name}
		}
		ch <- prometheus.MustNewConstMetric(descs.cpuUser, prometheus.GaugeValue, m.cpuUser, labels...)
		ch <- prometheus.MustNewConstMetric(descs.cpuSystem, prometheus.GaugeValue, m.cpuSystem, labels...)
		ch <- prometheus.MustNewConstMetric(descs.cpuTotal, prometheus.GaugeValue, m.cpuTotal, labels...)
		ch <- prometheus.MustNewConstMetric(descs.cpus, prometheus.GaugeValue, float64(m.cpus), labels...)
		ch <- prometheus.MustNewConstMetric(descs.cpu_info, prometheus.GaugeValue, 1, slices.Concat(labels,[]string{m.cpu_list})...)
		ch <- prometheus.MustNewConstMetric(descs.memoryRSS, prometheus.GaugeValue, m.memoryRSS, labels...)
		ch <- prometheus.MustNewConstMetric(descs.memoryUsed, prometheus.GaugeValue, m.memoryUsed, labels...)
		ch <- prometheus.MustNewConstMetric(descs.memoryTotal, prometheus.GaugeValue, m.memoryTotal, labels...)
		ch <- prometheus.MustNewConstMetric(descs.memoryCache, prometheus.GaugeValue, m.memoryCache, labels...)
		ch <- prometheus.MustNewConstMetric(descs.memoryFailCount, prometheus.GaugeValue, m.memoryFailCount, labels...)
		ch <- prometheus.MustNewConstMetric(descs.memswUsed, prometheus.GaugeValue, m.memswUsed, labels...)
		ch <- prometheus.MustNewConstMetric(descs.memswTotal, prometheus.GaugeValue, m.memswTotal, labels...)
		// These metrics currently have no cgroup v2 information
		if !e.cgroupv2 {
			ch <- prometheus.MustNewConstMetric(descs.memswFailCount, prometheus.GaugeValue, m.memswFailCount, labels...)
		}
		if m.userslice || m.job {
			ch <- prometheus.MustNewConstMetric(e.info, prometheus.GaugeValue, 1, m.name, m.username, m.uid, m.jobid)
		}
		if *collectProc {
			for exec, count := range m.processExec {
				ch <- prometheus.MustNewConstMetric(descs.processExec, prometheus.GaugeValue, count, slices.Concat(labels,[]string{exec})...)
			}
		}
	}
}

func getProcInfo(pids []int, metric *CgroupMetric, logger *slog.Logger) {
	executables := make(map[string]float64)
	procFS, err := procfs.NewFS(*ProcRoot)
	if err != nil {
		logger.Error("Unable to open procfs", "path", *ProcRoot)
		return
	}
	wg := &sync.WaitGroup{}
	wg.Add(len(pids))
	for _, pid := range pids {
		go func(p int) {
			proc, err := procFS.Proc(p)
			if err != nil {
				logger.Error("Unable to read PID", "pid", p)
				wg.Done()
				return
			}
			executable, err := proc.Executable()
			if err != nil {
				logger.Error("Unable to get executable for PID", "pid", p)
				wg.Done()
				return
			}
			if len(executable) > *collectProcMaxExec {
				logger.Debug("Executable will be truncated", "executable", executable, "len", len(executable), "pid", p)
				trim := *collectProcMaxExec / 2
				executable_prefix := executable[0:trim]
				executable_suffix := executable[len(executable)-trim:]
				executable = fmt.Sprintf("%s...%s", executable_prefix, executable_suffix)
			}
			metricLock.Lock()
			executables[executable] += 1
			metricLock.Unlock()
			wg.Done()
		}(pid)
	}
	wg.Wait()
	metric.processExec = executables
}

func parseCpuSet(cpuset string) ([]string, error) {
	var cpus []string
	var start, end int
	var err error
	if cpuset == "" {
		return nil, nil
	}
	ranges := strings.Split(cpuset, ",")
	for _, r := range ranges {
		boundaries := strings.Split(r, "-")
		if len(boundaries) == 1 {
			start, err = strconv.Atoi(boundaries[0])
			if err != nil {
				return nil, err
			}
			end = start
		} else if len(boundaries) == 2 {
			start, err = strconv.Atoi(boundaries[0])
			if err != nil {
				return nil, err
			}
			end, err = strconv.Atoi(boundaries[1])
			if err != nil {
				return nil, err
			}
		}
		for e := start; e <= end; e++ {
			cpu := strconv.Itoa(e)
			cpus = append(cpus, cpu)
		}
	}
	return cpus, nil
}

func getCPUs(path string, logger *slog.Logger) ([]string, error) {
	if !fileExists(path) {
		return nil, nil
	}
	cpusData, err := os.ReadFile(path)
	if err != nil {
		logger.Error("Error reading cpuset", "cpuset", path, "err", err)
		return nil, err
	}
	cpus, err := parseCpuSet(strings.TrimSuffix(string(cpusData), "\n"))
	if err != nil {
		logger.Error("Error parsing cpu set", "cpuset", path, "err", err)
		return nil, err
	}
	return cpus, nil
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func sliceContains(s interface{}, v interface{}) bool {
	slice := reflect.ValueOf(s)
	for i := 0; i < slice.Len(); i++ {
		if slice.Index(i).Interface() == v {
			return true
		}
	}
	return false
}
