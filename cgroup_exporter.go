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
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/containerd/cgroups"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/promlog/flag"
	"github.com/prometheus/common/version"
	"github.com/prometheus/procfs"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	namespace = "cgroup"
)

var (
	defCgroupRoot          = "/sys/fs/cgroup"
	defProcRoot            = "/proc"
	configPaths            = kingpin.Flag("config.paths", "Comma separated list of cgroup paths to check, eg /user.slice,/system.slice,/slurm").Required().String()
	listenAddress          = kingpin.Flag("web.listen-address", "Address to listen on for web interface and telemetry.").Default(":9306").String()
	disableExporterMetrics = kingpin.Flag("web.disable-exporter-metrics", "Exclude metrics about the exporter (promhttp_*, process_*, go_*)").Default("false").Bool()
	cgroupRoot             = kingpin.Flag("path.cgroup.root", "Root path to cgroup fs").Default(defCgroupRoot).String()
	procRoot               = kingpin.Flag("path.proc.root", "Root path to proc fs").Default(defProcRoot).String()
	collectProc            = kingpin.Flag("collect.proc", "Boolean that sets if to collect proc information").Default("false").Bool()
	collectProcMaxExec     = kingpin.Flag("collect.proc.max-exec", "Max length of process executable to record").Default("100").Int()
	metricLock             = sync.RWMutex{}
)

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
	processExec     map[string]float64
	err             bool
}

type Exporter struct {
	paths           []string
	collectError    *prometheus.Desc
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
	info            *prometheus.Desc
	processExec     *prometheus.Desc
	logger          log.Logger
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

func subsystem() ([]cgroups.Subsystem, error) {
	s := []cgroups.Subsystem{
		cgroups.NewCpuacct(*cgroupRoot),
		cgroups.NewMemory(*cgroupRoot),
	}
	return s, nil
}

func getCPUs(name string, logger log.Logger) ([]string, error) {
	cpusPath := fmt.Sprintf("%s/cpuset%s/cpuset.cpus", *cgroupRoot, name)
	if !fileExists(cpusPath) {
		return nil, nil
	}
	cpusData, err := os.ReadFile(cpusPath)
	if err != nil {
		level.Error(logger).Log("msg", "Error reading cpuset", "cpuset", cpusPath, "err", err)
		return nil, err
	}
	cpus, err := parseCpuSet(strings.TrimSuffix(string(cpusData), "\n"))
	if err != nil {
		level.Error(logger).Log("msg", "Error parsing cpu set", "cpuset", cpusPath, "err", err)
		return nil, err
	}
	return cpus, nil
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

func getInfo(name string, metric *CgroupMetric, logger log.Logger) {
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

func getProcInfo(pids []int, metric *CgroupMetric, logger log.Logger) {
	executables := make(map[string]float64)
	procFS, err := procfs.NewFS(*procRoot)
	if err != nil {
		level.Error(logger).Log("msg", "Unable to open procfs", "path", *procRoot)
		return
	}
	wg := &sync.WaitGroup{}
	wg.Add(len(pids))
	for _, pid := range pids {
		go func(p int) {
			proc, err := procFS.Proc(p)
			if err != nil {
				level.Error(logger).Log("msg", "Unable to read PID", "pid", p)
				wg.Done()
				return
			}
			executable, err := proc.Executable()
			if err != nil {
				level.Error(logger).Log("msg", "Unable to get executable for PID", "pid", p)
				wg.Done()
				return
			}
			if len(executable) > *collectProcMaxExec {
				level.Debug(logger).Log("msg", "Executable will be truncated", "executable", executable, "len", len(executable), "pid", p)
				executable = executable[len(executable)-*collectProcMaxExec:]
				executable = fmt.Sprintf("...%s", executable)
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

func getName(p cgroups.Process, path string, logger log.Logger) (string, error) {
	cpuacctPath := filepath.Join(*cgroupRoot, "cpuacct")
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

func NewExporter(paths []string, logger log.Logger) *Exporter {
	return &Exporter{
		paths: paths,
		cpuUser: prometheus.NewDesc(prometheus.BuildFQName(namespace, "cpu", "user_seconds"),
			"Cumalitive CPU user seconds for cgroup", []string{"cgroup"}, nil),
		cpuSystem: prometheus.NewDesc(prometheus.BuildFQName(namespace, "cpu", "system_seconds"),
			"Cumalitive CPU system seconds for cgroup", []string{"cgroup"}, nil),
		cpuTotal: prometheus.NewDesc(prometheus.BuildFQName(namespace, "cpu", "total_seconds"),
			"Cumalitive CPU total seconds for cgroup", []string{"cgroup"}, nil),
		cpus: prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "cpus"),
			"Number of CPUs in the cgroup", []string{"cgroup"}, nil),
		cpu_info: prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "cpu_info"),
			"Information about the cgroup CPUs", []string{"cgroup", "cpus"}, nil),
		memoryRSS: prometheus.NewDesc(prometheus.BuildFQName(namespace, "memory", "rss_bytes"),
			"Memory RSS used in bytes", []string{"cgroup"}, nil),
		memoryCache: prometheus.NewDesc(prometheus.BuildFQName(namespace, "memory", "cache_bytes"),
			"Memory cache used in bytes", []string{"cgroup"}, nil),
		memoryUsed: prometheus.NewDesc(prometheus.BuildFQName(namespace, "memory", "used_bytes"),
			"Memory used in bytes", []string{"cgroup"}, nil),
		memoryTotal: prometheus.NewDesc(prometheus.BuildFQName(namespace, "memory", "total_bytes"),
			"Memory total given to cgroup in bytes", []string{"cgroup"}, nil),
		memoryFailCount: prometheus.NewDesc(prometheus.BuildFQName(namespace, "memory", "fail_count"),
			"Memory fail count", []string{"cgroup"}, nil),
		memswUsed: prometheus.NewDesc(prometheus.BuildFQName(namespace, "memsw", "used_bytes"),
			"Swap used in bytes", []string{"cgroup"}, nil),
		memswTotal: prometheus.NewDesc(prometheus.BuildFQName(namespace, "memsw", "total_bytes"),
			"Swap total given to cgroup in bytes", []string{"cgroup"}, nil),
		memswFailCount: prometheus.NewDesc(prometheus.BuildFQName(namespace, "memsw", "fail_count"),
			"Swap fail count", []string{"cgroup"}, nil),
		info: prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "info"),
			"User slice information", []string{"cgroup", "username", "uid", "jobid"}, nil),
		processExec: prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "process_exec_count"),
			"Count of instances of a given process", []string{"cgroup", "exec"}, nil),
		collectError: prometheus.NewDesc(prometheus.BuildFQName(namespace, "exporter", "collect_error"),
			"Indicates collection error, 0=no error, 1=error", []string{"cgroup"}, nil),
		logger: logger,
	}
}

