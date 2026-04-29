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
	"math"

	"github.com/prometheus/client_golang/prometheus"
)

// portCounterDef describes one perfquery-derived port counter. It is the
// single source of truth shared between the switch and HCA collectors —
// counters are identical between the two (they come from the same IB
// Performance Management spec); only the subsystem ("switch" vs "hca") and
// the label set differ.
type portCounterDef struct {
	Name string                          // metric name without namespace / subsystem
	Help string                          // Prometheus HELP text
	Get  func(PerfQueryCounters) float64 // value extractor
}

// portCounterDefs is the ordered list of port counters emitted by both
// SwitchCollector and HCACollector. Order here drives the order of
// NewDesc registration; Prometheus output is sorted alphabetically by
// metric name on its own, so the order here is purely for readability.
var portCounterDefs = []portCounterDef{
	{Name: "port_transmit_data_bytes_total",
		Help: "Total data octets transmitted on this port (perfquery PortXmitData scaled to bytes — IB octets are 4-byte words).",
		Get:  func(c PerfQueryCounters) float64 { return c.PortXmitData }},
	{Name: "port_receive_data_bytes_total",
		Help: "Total data octets received on this port (perfquery PortRcvData scaled to bytes — IB octets are 4-byte words).",
		Get:  func(c PerfQueryCounters) float64 { return c.PortRcvData }},
	{Name: "port_transmit_packets_total",
		Help: "Total packets transmitted on this port (any size, any traffic class).",
		Get:  func(c PerfQueryCounters) float64 { return c.PortXmitPkts }},
	{Name: "port_receive_packets_total",
		Help: "Total packets received on this port (any size, any traffic class).",
		Get:  func(c PerfQueryCounters) float64 { return c.PortRcvPkts }},
	{Name: "port_unicast_transmit_packets_total",
		Help: "Total unicast packets transmitted on this port.",
		Get:  func(c PerfQueryCounters) float64 { return c.PortUnicastXmitPkts }},
	{Name: "port_unicast_receive_packets_total",
		Help: "Total unicast packets received on this port.",
		Get:  func(c PerfQueryCounters) float64 { return c.PortUnicastRcvPkts }},
	{Name: "port_multicast_transmit_packets_total",
		Help: "Total multicast packets transmitted on this port.",
		Get:  func(c PerfQueryCounters) float64 { return c.PortMulticastXmitPkts }},
	{Name: "port_multicast_receive_packets_total",
		Help: "Total multicast packets received on this port.",
		Get:  func(c PerfQueryCounters) float64 { return c.PortMulticastRcvPkts }},
	{Name: "port_symbol_error_total",
		Help: "Minor link errors detected on one or more physical lanes (SymbolErrorCounter).",
		Get:  func(c PerfQueryCounters) float64 { return c.SymbolErrorCounter }},
	{Name: "port_link_error_recovery_total",
		Help: "Times the link successfully completed the link error recovery process.",
		Get:  func(c PerfQueryCounters) float64 { return c.LinkErrorRecoveryCounter }},
	{Name: "port_link_downed_total",
		Help: "Times the link error recovery process failed and the link went down.",
		Get:  func(c PerfQueryCounters) float64 { return c.LinkDownedCounter }},
	{Name: "port_receive_errors_total",
		Help: "Errors detected on receive packets for any reason (PortRcvErrors).",
		Get:  func(c PerfQueryCounters) float64 { return c.PortRcvErrors }},
	{Name: "port_receive_remote_physical_errors_total",
		Help: "Receive errors caused by a remote physical-layer error (e.g. EBP marker).",
		Get:  func(c PerfQueryCounters) float64 { return c.PortRcvRemotePhysicalErrors }},
	{Name: "port_receive_switch_relay_errors_total",
		Help: "Packets dropped during switch routing because no relay path was available.",
		Get:  func(c PerfQueryCounters) float64 { return c.PortRcvSwitchRelayErrors }},
	{Name: "port_transmit_discards_total",
		Help: "Outbound packets discarded because the port was busy or down (PortXmitDiscards).",
		Get:  func(c PerfQueryCounters) float64 { return c.PortXmitDiscards }},
	{Name: "port_transmit_constraint_errors_total",
		Help: "Outbound packets discarded because of a partitioning or rate-limit constraint.",
		Get:  func(c PerfQueryCounters) float64 { return c.PortXmitConstraintErrors }},
	{Name: "port_receive_constraint_errors_total",
		Help: "Inbound packets discarded because of a partitioning or rate-limit constraint.",
		Get:  func(c PerfQueryCounters) float64 { return c.PortRcvConstraintErrors }},
	{Name: "port_local_link_integrity_errors_total",
		Help: "Local link integrity threshold errors (LocalLinkIntegrityErrors).",
		Get:  func(c PerfQueryCounters) float64 { return c.LocalLinkIntegrityErrors }},
	{Name: "port_excessive_buffer_overrun_errors_total",
		Help: "Excessive buffer overrun errors — receive buffer overran the configured threshold.",
		Get:  func(c PerfQueryCounters) float64 { return c.ExcessiveBufferOverrunErrors }},
	{Name: "port_vl15_dropped_total",
		Help: "Subnet management packets (VL15) dropped because of resource limitations.",
		Get:  func(c PerfQueryCounters) float64 { return c.VL15Dropped }},
	{Name: "port_transmit_wait_total",
		Help: "Time ticks during which the port had data to transmit but no flow-control credits available — primary congestion signal.",
		Get:  func(c PerfQueryCounters) float64 { return c.PortXmitWait }},
	{Name: "port_qp1_dropped_total",
		Help: "Subnet management QP1 packets dropped (QP1Dropped).",
		Get:  func(c PerfQueryCounters) float64 { return c.QP1Dropped }},
	// PortRcvErrorDetails counters — emitted only when --collector.*.rcv-err
	// -details is enabled (otherwise the values stay NaN and are skipped).
	{Name: "port_local_physical_errors_total",
		Help: "Local physical-layer errors detected on inbound traffic (PortLocalPhysicalErrors).",
		Get:  func(c PerfQueryCounters) float64 { return c.PortLocalPhysicalErrors }},
	{Name: "port_malformed_packet_errors_total",
		Help: "Inbound packets discarded because they were malformed.",
		Get:  func(c PerfQueryCounters) float64 { return c.PortMalformedPktErrors }},
	{Name: "port_buffer_overrun_errors_total",
		Help: "Inbound packets dropped because the receive buffer overran (PortBufferOverrunErrors).",
		Get:  func(c PerfQueryCounters) float64 { return c.PortBufferOverrunErrors }},
	{Name: "port_dlid_mapping_errors_total",
		Help: "Inbound packets dropped because the destination LID had no valid mapping (PortDLIDMappingErrors).",
		Get:  func(c PerfQueryCounters) float64 { return c.PortDLIDMappingErrors }},
	{Name: "port_vl_mapping_errors_total",
		Help: "Inbound packets dropped because the SL→VL mapping was invalid (PortVLMappingErrors).",
		Get:  func(c PerfQueryCounters) float64 { return c.PortVLMappingErrors }},
	{Name: "port_looping_errors_total",
		Help: "Inbound packets dropped because they were detected as looping (PortLoopingErrors).",
		Get:  func(c PerfQueryCounters) float64 { return c.PortLoopingErrors }},
}

