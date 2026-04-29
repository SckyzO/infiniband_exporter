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

package collectors

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"log/slog"

	kingpin "github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	CollectHCA       = kingpin.Flag("collector.hca", "Enable the HCA collector").Default("false").Bool()
	hcaCollectBase   = kingpin.Flag("collector.hca.base-metrics", "Collect base metrics").Default("true").Bool()
	hcaCollectRcvErr = kingpin.Flag("collector.hca.rcv-err-details", "Collect Rcv Error Details").Default("false").Bool()
)

type HCACollector struct {
	devices   *[]InfinibandDevice
	logger    *slog.Logger
	collector string
	portDescs map[string]*prometheus.Desc // built from portCounterDefs
	Duration  *prometheus.Desc
	Error     *prometheus.Desc
	Timeout   *prometheus.Desc
	Rate      *prometheus.Desc
	RawRate   *prometheus.Desc
	Uplink    *prometheus.Desc
	Info      *prometheus.Desc
	Up        *prometheus.Desc
}

type HCAMetrics struct {
	duration       float64
	timeout        float64
	error          float64
	rcvErrDuration float64
	rcvErrTimeout  float64
	rcvErrError    float64
}

func NewHCACollector(devices *[]InfinibandDevice, runonce bool, logger *slog.Logger) *HCACollector {
	labels := []string{"guid", "hca", "port", "switch"}
	collector := "hca"
	if runonce {
		collector = "hca-runonce"
	}
	return &HCACollector{
		devices:   devices,
		logger:    logger.With("collector", collector),
		collector: collector,
		portDescs: buildPortCounterDescs("hca", labels),
		Duration: prometheus.NewDesc(prometheus.BuildFQName(namespace, "hca", "collect_duration_seconds"),
			"Time spent collecting metrics for this device, in seconds.", []string{"guid", "collector"}, nil),
		Error: prometheus.NewDesc(prometheus.BuildFQName(namespace, "hca", "collect_error"),
			"1 if the most recent collection for this device errored, 0 otherwise.", []string{"guid", "collector"}, nil),
		Timeout: prometheus.NewDesc(prometheus.BuildFQName(namespace, "hca", "collect_timeout"),
			"1 if the most recent collection for this device timed out, 0 otherwise.", []string{"guid", "collector"}, nil),
		Rate: prometheus.NewDesc(prometheus.BuildFQName(namespace, "hca", "port_rate_bytes_per_second"),
			"Effective HCA port rate in bytes per second (after IB encoding overhead removed).", []string{"guid", "hca"}, nil),
		RawRate: prometheus.NewDesc(prometheus.BuildFQName(namespace, "hca", "port_raw_rate_bytes_per_second"),
			"Raw HCA port rate in bytes per second (signaling rate, before encoding overhead).", []string{"guid", "hca"}, nil),
		Uplink: prometheus.NewDesc(prometheus.BuildFQName(namespace, "hca", "uplink_info"),
			"Constant 1 describing the switch port this HCA port is connected to.", append(labels, []string{"uplink", "uplink_guid", "uplink_type", "uplink_port", "uplink_lid"}...), nil),
		Info: prometheus.NewDesc(prometheus.BuildFQName(namespace, "hca", "info"),
			"Constant 1 carrying HCA identification labels (lid, guid, hca name).", []string{"guid", "hca", "lid"}, nil),
		Up: prometheus.NewDesc(prometheus.BuildFQName(namespace, "hca", "up"),
			"1 if the latest perfquery scrape of this HCA succeeded, 0 otherwise (timeout or error).", []string{"guid", "hca"}, nil),
	}
}

func (h *HCACollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- h.Duration
	ch <- h.Error
	ch <- h.Timeout
	for _, d := range h.portDescs {
		ch <- d
	}
	ch <- h.Rate
	ch <- h.RawRate
	ch <- h.Uplink
	ch <- h.Info
	ch <- h.Up
}