func (e *Exporter) getMetrics(name string, pids map[string][]int) (CgroupMetric, error) {
	metric := CgroupMetric{name: name}
	level.Debug(e.logger).Log("msg", "Loading cgroup", "path", name)
	ctrl, err := cgroups.Load(subsystem, func(subsystem cgroups.Name) (string, error) {
		return name, nil
	})
	if err != nil {
		level.Error(e.logger).Log("msg", "Failed to load cgroups", "path", name, "err", err)
		metric.err = true
		return metric, err
	}
	stats, _ := ctrl.Stat(cgroups.IgnoreNotExist)
	metric.cpuUser = float64(stats.CPU.Usage.User) / 1000000000.0
	metric.cpuSystem = float64(stats.CPU.Usage.Kernel) / 1000000000.0
	metric.cpuTotal = float64(stats.CPU.Usage.Total) / 1000000000.0
	metric.memoryRSS = float64(stats.Memory.TotalRSS)
	metric.memoryCache = float64(stats.Memory.TotalCache)
	metric.memoryUsed = float64(stats.Memory.Usage.Usage)
	metric.memoryTotal = float64(stats.Memory.Usage.Limit)
	metric.memoryFailCount = float64(stats.Memory.Usage.Failcnt)
	metric.memswUsed = float64(stats.Memory.Swap.Usage)
	metric.memswTotal = float64(stats.Memory.Swap.Limit)
	metric.memswFailCount = float64(stats.Memory.Swap.Failcnt)
	if cpus, err := getCPUs(name, e.logger); err == nil {
		metric.cpus = len(cpus)
		metric.cpu_list = strings.Join(cpus, ",")
	}
	getInfo(name, &metric, e.logger)
	if *collectProc {
		if val, ok := pids[name]; ok {
			level.Debug(e.logger).Log("msg", "Get process info", "pids", fmt.Sprintf("%v", val))
			getProcInfo(val, &metric, e.logger)
		} else {
			level.Error(e.logger).Log("msg", "Unable to get PIDs", "path", name)
		}
	}
	return metric, nil
}