// buildPortCounterDescs registers one Prometheus descriptor per entry in
// portCounterDefs under the given subsystem (`switch` or `hca`) and label
// set, returning a map keyed by metric name.
func buildPortCounterDescs(subsystem string, labels []string) map[string]*prometheus.Desc {
	descs := make(map[string]*prometheus.Desc, len(portCounterDefs))
	for _, def := range portCounterDefs {
		descs[def.Name] = prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, def.Name),
			def.Help,
			labels,
			nil,
		)
	}
	return descs
}

// emitPortCounters fans out PerfQueryCounters values onto the metric
// channel using the descriptor map produced by buildPortCounterDescs.
// NaN-valued fields are skipped — that matches the previous explicit
// `if !math.IsNaN(c.X)` checks and lets the rcv-err detail counters
// (which stay NaN unless the rcv-err collector is enabled) be emitted
// uniformly with the rest.
func emitPortCounters(ch chan<- prometheus.Metric, descs map[string]*prometheus.Desc, c PerfQueryCounters, labelValues ...string) {
	for _, def := range portCounterDefs {
		val := def.Get(c)
		if math.IsNaN(val) {
			continue
		}
		desc, ok := descs[def.Name]
		if !ok {
			continue
		}
		ch <- prometheus.MustNewConstMetric(desc, prometheus.CounterValue, val, labelValues...)
	}
}