func (h *HCACollector) Collect(ch chan<- prometheus.Metric) {
	collectTime := time.Now()
	counters, metrics, errors, timeouts := h.collect()
	for _, c := range counters {
		emitPortCounters(ch, h.portDescs, c, c.device.GUID, c.device.Name, c.PortSelect, c.device.Switch)
	}
	if *hcaCollectBase {
		for _, device := range *h.devices {
			metric := metrics[device.GUID]
			ch <- prometheus.MustNewConstMetric(h.Rate, prometheus.GaugeValue, device.Rate, device.GUID, device.Name)
			ch <- prometheus.MustNewConstMetric(h.RawRate, prometheus.GaugeValue, device.RawRate, device.GUID, device.Name)
			ch <- prometheus.MustNewConstMetric(h.Info, prometheus.GaugeValue, 1, device.GUID, device.Name, device.LID)
			ch <- prometheus.MustNewConstMetric(h.Duration, prometheus.GaugeValue, metric.duration, device.GUID, h.collector)
			ch <- prometheus.MustNewConstMetric(h.Timeout, prometheus.GaugeValue, metric.timeout, device.GUID, h.collector)
			ch <- prometheus.MustNewConstMetric(h.Error, prometheus.GaugeValue, metric.error, device.GUID, h.collector)
			up := 1 - metric.error - metric.timeout
			ch <- prometheus.MustNewConstMetric(h.Up, prometheus.GaugeValue, up, device.GUID, device.Name)
			for port, uplink := range device.Uplinks {
				// Label order must match h.Uplink desc: guid, hca, port, switch,
				// uplink, uplink_guid, uplink_type, uplink_port, uplink_lid.
				ch <- prometheus.MustNewConstMetric(h.Uplink, prometheus.GaugeValue, 1, device.GUID, device.Name, port, device.Switch, uplink.Name, uplink.GUID, uplink.Type, uplink.PortNumber, uplink.LID)
			}
		}
	}
	if *hcaCollectRcvErr {
		for _, device := range *h.devices {
			metric := metrics[device.GUID]
			ch <- prometheus.MustNewConstMetric(h.Duration, prometheus.GaugeValue, metric.rcvErrDuration, device.GUID, fmt.Sprintf("%s-rcv-err", h.collector))
			ch <- prometheus.MustNewConstMetric(h.Timeout, prometheus.GaugeValue, metric.rcvErrTimeout, device.GUID, fmt.Sprintf("%s-rcv-err", h.collector))
			ch <- prometheus.MustNewConstMetric(h.Error, prometheus.GaugeValue, metric.rcvErrError, device.GUID, fmt.Sprintf("%s-rcv-err", h.collector))
		}
	}
	ch <- prometheus.MustNewConstMetric(collectErrors, prometheus.GaugeValue, errors, h.collector)
	ch <- prometheus.MustNewConstMetric(collecTimeouts, prometheus.GaugeValue, timeouts, h.collector)
	ch <- prometheus.MustNewConstMetric(collectDuration, prometheus.GaugeValue, time.Since(collectTime).Seconds(), h.collector)
	if strings.HasSuffix(h.collector, "-runonce") {
		ch <- prometheus.MustNewConstMetric(lastExecution, prometheus.GaugeValue, float64(time.Now().Unix()), h.collector)
	}
}

func (h *HCACollector) collect() ([]PerfQueryCounters, map[string]HCAMetrics, float64, float64) {
	var counters []PerfQueryCounters
	metrics := make(map[string]HCAMetrics)
	var countersLock sync.Mutex
	// Concurrent goroutines mutate these counters; atomics keep the race
	// detector silent and the totals correct.
	var errors, timeouts uint64
	limit := make(chan int, *maxConcurrent)
	wg := &sync.WaitGroup{}
	for _, device := range *h.devices {
		limit <- 1
		wg.Add(1)
		go func(device InfinibandDevice) {
			defer func() {
				<-limit
				wg.Done()
			}()
			ctxExtended, cancelExtended := context.WithTimeout(context.Background(), *perfqueryTimeout)
			defer cancelExtended()
			ports := getDevicePorts(device.Uplinks)
			perfqueryPorts := strings.Join(ports, ",")
			start := time.Now()
			extendedOut, err := PerfqueryExec(device.GUID, perfqueryPorts, []string{"-l", "-x"}, ctxExtended)
			metric := HCAMetrics{duration: time.Since(start).Seconds()}
			if err == context.DeadlineExceeded {
				metric.timeout = 1
				h.logger.Error("Timeout collecting extended perfquery counters", "guid", device.GUID)
				atomic.AddUint64(&timeouts, 1)
			} else if err != nil {
				metric.error = 1
				h.logger.Error("Error collecting extended perfquery counters", "guid", device.GUID, "err", err)
				atomic.AddUint64(&errors, 1)
			}
			if err != nil {
				return
			}
			deviceCounters, errs := perfqueryParse(device, extendedOut, h.logger)
			atomic.AddUint64(&errors, uint64(errs))
			if *hcaCollectBase {
				h.logger.Debug("Adding parsed counters", "count", len(deviceCounters), "guid", device.GUID, "name", device.Name)
				countersLock.Lock()
				counters = append(counters, deviceCounters...)
				countersLock.Unlock()
			}
			if *hcaCollectRcvErr {
				for _, deviceCounter := range deviceCounters {
					// Per-iteration scope so the timeout context cancels at
					// the end of each port query, not at goroutine return.
					func() {
						ctxRcvErr, cancelRcvErr := context.WithTimeout(context.Background(), *perfqueryTimeout)
						defer cancelRcvErr()
						rcvErrStart := time.Now()
						rcvErrOut, err := PerfqueryExec(device.GUID, deviceCounter.PortSelect, []string{"-E"}, ctxRcvErr)
						metric.rcvErrDuration = time.Since(rcvErrStart).Seconds()
						if err == context.DeadlineExceeded {
							metric.rcvErrTimeout = 1
							h.logger.Error("Timeout collecting rcvErr perfquery counters", "guid", device.GUID)
							atomic.AddUint64(&timeouts, 1)
							return
						} else if err != nil {
							metric.rcvErrError = 1
							h.logger.Error("Error collecting rcvErr perfquery counters", "guid", device.GUID, "err", err)
							atomic.AddUint64(&errors, 1)
							return
						}
						rcvErrCounters, errs := perfqueryParse(device, rcvErrOut, h.logger)
						atomic.AddUint64(&errors, uint64(errs))
						countersLock.Lock()
						counters = append(counters, rcvErrCounters...)
						countersLock.Unlock()
					}()
				}
			}
			countersLock.Lock()
			metrics[device.GUID] = metric
			countersLock.Unlock()
		}(device)
	}
	wg.Wait()
	close(limit)
	return counters, metrics, float64(errors), float64(timeouts)
}
