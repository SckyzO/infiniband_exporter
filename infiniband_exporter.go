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
	"path/filepath"

	kingpin "github.com/alecthomas/kingpin/v2"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/gofrs/flock"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/promlog/flag"
	"github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"
	webflag "github.com/prometheus/exporter-toolkit/web/kingpinflag"

	"github.com/SckyzO/infiniband_exporter/collectors"
)

const (
	metricsEndpoint         = "/metrics"
	internalMetricsEndpoint = "/internal/metrics"
)

var (
	runOnce  = kingpin.Flag("exporter.runonce", "Run exporter once and write metrics to file").Default("false").Bool()
	output   = kingpin.Flag("exporter.output", "Output file to write metrics to when using runonce").Default("").String()
	lockFile = kingpin.Flag("exporter.lockfile", "Lock file path").Default("/tmp/infiniband_exporter.lock").String()
	// /metrics serves only InfiniBand metrics. The Go runtime / process /
	// promhttp self-metrics are exposed on a separate endpoint
	// (/internal/metrics) so users can scrape the exporter's own health
	// at a different cadence — or skip it entirely.
	disableExporterMetrics = kingpin.Flag("web.disable-exporter-metrics", "Disable the "+internalMetricsEndpoint+" endpoint (Go runtime, process, promhttp)").Default("false").Bool()
	toolkitFlags           = webflag.AddFlags(kingpin.CommandLine, ":9315")
)

func setupGathers(runonce bool, logger log.Logger) prometheus.Gatherer {
	registry := prometheus.NewRegistry()

	ibnetdiscoverCollector := collectors.NewIBNetDiscover(runonce, logger)
	registry.MustRegister(ibnetdiscoverCollector)
	switches, hcas, err := ibnetdiscoverCollector.GetPorts()
	if err != nil {
		level.Error(logger).Log("msg", "Error collecting ports with ibnetdiscover", "err", err)
	} else {
		if *collectors.CollectSwitch {
			switchCollector := collectors.NewSwitchCollector(switches, runonce, logger)
			registry.MustRegister(switchCollector)
		}
		if *collectors.CollectIbswinfo {
			ibswinfoCollector := collectors.NewIbswinfoCollector(switches, runonce, logger)
			registry.MustRegister(ibswinfoCollector)
		}
		if *collectors.CollectHCA {
			hcaCollector := collectors.NewHCACollector(hcas, runonce, logger)
			registry.MustRegister(hcaCollector)
		}
	}

	return prometheus.Gatherers{registry}
}

func metricsHandler(logger log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		gatherers := setupGathers(false, logger)

		// Delegate http serving to Prometheus client library, which will call collector.Collect.
		h := promhttp.HandlerFor(gatherers, promhttp.HandlerOpts{})
		h.ServeHTTP(w, r)
	}
}

func writeMetrics(logger log.Logger) error {
	tmp, err := os.CreateTemp(filepath.Dir(*output), filepath.Base(*output))
	if err != nil {
		level.Error(logger).Log("msg", "Unable to create temporary file", "err", err)
		return err
	}
	// Best-effort cleanup; if Rename succeeded the file is already gone
	// and Remove will return ENOENT, which we deliberately ignore.
	defer func() { _ = os.Remove(tmp.Name()) }()
	gatherers := setupGathers(true, logger)
	err = prometheus.WriteToTextfile(tmp.Name(), gatherers)
	if err != nil {
		level.Error(logger).Log("msg", "Error writing Prometheus metrics to file", "path", tmp.Name(), "err", err)
		return err
	}
	err = os.Rename(tmp.Name(), *output)
	if err != nil {
		level.Error(logger).Log("msg", "Error renaming temporary file to output", "tmp", tmp.Name(), "output", *output, "err", err)
		return err
	}
	return nil
}

func run(logger log.Logger) error {
	if *runOnce {
		if *output == "" {
			return fmt.Errorf("Must specify output path when using runonce mode")
		}
		fileLock := flock.New(*lockFile)
		unlocked, err := fileLock.TryLock()
		if err != nil {
			level.Error(logger).Log("msg", "Unable to obtain lock on lock file", "lockfile", *lockFile)
			return err
		}
		if !unlocked {
			return fmt.Errorf("Lock file %s is locked", *lockFile)
		}
		err = writeMetrics(logger)
		if err != nil {
			return err
		}
		return nil
	}
	level.Info(logger).Log("msg", "Starting infiniband_exporter", "version", version.Info())
	level.Info(logger).Log("msg", "Build context", "build_context", version.BuildContext())

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		body := `<html>
             <head><title>InfiniBand Exporter</title></head>
             <body>
             <h1>InfiniBand Exporter</h1>
             <p><a href='` + metricsEndpoint + `'>InfiniBand metrics</a></p>`
		if !*disableExporterMetrics {
			body += `<p><a href='` + internalMetricsEndpoint + `'>Exporter internal metrics (Go runtime, process, promhttp)</a></p>`
		}
		body += `</body></html>`
		//nolint:errcheck
		w.Write([]byte(body))
	})
	http.Handle(metricsEndpoint, metricsHandler(logger))
	if !*disableExporterMetrics {
		// Default registry already has go_*, process_* and promhttp_*
		// collectors registered by client_golang's init.
		http.Handle(internalMetricsEndpoint, promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{}))
	}
	srv := &http.Server{}
	if err := web.ListenAndServe(srv, toolkitFlags, logger); err != nil {
		level.Error(logger).Log("msg", "Error starting HTTP server", "err", err)
		return err
	}
	return nil
}

func main() {
	promlogConfig := &promlog.Config{}
	flag.AddFlags(kingpin.CommandLine, promlogConfig)
	kingpin.Version(version.Print("infiniband_exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	logger := promlog.New(promlogConfig)

	err := run(logger)
	if err != nil {
		level.Error(logger).Log("err", err)
		os.Exit(1)
	}
}