func (e *Exporter) collect() ([]CgroupMetric, error) {
	var names []string
	var metrics []CgroupMetric
	for _, path := range e.paths {
		level.Debug(e.logger).Log("msg", "Loading cgroup", "path", path)
		control, err := cgroups.Load(subsystem, cgroups.StaticPath(path))
		if err != nil {
			level.Error(e.logger).Log("msg", "Error loading cgroup subsystem", "path", path, "err", err)
			metric := CgroupMetric{name: path, err: true}
			metrics = append(metrics, metric)
			continue
		}
		processes, err := control.Processes(cgroups.Cpuacct, true)
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
			name, err := getName(p, path, e.logger)
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
				metric, _ := e.getMetrics(n, p)
				metricLock.Lock()
				metrics = append(metrics, metric)
				metricLock.Unlock()
				wg.Done()
			}(name, pids)
		}
		wg.Wait()
	}
	return metrics, nil
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.cpuUser
	ch <- e.cpuSystem
	ch <- e.cpuTotal
	ch <- e.cpus
	ch <- e.cpu_info
	ch <- e.memoryRSS
	ch <- e.memoryCache
	ch <- e.memoryUsed
	ch <- e.memoryTotal
	ch <- e.memoryFailCount
	ch <- e.memswUsed
	ch <- e.memswTotal
	ch <- e.memswFailCount
	ch <- e.info
	if *collectProc {
		ch <- e.processExec
	}
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	metrics, _ := e.collect()
	for _, m := range metrics {
		if m.err {
			ch <- prometheus.MustNewConstMetric(e.collectError, prometheus.GaugeValue, 1, m.name)
		}
		ch <- prometheus.MustNewConstMetric(e.cpuUser, prometheus.GaugeValue, m.cpuUser, m.name)
		ch <- prometheus.MustNewConstMetric(e.cpuSystem, prometheus.GaugeValue, m.cpuSystem, m.name)
		ch <- prometheus.MustNewConstMetric(e.cpuTotal, prometheus.GaugeValue, m.cpuTotal, m.name)
		ch <- prometheus.MustNewConstMetric(e.cpus, prometheus.GaugeValue, float64(m.cpus), m.name)
		ch <- prometheus.MustNewConstMetric(e.cpu_info, prometheus.GaugeValue, 1, m.name, m.cpu_list)
		ch <- prometheus.MustNewConstMetric(e.memoryRSS, prometheus.GaugeValue, m.memoryRSS, m.name)
		ch <- prometheus.MustNewConstMetric(e.memoryCache, prometheus.GaugeValue, m.memoryCache, m.name)
		ch <- prometheus.MustNewConstMetric(e.memoryUsed, prometheus.GaugeValue, m.memoryUsed, m.name)
		ch <- prometheus.MustNewConstMetric(e.memoryTotal, prometheus.GaugeValue, m.memoryTotal, m.name)
		ch <- prometheus.MustNewConstMetric(e.memoryFailCount, prometheus.GaugeValue, m.memoryFailCount, m.name)
		ch <- prometheus.MustNewConstMetric(e.memswUsed, prometheus.GaugeValue, m.memswUsed, m.name)
		ch <- prometheus.MustNewConstMetric(e.memswTotal, prometheus.GaugeValue, m.memswTotal, m.name)
		ch <- prometheus.MustNewConstMetric(e.memswFailCount, prometheus.GaugeValue, m.memswFailCount, m.name)
		if m.userslice || m.job {
			ch <- prometheus.MustNewConstMetric(e.info, prometheus.GaugeValue, 1, m.name, m.username, m.uid, m.jobid)
		}
		if *collectProc {
			for exec, count := range m.processExec {
				ch <- prometheus.MustNewConstMetric(e.processExec, prometheus.GaugeValue, count, m.name, exec)
			}
		}
	}
}

func metricsHandler(logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		registry := prometheus.NewRegistry()

		paths := strings.Split(*configPaths, ",")

		exporter := NewExporter(paths, logger)
		registry.MustRegister(exporter)

		gatherers := prometheus.Gatherers{registry}
		if !*disableExporterMetrics {
			gatherers = append(gatherers, prometheus.DefaultGatherer)
		}

		// Delegate http serving to Prometheus client library, which will call collector.Collect.
		h := promhttp.HandlerFor(gatherers, promhttp.HandlerOpts{})
		h.ServeHTTP(w, r)
	}
}

func main() {
	metricsEndpoint := "/metrics"
	promlogConfig := &promlog.Config{}
	flag.AddFlags(kingpin.CommandLine, promlogConfig)
	kingpin.Version(version.Print("cgroup_exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	logger := promlog.New(promlogConfig)
	level.Info(logger).Log("msg", "Starting cgroup_exporter", "version", version.Info())
	level.Info(logger).Log("msg", "Build context", "build_context", version.BuildContext())
	level.Info(logger).Log("msg", "Starting Server", "address", *listenAddress)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		//nolint:errcheck
		w.Write([]byte(`<html>
             <head><title>cgroup Exporter</title></head>
             <body>
             <h1>cgroup Exporter</h1>
             <p><a href='` + metricsEndpoint + `'>Metrics</a></p>
             </body>
             </html>`))
	})
	http.Handle(metricsEndpoint, metricsHandler(logger))
	err := http.ListenAndServe(*listenAddress, nil)
	if err != nil {
		level.Error(logger).Log("err", err)
		os.Exit(1)
	}
}
