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
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	kingpin "github.com/alecthomas/kingpin/v2"
	"github.com/gofrs/flock"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promslog"
	"github.com/prometheus/common/promslog/flag"
	"github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"
	webflag "github.com/prometheus/exporter-toolkit/web/kingpinflag"

	ibcollectors "github.com/SckyzO/infiniband_exporter/collectors"
)

const (
	metricsEndpoint = "/metrics"
	healthEndpoint  = "/healthz"
)

var (
	runOnce  = kingpin.Flag("exporter.runonce", "Run exporter once and write metrics to file").Default("false").Bool()
	output   = kingpin.Flag("exporter.output", "Output file to write metrics to when using runonce").Default("").String()
	lockFile = kingpin.Flag("exporter.lockfile", "Lock file path").Default("/tmp/infiniband_exporter.lock").String()
	// When true, the registry skips registering Go runtime / process collectors.
	// build_info is always registered. Filtering of go_*/process_*/promhttp_*
	// at scrape time is left to Prometheus metric_relabel_configs.
	disableExporterMetrics = kingpin.Flag("web.disable-exporter-metrics", "Exclude Go runtime and process metrics from /metrics").Default("false").Bool()
	toolkitFlags           = webflag.AddFlags(kingpin.CommandLine, ":9315")
)

func setupGathers(runonce bool, logger *slog.Logger) prometheus.Gatherer {
	registry := prometheus.NewRegistry()

	// Always expose build_info — surface version, revision, and Go toolchain
	// to operators without requiring a separate scrape job.
	registry.MustRegister(collectors.NewBuildInfoCollector())
	if !runonce && !*disableExporterMetrics {
		// Go and Process collectors are not useful in runonce/textfile mode
		// (the process exits immediately) so we skip them there.
		registry.MustRegister(
			collectors.NewGoCollector(),
			collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		)
	}

	ibnetdiscoverCollector := ibcollectors.NewIBNetDiscover(runonce, logger)
	registry.MustRegister(ibnetdiscoverCollector)
	switches, hcas, err := ibnetdiscoverCollector.GetPorts()
	if err != nil {
		logger.Error("Error collecting ports with ibnetdiscover", "err", err)
	} else {
		if *ibcollectors.CollectSwitch {
			registry.MustRegister(ibcollectors.NewSwitchCollector(switches, runonce, logger))
		}
		if *ibcollectors.CollectIbswinfo {
			registry.MustRegister(ibcollectors.NewIbswinfoCollector(switches, runonce, logger))
		}
		if *ibcollectors.CollectHCA {
			registry.MustRegister(ibcollectors.NewHCACollector(hcas, runonce, logger))
		}
	}

	return registry
}

func metricsHandler(logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		gatherer := setupGathers(false, logger)
		h := promhttp.HandlerFor(gatherer, promhttp.HandlerOpts{
			EnableOpenMetrics: true,
		})
		h.ServeHTTP(w, r)
	}
}

func writeMetrics(logger *slog.Logger) error {
	tmp, err := os.CreateTemp(filepath.Dir(*output), filepath.Base(*output))
	if err != nil {
		logger.Error("Unable to create temporary file", "err", err)
		return err
	}
	// Best-effort cleanup; Rename consumes the tempfile on success.
	defer func() { _ = os.Remove(tmp.Name()) }()
	gatherer := setupGathers(true, logger)
	if err := prometheus.WriteToTextfile(tmp.Name(), gatherer); err != nil {
		logger.Error("Error writing Prometheus metrics to file", "path", tmp.Name(), "err", err)
		return err
	}
	if err := os.Rename(tmp.Name(), *output); err != nil {
		logger.Error("Error renaming temporary file to output", "tmp", tmp.Name(), "output", *output, "err", err)
		return err
	}
	return nil
}

func run(logger *slog.Logger) error {
	if *runOnce {
		if *output == "" {
			return fmt.Errorf("must specify output path when using runonce mode")
		}
		fileLock := flock.New(*lockFile)
		unlocked, err := fileLock.TryLock()
		if err != nil {
			logger.Error("Unable to obtain lock on lock file", "lockfile", *lockFile)
			return err
		}
		if !unlocked {
			return fmt.Errorf("lock file %s is locked", *lockFile)
		}
		return writeMetrics(logger)
	}
	logger.Info("Starting infiniband_exporter", "version", version.Info())
	logger.Info("Build context", "build_context", version.BuildContext())

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html>
             <head><title>InfiniBand Exporter</title></head>
             <body>
             <h1>InfiniBand Exporter</h1>
             <p><a href='` + metricsEndpoint + `'>Metrics</a></p>
             </body>
             </html>`))
	})
	http.Handle(metricsEndpoint, metricsHandler(logger))
	// /healthz returns 200 OK as long as the HTTP server is up. This lets
	// orchestrators (Kubernetes, systemd watchdog) distinguish "exporter
	// process alive" from "InfiniBand fabric reachable".
	http.HandleFunc(healthEndpoint, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	srv := &http.Server{}
	if err := web.ListenAndServe(srv, toolkitFlags, logger); err != nil {
		logger.Error("Error starting HTTP server", "err", err)
		return err
	}
	return nil
}

func main() {
	promslogConfig := &promslog.Config{}
	flag.AddFlags(kingpin.CommandLine, promslogConfig)
	kingpin.Version(version.Print("infiniband_exporter"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	logger := promslog.New(promslogConfig)

	if err := run(logger); err != nil {
		logger.Error("Exporter failed", "err", err)
		os.Exit(1)
	}
}
