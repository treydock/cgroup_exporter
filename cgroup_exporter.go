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
	"strings"

	"github.com/alecthomas/kingpin/v2"
	"github.com/containerd/cgroups/v3"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	versionCollector "github.com/prometheus/client_golang/prometheus/collectors/version"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/promlog/flag"
	"github.com/prometheus/common/version"
	"github.com/treydock/cgroup_exporter/collector"
)

var (
	configPaths            = kingpin.Flag("config.paths", "Comma separated list of cgroup paths to check, eg /user.slice,/system.slice,/slurm").Required().String()
	listenAddress          = kingpin.Flag("web.listen-address", "Address to listen on for web interface and telemetry.").Default(":9306").String()
	disableExporterMetrics = kingpin.Flag("web.disable-exporter-metrics", "Exclude metrics about the exporter (promhttp_*, process_*, go_*)").Default("false").Bool()
)

func metricsHandler(logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		registry := prometheus.NewRegistry()

		paths := strings.Split(*configPaths, ",")

		var cgroupV2 bool
		if cgroups.Mode() == cgroups.Unified {
			cgroupV2 = true
		}
		cgroupCollector := collector.NewCgroupCollector(cgroupV2, paths, logger)
		registry.MustRegister(cgroupCollector)
		registry.MustRegister(versionCollector.NewCollector(fmt.Sprintf("%s_exporter", collector.Namespace)))

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
