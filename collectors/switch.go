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
	"math"
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
	devices                      *[]InfinibandDevice
	logger                       *slog.Logger
	collector                    string
	Duration                     *prometheus.Desc
	Error                        *prometheus.Desc
	Timeout                      *prometheus.Desc
	PortXmitData                 *prometheus.Desc
	PortRcvData                  *prometheus.Desc
	PortXmitPkts                 *prometheus.Desc
	PortRcvPkts                  *prometheus.Desc
	PortUnicastXmitPkts          *prometheus.Desc
	PortUnicastRcvPkts           *prometheus.Desc
	PortMulticastXmitPkts        *prometheus.Desc
	PortMulticastRcvPkts         *prometheus.Desc
	SymbolErrorCounter           *prometheus.Desc
	LinkErrorRecoveryCounter     *prometheus.Desc
	LinkDownedCounter            *prometheus.Desc
	PortRcvErrors                *prometheus.Desc
	PortRcvRemotePhysicalErrors  *prometheus.Desc
	PortRcvSwitchRelayErrors     *prometheus.Desc
	PortXmitDiscards             *prometheus.Desc
	PortXmitConstraintErrors     *prometheus.Desc
	PortRcvConstraintErrors      *prometheus.Desc
	LocalLinkIntegrityErrors     *prometheus.Desc
	ExcessiveBufferOverrunErrors *prometheus.Desc
	VL15Dropped                  *prometheus.Desc
	PortXmitWait                 *prometheus.Desc
	QP1Dropped                   *prometheus.Desc
	PortLocalPhysicalErrors      *prometheus.Desc
	PortMalformedPktErrors       *prometheus.Desc
	PortBufferOverrunErrors      *prometheus.Desc
	PortDLIDMappingErrors        *prometheus.Desc
	PortVLMappingErrors          *prometheus.Desc
	PortLoopingErrors            *prometheus.Desc
	Rate                         *prometheus.Desc
	RawRate                      *prometheus.Desc
	Uplink                       *prometheus.Desc
	Info                         *prometheus.Desc
	Up                           *prometheus.Desc
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
		Duration: prometheus.NewDesc(prometheus.BuildFQName(namespace, "switch", "collect_duration_seconds"),
			"Time spent collecting metrics for this device, in seconds.", []string{"guid", "collector"}, nil),
		Error: prometheus.NewDesc(prometheus.BuildFQName(namespace, "switch", "collect_error"),
			"1 if the most recent collection for this device errored, 0 otherwise.", []string{"guid", "collector"}, nil),
		Timeout: prometheus.NewDesc(prometheus.BuildFQName(namespace, "switch", "collect_timeout"),
			"1 if the most recent collection for this device timed out, 0 otherwise.", []string{"guid", "collector"}, nil),
		PortXmitData: prometheus.NewDesc(prometheus.BuildFQName(namespace, "switch", "port_transmit_data_bytes_total"),
			"Total data octets transmitted on this port (perfquery PortXmitData scaled to bytes — IB octets are 4-byte words).", labels, nil),
		PortRcvData: prometheus.NewDesc(prometheus.BuildFQName(namespace, "switch", "port_receive_data_bytes_total"),
			"Total data octets received on this port (perfquery PortRcvData scaled to bytes — IB octets are 4-byte words).", labels, nil),
		PortXmitPkts: prometheus.NewDesc(prometheus.BuildFQName(namespace, "switch", "port_transmit_packets_total"),
			"Total packets transmitted on this port (any size, any traffic class).", labels, nil),
		PortRcvPkts: prometheus.NewDesc(prometheus.BuildFQName(namespace, "switch", "port_receive_packets_total"),
			"Total packets received on this port (any size, any traffic class).", labels, nil),
		PortUnicastXmitPkts: prometheus.NewDesc(prometheus.BuildFQName(namespace, "switch", "port_unicast_transmit_packets_total"),
			"Total unicast packets transmitted on this port.", labels, nil),
		PortUnicastRcvPkts: prometheus.NewDesc(prometheus.BuildFQName(namespace, "switch", "port_unicast_receive_packets_total"),
			"Total unicast packets received on this port.", labels, nil),
		PortMulticastXmitPkts: prometheus.NewDesc(prometheus.BuildFQName(namespace, "switch", "port_multicast_transmit_packets_total"),
			"Total multicast packets transmitted on this port.", labels, nil),
		PortMulticastRcvPkts: prometheus.NewDesc(prometheus.BuildFQName(namespace, "switch", "port_multicast_receive_packets_total"),
			"Total multicast packets received on this port.", labels, nil),
		SymbolErrorCounter: prometheus.NewDesc(prometheus.BuildFQName(namespace, "switch", "port_symbol_error_total"),
			"Minor link errors detected on one or more physical lanes (SymbolErrorCounter).", labels, nil),
		LinkErrorRecoveryCounter: prometheus.NewDesc(prometheus.BuildFQName(namespace, "switch", "port_link_error_recovery_total"),
			"Times the link successfully completed the link error recovery process.", labels, nil),
		LinkDownedCounter: prometheus.NewDesc(prometheus.BuildFQName(namespace, "switch", "port_link_downed_total"),
			"Times the link error recovery process failed and the link went down.", labels, nil),
		PortRcvErrors: prometheus.NewDesc(prometheus.BuildFQName(namespace, "switch", "port_receive_errors_total"),
			"Errors detected on receive packets for any reason (PortRcvErrors).", labels, nil),
		PortRcvRemotePhysicalErrors: prometheus.NewDesc(prometheus.BuildFQName(namespace, "switch", "port_receive_remote_physical_errors_total"),
			"Receive errors caused by a remote physical-layer error (e.g. EBP marker).", labels, nil),
		PortRcvSwitchRelayErrors: prometheus.NewDesc(prometheus.BuildFQName(namespace, "switch", "port_receive_switch_relay_errors_total"),
			"Packets dropped during switch routing because no relay path was available.", labels, nil),
		PortXmitDiscards: prometheus.NewDesc(prometheus.BuildFQName(namespace, "switch", "port_transmit_discards_total"),
			"Outbound packets discarded because the port was busy or down (PortXmitDiscards).", labels, nil),
		PortXmitConstraintErrors: prometheus.NewDesc(prometheus.BuildFQName(namespace, "switch", "port_transmit_constraint_errors_total"),
			"Outbound packets discarded because of a partitioning or rate-limit constraint.", labels, nil),
		PortRcvConstraintErrors: prometheus.NewDesc(prometheus.BuildFQName(namespace, "switch", "port_receive_constraint_errors_total"),
			"Inbound packets discarded because of a partitioning or rate-limit constraint.", labels, nil),
		LocalLinkIntegrityErrors: prometheus.NewDesc(prometheus.BuildFQName(namespace, "switch", "port_local_link_integrity_errors_total"),
			"Local link integrity threshold errors (LocalLinkIntegrityErrors).", labels, nil),
		ExcessiveBufferOverrunErrors: prometheus.NewDesc(prometheus.BuildFQName(namespace, "switch", "port_excessive_buffer_overrun_errors_total"),
			"Excessive buffer overrun errors — receive buffer overran the configured threshold.", labels, nil),
		VL15Dropped: prometheus.NewDesc(prometheus.BuildFQName(namespace, "switch", "port_vl15_dropped_total"),
			"Subnet management packets (VL15) dropped because of resource limitations.", labels, nil),
		PortXmitWait: prometheus.NewDesc(prometheus.BuildFQName(namespace, "switch", "port_transmit_wait_total"),
			"Time ticks during which the port had data to transmit but no flow-control credits available — primary congestion signal.", labels, nil),
		QP1Dropped: prometheus.NewDesc(prometheus.BuildFQName(namespace, "switch", "port_qp1_dropped_total"),
			"Subnet management QP1 packets dropped (QP1Dropped).", labels, nil),
		PortLocalPhysicalErrors: prometheus.NewDesc(prometheus.BuildFQName(namespace, "switch", "port_local_physical_errors_total"),
			"Local physical-layer errors detected on inbound traffic (PortLocalPhysicalErrors).", labels, nil),
		PortMalformedPktErrors: prometheus.NewDesc(prometheus.BuildFQName(namespace, "switch", "port_malformed_packet_errors_total"),
			"Inbound packets discarded because they were malformed.", labels, nil),
		PortBufferOverrunErrors: prometheus.NewDesc(prometheus.BuildFQName(namespace, "switch", "port_buffer_overrun_errors_total"),
			"Inbound packets dropped because the receive buffer overran (PortBufferOverrunErrors).", labels, nil),
		PortDLIDMappingErrors: prometheus.NewDesc(prometheus.BuildFQName(namespace, "switch", "port_dlid_mapping_errors_total"),
			"Inbound packets dropped because the destination LID had no valid mapping (PortDLIDMappingErrors).", labels, nil),
		PortVLMappingErrors: prometheus.NewDesc(prometheus.BuildFQName(namespace, "switch", "port_vl_mapping_errors_total"),
			"Inbound packets dropped because the SL→VL mapping was invalid (PortVLMappingErrors).", labels, nil),
		PortLoopingErrors: prometheus.NewDesc(prometheus.BuildFQName(namespace, "switch", "port_looping_errors_total"),
			"Inbound packets dropped because they were detected as looping (PortLoopingErrors).", labels, nil),
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
	ch <- s.PortXmitData
	ch <- s.PortRcvData
	ch <- s.PortXmitPkts
	ch <- s.PortRcvPkts
	ch <- s.PortUnicastXmitPkts
	ch <- s.PortUnicastRcvPkts
	ch <- s.PortMulticastXmitPkts
	ch <- s.PortMulticastRcvPkts
	ch <- s.SymbolErrorCounter
	ch <- s.LinkErrorRecoveryCounter
	ch <- s.LinkDownedCounter
	ch <- s.PortRcvErrors
	ch <- s.PortRcvRemotePhysicalErrors
	ch <- s.PortRcvSwitchRelayErrors
	ch <- s.PortXmitDiscards
	ch <- s.PortXmitConstraintErrors
	ch <- s.PortRcvConstraintErrors
	ch <- s.LocalLinkIntegrityErrors
	ch <- s.ExcessiveBufferOverrunErrors
	ch <- s.VL15Dropped
	ch <- s.PortXmitWait
	ch <- s.QP1Dropped
	ch <- s.PortLocalPhysicalErrors
	ch <- s.PortMalformedPktErrors
	ch <- s.PortBufferOverrunErrors
	ch <- s.PortDLIDMappingErrors
	ch <- s.PortVLMappingErrors
	ch <- s.PortLoopingErrors
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
		switchName := ""
		if c.device.Name != "" {
			switchName = c.device.Name
		}
		if !math.IsNaN(c.PortXmitData) {
			ch <- prometheus.MustNewConstMetric(s.PortXmitData, prometheus.CounterValue, c.PortXmitData, c.device.GUID, c.PortSelect, switchName)
		}
		if !math.IsNaN(c.PortRcvData) {
			ch <- prometheus.MustNewConstMetric(s.PortRcvData, prometheus.CounterValue, c.PortRcvData, c.device.GUID, c.PortSelect, switchName)
		}
		if !math.IsNaN(c.PortXmitPkts) {
			ch <- prometheus.MustNewConstMetric(s.PortXmitPkts, prometheus.CounterValue, c.PortXmitPkts, c.device.GUID, c.PortSelect, switchName)
		}
		if !math.IsNaN(c.PortRcvPkts) {
			ch <- prometheus.MustNewConstMetric(s.PortRcvPkts, prometheus.CounterValue, c.PortRcvPkts, c.device.GUID, c.PortSelect, switchName)
		}
		if !math.IsNaN(c.PortUnicastXmitPkts) {
			ch <- prometheus.MustNewConstMetric(s.PortUnicastXmitPkts, prometheus.CounterValue, c.PortUnicastXmitPkts, c.device.GUID, c.PortSelect, switchName)
		}
		if !math.IsNaN(c.PortUnicastRcvPkts) {
			ch <- prometheus.MustNewConstMetric(s.PortUnicastRcvPkts, prometheus.CounterValue, c.PortUnicastRcvPkts, c.device.GUID, c.PortSelect, switchName)
		}
		if !math.IsNaN(c.PortMulticastXmitPkts) {
			ch <- prometheus.MustNewConstMetric(s.PortMulticastXmitPkts, prometheus.CounterValue, c.PortMulticastXmitPkts, c.device.GUID, c.PortSelect, switchName)
		}
		if !math.IsNaN(c.PortMulticastRcvPkts) {
			ch <- prometheus.MustNewConstMetric(s.PortMulticastRcvPkts, prometheus.CounterValue, c.PortMulticastRcvPkts, c.device.GUID, c.PortSelect, switchName)
		}
		if !math.IsNaN(c.SymbolErrorCounter) {
			ch <- prometheus.MustNewConstMetric(s.SymbolErrorCounter, prometheus.CounterValue, c.SymbolErrorCounter, c.device.GUID, c.PortSelect, switchName)
		}
		if !math.IsNaN(c.LinkErrorRecoveryCounter) {
			ch <- prometheus.MustNewConstMetric(s.LinkErrorRecoveryCounter, prometheus.CounterValue, c.LinkErrorRecoveryCounter, c.device.GUID, c.PortSelect, switchName)
		}
		if !math.IsNaN(c.LinkDownedCounter) {
			ch <- prometheus.MustNewConstMetric(s.LinkDownedCounter, prometheus.CounterValue, c.LinkDownedCounter, c.device.GUID, c.PortSelect, switchName)
		}
		if !math.IsNaN(c.PortRcvErrors) {
			ch <- prometheus.MustNewConstMetric(s.PortRcvErrors, prometheus.CounterValue, c.PortRcvErrors, c.device.GUID, c.PortSelect, switchName)
		}
		if !math.IsNaN(c.PortRcvRemotePhysicalErrors) {
			ch <- prometheus.MustNewConstMetric(s.PortRcvRemotePhysicalErrors, prometheus.CounterValue, c.PortRcvRemotePhysicalErrors, c.device.GUID, c.PortSelect, switchName)
		}
		if !math.IsNaN(c.PortRcvSwitchRelayErrors) {
			ch <- prometheus.MustNewConstMetric(s.PortRcvSwitchRelayErrors, prometheus.CounterValue, c.PortRcvSwitchRelayErrors, c.device.GUID, c.PortSelect, switchName)
		}
		if !math.IsNaN(c.PortXmitDiscards) {
			ch <- prometheus.MustNewConstMetric(s.PortXmitDiscards, prometheus.CounterValue, c.PortXmitDiscards, c.device.GUID, c.PortSelect, switchName)
		}
		if !math.IsNaN(c.PortXmitConstraintErrors) {
			ch <- prometheus.MustNewConstMetric(s.PortXmitConstraintErrors, prometheus.CounterValue, c.PortXmitConstraintErrors, c.device.GUID, c.PortSelect, switchName)
		}
		if !math.IsNaN(c.PortRcvConstraintErrors) {
			ch <- prometheus.MustNewConstMetric(s.PortRcvConstraintErrors, prometheus.CounterValue, c.PortRcvConstraintErrors, c.device.GUID, c.PortSelect, switchName)
		}
		if !math.IsNaN(c.LocalLinkIntegrityErrors) {
			ch <- prometheus.MustNewConstMetric(s.LocalLinkIntegrityErrors, prometheus.CounterValue, c.LocalLinkIntegrityErrors, c.device.GUID, c.PortSelect, switchName)
		}
		if !math.IsNaN(c.ExcessiveBufferOverrunErrors) {
			ch <- prometheus.MustNewConstMetric(s.ExcessiveBufferOverrunErrors, prometheus.CounterValue, c.ExcessiveBufferOverrunErrors, c.device.GUID, c.PortSelect, switchName)
		}
		if !math.IsNaN(c.VL15Dropped) {
			ch <- prometheus.MustNewConstMetric(s.VL15Dropped, prometheus.CounterValue, c.VL15Dropped, c.device.GUID, c.PortSelect, switchName)
		}
		if !math.IsNaN(c.PortXmitWait) {
			ch <- prometheus.MustNewConstMetric(s.PortXmitWait, prometheus.CounterValue, c.PortXmitWait, c.device.GUID, c.PortSelect, switchName)
		}
		if !math.IsNaN(c.QP1Dropped) {
			ch <- prometheus.MustNewConstMetric(s.QP1Dropped, prometheus.CounterValue, c.QP1Dropped, c.device.GUID, c.PortSelect, switchName)
		}
		if !math.IsNaN(c.PortLocalPhysicalErrors) {
			ch <- prometheus.MustNewConstMetric(s.PortLocalPhysicalErrors, prometheus.CounterValue, c.PortLocalPhysicalErrors, c.device.GUID, c.PortSelect, switchName)
		}
		if !math.IsNaN(c.PortMalformedPktErrors) {
			ch <- prometheus.MustNewConstMetric(s.PortMalformedPktErrors, prometheus.CounterValue, c.PortMalformedPktErrors, c.device.GUID, c.PortSelect, switchName)
		}
		if !math.IsNaN(c.PortBufferOverrunErrors) {
			ch <- prometheus.MustNewConstMetric(s.PortBufferOverrunErrors, prometheus.CounterValue, c.PortBufferOverrunErrors, c.device.GUID, c.PortSelect, switchName)
		}
		if !math.IsNaN(c.PortDLIDMappingErrors) {
			ch <- prometheus.MustNewConstMetric(s.PortDLIDMappingErrors, prometheus.CounterValue, c.PortDLIDMappingErrors, c.device.GUID, c.PortSelect, switchName)
		}
		if !math.IsNaN(c.PortVLMappingErrors) {
			ch <- prometheus.MustNewConstMetric(s.PortVLMappingErrors, prometheus.CounterValue, c.PortVLMappingErrors, c.device.GUID, c.PortSelect, switchName)
		}
		if !math.IsNaN(c.PortLoopingErrors) {
			ch <- prometheus.MustNewConstMetric(s.PortLoopingErrors, prometheus.CounterValue, c.PortLoopingErrors, c.device.GUID, c.PortSelect, switchName)
		}
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
