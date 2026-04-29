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
	CollectSwitch       = kingpin.Flag("collector.switch", "Enable the switch collector").Default("true").Bool()
	switchCollectBase   = kingpin.Flag("collector.switch.base-metrics", "Collect base metrics").Default("true").Bool()
	switchCollectRcvErr = kingpin.Flag("collector.switch.rcv-err-details", "Collect Rcv Error Details").Default("false").Bool()
)

type SwitchCollector struct {
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

type SwitchMetrics struct {
	duration       float64
	timeout        float64
	error          float64
	rcvErrDuration float64
	rcvErrTimeout  float64
	rcvErrError    float64
}

func NewSwitchCollector(devices *[]InfinibandDevice, runonce bool, logger *slog.Logger) *SwitchCollector {
	labels := []string{"guid", "port", "switch"}
	collector := "switch"
	if runonce {
		collector = "switch-runonce"
	}
	return &SwitchCollector{
		devices:   devices,
		logger:    logger.With("collector", collector),
		collector: collector,
		portDescs: buildPortCounterDescs("switch", labels),
		Duration: prometheus.NewDesc(prometheus.BuildFQName(namespace, "switch", "collect_duration_seconds"),
			"Time spent collecting metrics for this device, in seconds.", []string{"guid", "collector"}, nil),
		Error: prometheus.NewDesc(prometheus.BuildFQName(namespace, "switch", "collect_error"),
			"1 if the most recent collection for this device errored, 0 otherwise.", []string{"guid", "collector"}, nil),
		Timeout: prometheus.NewDesc(prometheus.BuildFQName(namespace, "switch", "collect_timeout"),
			"1 if the most recent collection for this device timed out, 0 otherwise.", []string{"guid", "collector"}, nil),
		Rate: prometheus.NewDesc(prometheus.BuildFQName(namespace, "switch", "port_rate_bytes_per_second"),
			"Effective port rate in bytes per second (after IB encoding overhead removed).", labels, nil),
		RawRate: prometheus.NewDesc(prometheus.BuildFQName(namespace, "switch", "port_raw_rate_bytes_per_second"),
			"Raw port rate in bytes per second (signaling rate, before encoding overhead).", labels, nil),
		Uplink: prometheus.NewDesc(prometheus.BuildFQName(namespace, "switch", "uplink_info"),
			"Constant 1 describing the device connected to this switch port.", append(labels, []string{"uplink", "uplink_guid", "uplink_type", "uplink_port", "uplink_lid"}...), nil),
		Info: prometheus.NewDesc(prometheus.BuildFQName(namespace, "switch", "info"),
			"Constant 1 carrying switch identification labels (lid, guid, switch name)", []string{"guid", "switch", "lid"}, nil),
		Up: prometheus.NewDesc(prometheus.BuildFQName(namespace, "switch", "up"),
			"1 if the latest perfquery scrape of this switch succeeded, 0 otherwise (timeout or error).", []string{"guid", "switch"}, nil),
	}
}

func (s *SwitchCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- s.Duration
	ch <- s.Error
	ch <- s.Timeout
	for _, d := range s.portDescs {
		ch <- d
	}
	ch <- s.Rate
	ch <- s.RawRate
	ch <- s.Uplink
	ch <- s.Info
	ch <- s.Up
}

func (s *SwitchCollector) Collect(ch chan<- prometheus.Metric) {
	collectTime := time.Now()
	counters, metrics, errors, timeouts := s.collect()
	for _, c := range counters {
		emitPortCounters(ch, s.portDescs, c, c.device.GUID, c.PortSelect, c.device.Name)
	}
	if *switchCollectBase {
		for _, device := range *s.devices {
			metric := metrics[device.GUID]
			ch <- prometheus.MustNewConstMetric(s.Info, prometheus.GaugeValue, 1, device.GUID, device.Name, device.LID)
			ch <- prometheus.MustNewConstMetric(s.Duration, prometheus.GaugeValue, metric.duration, device.GUID, s.collector)
			ch <- prometheus.MustNewConstMetric(s.Timeout, prometheus.GaugeValue, metric.timeout, device.GUID, s.collector)
			ch <- prometheus.MustNewConstMetric(s.Error, prometheus.GaugeValue, metric.error, device.GUID, s.collector)
			// up = 1 - (error || timeout). error and timeout are mutually exclusive
			// in collect(), so summing then subtracting from 1 yields 0/1 cleanly.
			up := 1 - metric.error - metric.timeout
			ch <- prometheus.MustNewConstMetric(s.Up, prometheus.GaugeValue, up, device.GUID, device.Name)
			for port, uplink := range device.Uplinks {
				ch <- prometheus.MustNewConstMetric(s.Rate, prometheus.GaugeValue, uplink.Rate, device.GUID, port, device.Name)
				ch <- prometheus.MustNewConstMetric(s.RawRate, prometheus.GaugeValue, uplink.RawRate, device.GUID, port, device.Name)
				ch <- prometheus.MustNewConstMetric(s.Uplink, prometheus.GaugeValue, 1, device.GUID, port, device.Name, uplink.Name, uplink.GUID, uplink.Type, uplink.PortNumber, uplink.LID)
			}
		}
	}
	if *switchCollectRcvErr {
		for _, device := range *s.devices {
			metric := metrics[device.GUID]
			ch <- prometheus.MustNewConstMetric(s.Duration, prometheus.GaugeValue, metric.rcvErrDuration, device.GUID, fmt.Sprintf("%s-rcv-err", s.collector))
			ch <- prometheus.MustNewConstMetric(s.Timeout, prometheus.GaugeValue, metric.rcvErrTimeout, device.GUID, fmt.Sprintf("%s-rcv-err", s.collector))
			ch <- prometheus.MustNewConstMetric(s.Error, prometheus.GaugeValue, metric.rcvErrError, device.GUID, fmt.Sprintf("%s-rcv-err", s.collector))
		}
	}
	ch <- prometheus.MustNewConstMetric(collectErrors, prometheus.GaugeValue, errors, s.collector)
	ch <- prometheus.MustNewConstMetric(collecTimeouts, prometheus.GaugeValue, timeouts, s.collector)
	ch <- prometheus.MustNewConstMetric(collectDuration, prometheus.GaugeValue, time.Since(collectTime).Seconds(), s.collector)
	if strings.HasSuffix(s.collector, "-runonce") {
		ch <- prometheus.MustNewConstMetric(lastExecution, prometheus.GaugeValue, float64(time.Now().Unix()), s.collector)
	}
}

func (s *SwitchCollector) collect() ([]PerfQueryCounters, map[string]SwitchMetrics, float64, float64) {
	var counters []PerfQueryCounters
	metrics := make(map[string]SwitchMetrics)
	var countersLock sync.Mutex
	// errors/timeouts are mutated from N concurrent goroutines (capped by
	// --perfquery.max-concurrent); use atomics to avoid the data race that
	// `go test -race` would otherwise flag.
	var errors, timeouts uint64
	limit := make(chan int, *maxConcurrent)
	wg := &sync.WaitGroup{}
	for _, device := range *s.devices {
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
			metric := SwitchMetrics{duration: time.Since(start).Seconds()}
			if err == context.DeadlineExceeded {
				metric.timeout = 1
				s.logger.Error("Timeout collecting extended perfquery counters", "guid", device.GUID)
				atomic.AddUint64(&timeouts, 1)
			} else if err != nil {
				metric.error = 1
				s.logger.Error("Error collecting extended perfquery counters", "guid", device.GUID, "err", err)
				atomic.AddUint64(&errors, 1)
			}
			if err != nil {
				return
			}
			deviceCounters, errs := perfqueryParse(device, extendedOut, s.logger)
			atomic.AddUint64(&errors, uint64(errs))
			if *switchCollectBase {
				s.logger.Debug("Adding parsed counters", "count", len(deviceCounters), "guid", device.GUID, "name", device.Name)
				countersLock.Lock()
				counters = append(counters, deviceCounters...)
				countersLock.Unlock()
			}
			if *switchCollectRcvErr {
				for _, deviceCounter := range deviceCounters {
					// Wrap each perfquery call in its own scope so the
					// timeout context is cancelled at the end of the
					// iteration, not accumulated until the goroutine
					// returns.
					func() {
						ctxRcvErr, cancelRcvErr := context.WithTimeout(context.Background(), *perfqueryTimeout)
						defer cancelRcvErr()
						rcvErrStart := time.Now()
						rcvErrOut, err := PerfqueryExec(device.GUID, deviceCounter.PortSelect, []string{"-E"}, ctxRcvErr)
						metric.rcvErrDuration = time.Since(rcvErrStart).Seconds()
						if err == context.DeadlineExceeded {
							metric.rcvErrTimeout = 1
							s.logger.Error("Timeout collecting rcvErr perfquery counters", "guid", device.GUID)
							atomic.AddUint64(&timeouts, 1)
							return
						} else if err != nil {
							metric.rcvErrError = 1
							s.logger.Error("Error collecting rcvErr perfquery counters", "guid", device.GUID, "err", err)
							atomic.AddUint64(&errors, 1)
							return
						}
						rcvErrCounters, errs := perfqueryParse(device, rcvErrOut, s.logger)
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
