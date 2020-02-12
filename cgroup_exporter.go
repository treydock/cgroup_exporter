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
	"io/ioutil"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/containerd/cgroups"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/version"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	namespace = "cgroup"
)

var (
	defCgroupRoot          = "/sys/fs/cgroup"
	configPaths            = kingpin.Flag("config.paths", "Comma separated list of cgroup paths to check, eg /users.slice,/system.slice,/slurm").Required().String()
	listenAddress          = kingpin.Flag("web.listen-address", "Address to listen on for web interface and telemetry.").Default(":9304").String()
	disableExporterMetrics = kingpin.Flag("web.disable-exporter-metrics", "Exclude metrics about the exporter (promhttp_*, process_*, go_*)").Default("false").Bool()
	cgroupRoot             = kingpin.Flag("path.cgroup.root", "Root path to cgroup fs").Default(defCgroupRoot).String()
)

type CgroupMetric struct {
	name        string
	cpuUser     float64
	cpuSystem   float64
	cpuTotal    float64
	cpus        int
	memoryUsed  float64
	uid         string
	username    string
	memoryTotal float64
}

type Exporter struct {
	paths       []string
	cpuUser     *prometheus.Desc
	cpuSystem   *prometheus.Desc
	cpuTotal    *prometheus.Desc
	cpus        *prometheus.Desc
	memoryUsed  *prometheus.Desc
	memoryTotal *prometheus.Desc
	userslice   *prometheus.Desc
	success     *prometheus.Desc
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func sliceContains(slice []string, str string) bool {
	for _, s := range slice {
		if str == s {
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

func getCPUs(name string) (int, error) {
	cpusPath := fmt.Sprintf("%s/cpuset%s/cpuset.cpus", *cgroupRoot, name)
	if !fileExists(cpusPath) {
		return 0, nil
	}
	cpusData, err := ioutil.ReadFile(cpusPath)
	if err != nil {
		log.Errorf("Error reading %s: %s", cpusPath, err.Error())
		return 0, err
	}
	cpus, err := parseCpuSet(string(cpusData))
	if err != nil {
		log.Errorf("Error parsing cpu set %s", err.Error())
		return 0, err
	}
	return cpus, nil
}

func parseCpuSet(cpuset string) (int, error) {
	var cpus int
	if cpuset == "" {
		return 0, nil
	}
	ranges := strings.Split(cpuset, ",")
	for _, r := range ranges {
		boundaries := strings.Split(r, "-")
		if len(boundaries) == 1 {
			cpus++
		} else if len(boundaries) == 2 {
			start, err := strconv.Atoi(boundaries[0])
			if err != nil {
				return 0, err
			}
			end, err := strconv.Atoi(boundaries[1])
			if err != nil {
				return 0, err
			}
			for e := start; e <= end; e++ {
				cpus++
			}
		}
	}
	return cpus, nil
}

func NewExporter(paths []string) *Exporter {
	return &Exporter{
		paths: paths,
		cpuUser: prometheus.NewDesc(prometheus.BuildFQName(namespace, "cpu", "user_seconds"),
			"Cumalitive CPU user seconds for cgroup", []string{"cgroup"}, nil),
		cpuSystem: prometheus.NewDesc(prometheus.BuildFQName(namespace, "cpu", "kernel_seconds"),
			"Cumalitive CPU kernel seconds for cgroup", []string{"cgroup"}, nil),
		cpuTotal: prometheus.NewDesc(prometheus.BuildFQName(namespace, "cpu", "total_seconds"),
			"Cumalitive CPU total seconds for cgroup", []string{"cgroup"}, nil),
		cpus: prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "cpus"),
			"Number of CPUs in the cgroup", []string{"cgroup"}, nil),
		memoryUsed: prometheus.NewDesc(prometheus.BuildFQName(namespace, "memory", "used_bytes"),
			"Memory used in bytes", []string{"cgroup"}, nil),
		memoryTotal: prometheus.NewDesc(prometheus.BuildFQName(namespace, "memory", "total_bytes"),
			"Memory total given to cgroup in bytes", []string{"cgroup"}, nil),
		userslice: prometheus.NewDesc(prometheus.BuildFQName(namespace, "userslice", "info"),
			"User slice information", []string{"cgroup", "username", "uid"}, nil),
		success: prometheus.NewDesc(prometheus.BuildFQName(namespace, "exporter", "success"),
			"Exporter status, 1=successful 0=errors", nil, nil),
	}
}

func (e *Exporter) collect() ([]CgroupMetric, error) {
	var names []string
	var metrics []CgroupMetric
	for _, path := range e.paths {
		control, err := cgroups.Load(subsystem, cgroups.StaticPath(path))
		if err != nil {
			log.Errorf("Error loading cgroup subsystem path %s: %s", path, err.Error())
			return nil, err
		}
		processes, err := control.Processes(cgroups.Cpuacct, true)
		if err != nil {
			log.Errorf("Error loading cgroup processes for path %s: %s", path, err.Error())
			return nil, err
		}
		for _, p := range processes {
			cpuacctPath := filepath.Join(*cgroupRoot, "cpuacct")
			name := strings.TrimPrefix(p.Path, cpuacctPath)
			name = strings.TrimSuffix(name, "/")
			if sliceContains(names, name) {
				continue
			}
			names = append(names, name)
			metric := CgroupMetric{name: name}
			ctrl, err := cgroups.Load(subsystem, func(subsystem cgroups.Name) (string, error) {
				return name, nil
			})
			if err != nil {
				log.Errorf("Failed to load cgroups for %s: %s", name, err.Error())
				return nil, err
			}
			stats, _ := ctrl.Stat(cgroups.IgnoreNotExist)
			metric.cpuUser = float64(stats.CPU.Usage.User) / 1000000000.0
			metric.cpuSystem = float64(stats.CPU.Usage.Kernel) / 1000000000.0
			metric.cpuTotal = float64(stats.CPU.Usage.Total) / 1000000000.0
			metric.memoryUsed = float64(stats.Memory.Usage.Usage)
			metric.memoryTotal = float64(stats.Memory.Usage.Limit)
			if cpus, err := getCPUs(name); err == nil {
				metric.cpus = cpus
			}
			pathBase := filepath.Base(name)
			userSlicePattern := regexp.MustCompile("^user-([0-9]+).slice$")
			match := userSlicePattern.FindStringSubmatch(pathBase)
			if len(match) == 1 {
				metric.uid = match[0]
				user, err := user.LookupId(match[0])
				if err != nil {
					log.Errorf("Error looking up user slice uid %s: %s", match[0], err.Error())
				} else {
					metric.username = user.Name
				}
			}
			metrics = append(metrics, metric)
		}
	}

	return metrics, nil
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.cpuUser
	ch <- e.cpuSystem
	ch <- e.cpuTotal
	ch <- e.cpus
	ch <- e.memoryUsed
	ch <- e.memoryTotal
	ch <- e.success
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	metrics, err := e.collect()
	if err != nil {
		log.Errorf("Exporter error: %s", err.Error())
		ch <- prometheus.MustNewConstMetric(e.success, prometheus.GaugeValue, 0)
	} else {
		ch <- prometheus.MustNewConstMetric(e.success, prometheus.GaugeValue, 1)
	}
	for _, m := range metrics {
		ch <- prometheus.MustNewConstMetric(e.cpuUser, prometheus.GaugeValue, m.cpuUser, m.name)
		ch <- prometheus.MustNewConstMetric(e.cpuSystem, prometheus.GaugeValue, m.cpuSystem, m.name)
		ch <- prometheus.MustNewConstMetric(e.cpuTotal, prometheus.GaugeValue, m.cpuTotal, m.name)
		ch <- prometheus.MustNewConstMetric(e.cpus, prometheus.GaugeValue, float64(m.cpus), m.name)
		ch <- prometheus.MustNewConstMetric(e.memoryUsed, prometheus.GaugeValue, m.memoryUsed, m.name)
		ch <- prometheus.MustNewConstMetric(e.memoryTotal, prometheus.GaugeValue, m.memoryTotal, m.name)
		if m.username != "" {
			ch <- prometheus.MustNewConstMetric(e.userslice, prometheus.GaugeValue, 1, m.name, m.username, m.uid)
		}
	}
}

func metricsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		registry := prometheus.NewRegistry()

		paths := strings.Split(*configPaths, ",")

		exporter := NewExporter(paths)
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
	log.AddFlags(kingpin.CommandLine)
	kingpin.Version(version.Print("cgroup_exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	log.Infoln("Starting cgroup_exporter", version.Info())
	log.Infoln("Build context", version.BuildContext())
	log.Infoln("Starting Server:", *listenAddress)

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
	http.Handle(metricsEndpoint, metricsHandler())
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
