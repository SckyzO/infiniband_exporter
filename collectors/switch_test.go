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
	"strings"
	"testing"

	"log/slog"

	kingpin "github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestSwitchCollector(t *testing.T) {
	if _, err := kingpin.CommandLine.Parse([]string{}); err != nil {
		t.Fatal(err)
	}
	SetPerfqueryExecs(t, false, false)
	expected := `
		# HELP infiniband_exporter_collect_errors Number of errors that occurred during collection
		# TYPE infiniband_exporter_collect_errors gauge
		infiniband_exporter_collect_errors{collector="switch"} 0
		# HELP infiniband_exporter_collect_timeouts Number of timeouts that occurred during collection
		# TYPE infiniband_exporter_collect_timeouts gauge
		infiniband_exporter_collect_timeouts{collector="switch"} 0
		# HELP infiniband_switch_info Constant 1 carrying switch identification labels (lid, guid, switch name)
		# TYPE infiniband_switch_info gauge
		infiniband_switch_info{guid="0x506b4b03005c2740",lid="2052",switch="iswr0l1"} 1
		infiniband_switch_info{guid="0x7cfe9003009ce5b0",lid="1719",switch="iswr1l1"} 1
		# HELP infiniband_switch_port_excessive_buffer_overrun_errors_total Excessive buffer overrun errors — receive buffer overran the configured threshold.
		# TYPE infiniband_switch_port_excessive_buffer_overrun_errors_total counter
		infiniband_switch_port_excessive_buffer_overrun_errors_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 0
		infiniband_switch_port_excessive_buffer_overrun_errors_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 0
		infiniband_switch_port_excessive_buffer_overrun_errors_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 0
		# HELP infiniband_switch_port_link_downed_total Times the link error recovery process failed and the link went down.
		# TYPE infiniband_switch_port_link_downed_total counter
		infiniband_switch_port_link_downed_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 1
		infiniband_switch_port_link_downed_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 0
		infiniband_switch_port_link_downed_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 0
		# HELP infiniband_switch_port_link_error_recovery_total Times the link successfully completed the link error recovery process.
		# TYPE infiniband_switch_port_link_error_recovery_total counter
		infiniband_switch_port_link_error_recovery_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 0
		infiniband_switch_port_link_error_recovery_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 0
		infiniband_switch_port_link_error_recovery_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 0
		# HELP infiniband_switch_port_local_link_integrity_errors_total Local link integrity threshold errors (LocalLinkIntegrityErrors).
		# TYPE infiniband_switch_port_local_link_integrity_errors_total counter
		infiniband_switch_port_local_link_integrity_errors_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 0
		infiniband_switch_port_local_link_integrity_errors_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 0
		infiniband_switch_port_local_link_integrity_errors_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 0
		# HELP infiniband_switch_port_multicast_receive_packets_total Total multicast packets received on this port.
		# TYPE infiniband_switch_port_multicast_receive_packets_total counter
		infiniband_switch_port_multicast_receive_packets_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 6694940
		infiniband_switch_port_multicast_receive_packets_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 5584846741
		infiniband_switch_port_multicast_receive_packets_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 0
		# HELP infiniband_switch_port_multicast_transmit_packets_total Total multicast packets transmitted on this port.
		# TYPE infiniband_switch_port_multicast_transmit_packets_total counter
		infiniband_switch_port_multicast_transmit_packets_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 5623645694
		infiniband_switch_port_multicast_transmit_packets_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 25038914
		infiniband_switch_port_multicast_transmit_packets_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 0
		# HELP infiniband_switch_port_qp1_dropped_total Subnet management QP1 packets dropped (QP1Dropped).
		# TYPE infiniband_switch_port_qp1_dropped_total counter
		infiniband_switch_port_qp1_dropped_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 0
		infiniband_switch_port_qp1_dropped_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 0
		infiniband_switch_port_qp1_dropped_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 0
		# HELP infiniband_switch_port_rate_bytes_per_second Effective port rate in bytes per second (after IB encoding overhead removed).
		# TYPE infiniband_switch_port_rate_bytes_per_second gauge
		infiniband_switch_port_rate_bytes_per_second{guid="0x506b4b03005c2740",port="35",switch="iswr0l1"} 1.25e+10
		infiniband_switch_port_rate_bytes_per_second{guid="0x7cfe9003009ce5b0",port="10",switch="iswr1l1"} 1.25e+10
		infiniband_switch_port_rate_bytes_per_second{guid="0x7cfe9003009ce5b0",port="11",switch="iswr1l1"} 1.25e+10
		infiniband_switch_port_rate_bytes_per_second{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 1.25e+10
		# HELP infiniband_switch_port_raw_rate_bytes_per_second Raw port rate in bytes per second (signaling rate, before encoding overhead).
		# TYPE infiniband_switch_port_raw_rate_bytes_per_second gauge
		infiniband_switch_port_raw_rate_bytes_per_second{guid="0x506b4b03005c2740",port="35",switch="iswr0l1"} 1.2890625e+10
		infiniband_switch_port_raw_rate_bytes_per_second{guid="0x7cfe9003009ce5b0",port="10",switch="iswr1l1"} 1.2890625e+10
		infiniband_switch_port_raw_rate_bytes_per_second{guid="0x7cfe9003009ce5b0",port="11",switch="iswr1l1"} 1.2890625e+10
		infiniband_switch_port_raw_rate_bytes_per_second{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 1.2890625e+10
		# HELP infiniband_switch_port_receive_constraint_errors_total Inbound packets discarded because of a partitioning or rate-limit constraint.
		# TYPE infiniband_switch_port_receive_constraint_errors_total counter
		infiniband_switch_port_receive_constraint_errors_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 0
		infiniband_switch_port_receive_constraint_errors_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 0
		infiniband_switch_port_receive_constraint_errors_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 0
		# HELP infiniband_switch_port_receive_data_bytes_total Total data octets received on this port (perfquery PortRcvData scaled to bytes — IB octets are 4-byte words).
		# TYPE infiniband_switch_port_receive_data_bytes_total counter
		infiniband_switch_port_receive_data_bytes_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 715049367846516
		infiniband_switch_port_receive_data_bytes_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 49116115103004
		infiniband_switch_port_receive_data_bytes_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 156315219973512
		# HELP infiniband_switch_port_receive_errors_total Errors detected on receive packets for any reason (PortRcvErrors).
		# TYPE infiniband_switch_port_receive_errors_total counter
		infiniband_switch_port_receive_errors_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 0
		infiniband_switch_port_receive_errors_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 0
		infiniband_switch_port_receive_errors_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 0
		# HELP infiniband_switch_port_receive_packets_total Total packets received on this port (any size, any traffic class).
		# TYPE infiniband_switch_port_receive_packets_total counter
		infiniband_switch_port_receive_packets_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 387654829341
		infiniband_switch_port_receive_packets_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 32262508468
		infiniband_switch_port_receive_packets_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 93660802641
		# HELP infiniband_switch_port_receive_remote_physical_errors_total Receive errors caused by a remote physical-layer error (e.g. EBP marker).
		# TYPE infiniband_switch_port_receive_remote_physical_errors_total counter
		infiniband_switch_port_receive_remote_physical_errors_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 0
		infiniband_switch_port_receive_remote_physical_errors_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 0
		infiniband_switch_port_receive_remote_physical_errors_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 0
		# HELP infiniband_switch_port_receive_switch_relay_errors_total Packets dropped during switch routing because no relay path was available.
		# TYPE infiniband_switch_port_receive_switch_relay_errors_total counter
		infiniband_switch_port_receive_switch_relay_errors_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 7
		infiniband_switch_port_receive_switch_relay_errors_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 0
		infiniband_switch_port_receive_switch_relay_errors_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 0
		# HELP infiniband_switch_port_symbol_error_total Minor link errors detected on one or more physical lanes (SymbolErrorCounter).
		# TYPE infiniband_switch_port_symbol_error_total counter
		infiniband_switch_port_symbol_error_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 0
		infiniband_switch_port_symbol_error_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 0
		infiniband_switch_port_symbol_error_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 0
		# HELP infiniband_switch_port_transmit_constraint_errors_total Outbound packets discarded because of a partitioning or rate-limit constraint.
		# TYPE infiniband_switch_port_transmit_constraint_errors_total counter
		infiniband_switch_port_transmit_constraint_errors_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 0
		infiniband_switch_port_transmit_constraint_errors_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 0
		infiniband_switch_port_transmit_constraint_errors_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 0
		# HELP infiniband_switch_port_transmit_data_bytes_total Total data octets transmitted on this port (perfquery PortXmitData scaled to bytes — IB octets are 4-byte words).
		# TYPE infiniband_switch_port_transmit_data_bytes_total counter
		infiniband_switch_port_transmit_data_bytes_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 715166628708940
		infiniband_switch_port_transmit_data_bytes_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 145192107443712
		infiniband_switch_port_transmit_data_bytes_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 104026280056104
		# HELP infiniband_switch_port_transmit_discards_total Outbound packets discarded because the port was busy or down (PortXmitDiscards).
		# TYPE infiniband_switch_port_transmit_discards_total counter
		infiniband_switch_port_transmit_discards_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 20046
		infiniband_switch_port_transmit_discards_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 0
		infiniband_switch_port_transmit_discards_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 0
		# HELP infiniband_switch_port_transmit_packets_total Total packets transmitted on this port (any size, any traffic class).
		# TYPE infiniband_switch_port_transmit_packets_total counter
		infiniband_switch_port_transmit_packets_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 393094651266
		infiniband_switch_port_transmit_packets_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 101733204203
		infiniband_switch_port_transmit_packets_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 122978948297
		# HELP infiniband_switch_port_transmit_wait_total Time ticks during which the port had data to transmit but no flow-control credits available — primary congestion signal.
		# TYPE infiniband_switch_port_transmit_wait_total counter
		infiniband_switch_port_transmit_wait_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 41864608
		infiniband_switch_port_transmit_wait_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 22730501
		infiniband_switch_port_transmit_wait_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 36510964
		# HELP infiniband_switch_port_unicast_receive_packets_total Total unicast packets received on this port.
		# TYPE infiniband_switch_port_unicast_receive_packets_total counter
		infiniband_switch_port_unicast_receive_packets_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 387648134400
		infiniband_switch_port_unicast_receive_packets_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 26677661727
		infiniband_switch_port_unicast_receive_packets_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 93660802641
		# HELP infiniband_switch_port_unicast_transmit_packets_total Total unicast packets transmitted on this port.
		# TYPE infiniband_switch_port_unicast_transmit_packets_total counter
		infiniband_switch_port_unicast_transmit_packets_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 387471005571
		infiniband_switch_port_unicast_transmit_packets_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 101708165289
		infiniband_switch_port_unicast_transmit_packets_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 122978948297
		# HELP infiniband_switch_port_vl15_dropped_total Subnet management packets (VL15) dropped because of resource limitations.
		# TYPE infiniband_switch_port_vl15_dropped_total counter
		infiniband_switch_port_vl15_dropped_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 0
		infiniband_switch_port_vl15_dropped_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 0
		infiniband_switch_port_vl15_dropped_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 0
		# HELP infiniband_switch_uplink_info Constant 1 describing the device connected to this switch port.
		# TYPE infiniband_switch_uplink_info gauge
		infiniband_switch_uplink_info{guid="0x506b4b03005c2740",port="35",switch="iswr0l1",uplink="p0001 HCA-1",uplink_guid="0x506b4b0300cc02a6",uplink_lid="1432",uplink_port="1",uplink_type="CA"} 1
		infiniband_switch_uplink_info{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1",uplink="ib-i1l2s01",uplink_guid="0x7cfe900300b07320",uplink_lid="1516",uplink_port="1",uplink_type="SW"} 1
		infiniband_switch_uplink_info{guid="0x7cfe9003009ce5b0",port="10",switch="iswr1l1",uplink="o0001 HCA-1",uplink_guid="0x7cfe9003003b4bde",uplink_lid="134",uplink_port="1",uplink_type="CA"} 1
		infiniband_switch_uplink_info{guid="0x7cfe9003009ce5b0",port="11",switch="iswr1l1",uplink="o0002 HCA-1",uplink_guid="0x7cfe9003003b4b96",uplink_lid="133",uplink_port="1",uplink_type="CA"} 1
	`
	collector := NewSwitchCollector(&switchDevices, false, slog.New(slog.DiscardHandler))
	gatherers := setupGatherer(collector)
	if val, err := testutil.GatherAndCount(gatherers); err != nil {
		t.Errorf("Unexpected error: %v", err)
	} else if val != 91 {
		t.Errorf("Unexpected collection count %d, expected 91", val)
	}
	if err := testutil.GatherAndCompare(gatherers, strings.NewReader(expected),
		"infiniband_switch_port_excessive_buffer_overrun_errors_total", "infiniband_switch_port_link_downed_total",
		"infiniband_switch_port_link_error_recovery_total", "infiniband_switch_port_local_link_integrity_errors_total",
		"infiniband_switch_port_multicast_receive_packets_total", "infiniband_switch_port_multicast_transmit_packets_total",
		"infiniband_switch_port_qp1_dropped_total", "infiniband_switch_port_receive_constraint_errors_total",
		"infiniband_switch_port_receive_data_bytes_total", "infiniband_switch_port_receive_errors_total",
		"infiniband_switch_port_receive_packets_total", "infiniband_switch_port_receive_remote_physical_errors_total",
		"infiniband_switch_port_receive_switch_relay_errors_total", "infiniband_switch_port_symbol_error_total",
		"infiniband_switch_port_transmit_constraint_errors_total", "infiniband_switch_port_transmit_data_bytes_total",
		"infiniband_switch_port_transmit_discards_total", "infiniband_switch_port_transmit_packets_total",
		"infiniband_switch_port_transmit_wait_total", "infiniband_switch_port_unicast_receive_packets_total",
		"infiniband_switch_port_unicast_transmit_packets_total", "infiniband_switch_port_vl15_dropped_total",
		"infiniband_switch_port_buffer_overrun_errors_total",
		"infiniband_switch_info", "infiniband_switch_port_rate_bytes_per_second", "infiniband_switch_port_raw_rate_bytes_per_second", "infiniband_switch_uplink_info",
		"infiniband_exporter_collect_errors", "infiniband_exporter_collect_timeouts"); err != nil {
		t.Errorf("unexpected collecting result:\n%s", err)
	}
}

func TestSwitchCollectorFull(t *testing.T) {
	if _, err := kingpin.CommandLine.Parse([]string{"--collector.switch.rcv-err-details"}); err != nil {
		t.Fatal(err)
	}
	SetPerfqueryExecs(t, false, false)
	expected := `
		# HELP infiniband_exporter_collect_errors Number of errors that occurred during collection
		# TYPE infiniband_exporter_collect_errors gauge
		infiniband_exporter_collect_errors{collector="switch"} 0
		# HELP infiniband_exporter_collect_timeouts Number of timeouts that occurred during collection
		# TYPE infiniband_exporter_collect_timeouts gauge
		infiniband_exporter_collect_timeouts{collector="switch"} 0
		# HELP infiniband_switch_info Constant 1 carrying switch identification labels (lid, guid, switch name)
		# TYPE infiniband_switch_info gauge
		infiniband_switch_info{guid="0x506b4b03005c2740",lid="2052",switch="iswr0l1"} 1
		infiniband_switch_info{guid="0x7cfe9003009ce5b0",lid="1719",switch="iswr1l1"} 1
		# HELP infiniband_switch_port_buffer_overrun_errors_total Inbound packets dropped because the receive buffer overran (PortBufferOverrunErrors).
		# TYPE infiniband_switch_port_buffer_overrun_errors_total counter
		infiniband_switch_port_buffer_overrun_errors_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 0
		infiniband_switch_port_buffer_overrun_errors_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 0
		infiniband_switch_port_buffer_overrun_errors_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 0
		# HELP infiniband_switch_port_dlid_mapping_errors_total Inbound packets dropped because the destination LID had no valid mapping (PortDLIDMappingErrors).
		# TYPE infiniband_switch_port_dlid_mapping_errors_total counter
		infiniband_switch_port_dlid_mapping_errors_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 0
		infiniband_switch_port_dlid_mapping_errors_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 0
		infiniband_switch_port_dlid_mapping_errors_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 0
		# HELP infiniband_switch_port_excessive_buffer_overrun_errors_total Excessive buffer overrun errors — receive buffer overran the configured threshold.
		# TYPE infiniband_switch_port_excessive_buffer_overrun_errors_total counter
		infiniband_switch_port_excessive_buffer_overrun_errors_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 0
		infiniband_switch_port_excessive_buffer_overrun_errors_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 0
		infiniband_switch_port_excessive_buffer_overrun_errors_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 0
		# HELP infiniband_switch_port_link_downed_total Times the link error recovery process failed and the link went down.
		# TYPE infiniband_switch_port_link_downed_total counter
		infiniband_switch_port_link_downed_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 1
		infiniband_switch_port_link_downed_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 0
		infiniband_switch_port_link_downed_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 0
		# HELP infiniband_switch_port_link_error_recovery_total Times the link successfully completed the link error recovery process.
		# TYPE infiniband_switch_port_link_error_recovery_total counter
		infiniband_switch_port_link_error_recovery_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 0
		infiniband_switch_port_link_error_recovery_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 0
		infiniband_switch_port_link_error_recovery_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 0
		# HELP infiniband_switch_port_local_link_integrity_errors_total Local link integrity threshold errors (LocalLinkIntegrityErrors).
		# TYPE infiniband_switch_port_local_link_integrity_errors_total counter
		infiniband_switch_port_local_link_integrity_errors_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 0
		infiniband_switch_port_local_link_integrity_errors_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 0
		infiniband_switch_port_local_link_integrity_errors_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 0
		# HELP infiniband_switch_port_local_physical_errors_total Local physical-layer errors detected on inbound traffic (PortLocalPhysicalErrors).
		# TYPE infiniband_switch_port_local_physical_errors_total counter
		infiniband_switch_port_local_physical_errors_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 0
		infiniband_switch_port_local_physical_errors_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 0
		infiniband_switch_port_local_physical_errors_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 0
		# HELP infiniband_switch_port_looping_errors_total Inbound packets dropped because they were detected as looping (PortLoopingErrors).
		# TYPE infiniband_switch_port_looping_errors_total counter
		infiniband_switch_port_looping_errors_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 0
		infiniband_switch_port_looping_errors_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 0
		infiniband_switch_port_looping_errors_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 0
		# HELP infiniband_switch_port_malformed_packet_errors_total Inbound packets discarded because they were malformed.
		# TYPE infiniband_switch_port_malformed_packet_errors_total counter
		infiniband_switch_port_malformed_packet_errors_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 0
		infiniband_switch_port_malformed_packet_errors_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 0
		infiniband_switch_port_malformed_packet_errors_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 0
		# HELP infiniband_switch_port_multicast_receive_packets_total Total multicast packets received on this port.
		# TYPE infiniband_switch_port_multicast_receive_packets_total counter
		infiniband_switch_port_multicast_receive_packets_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 6694940
		infiniband_switch_port_multicast_receive_packets_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 5584846741
		infiniband_switch_port_multicast_receive_packets_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 0
		# HELP infiniband_switch_port_multicast_transmit_packets_total Total multicast packets transmitted on this port.
		# TYPE infiniband_switch_port_multicast_transmit_packets_total counter
		infiniband_switch_port_multicast_transmit_packets_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 5623645694
		infiniband_switch_port_multicast_transmit_packets_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 25038914
		infiniband_switch_port_multicast_transmit_packets_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 0
		# HELP infiniband_switch_port_qp1_dropped_total Subnet management QP1 packets dropped (QP1Dropped).
		# TYPE infiniband_switch_port_qp1_dropped_total counter
		infiniband_switch_port_qp1_dropped_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 0
		infiniband_switch_port_qp1_dropped_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 0
		infiniband_switch_port_qp1_dropped_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 0
		# HELP infiniband_switch_port_rate_bytes_per_second Effective port rate in bytes per second (after IB encoding overhead removed).
		# TYPE infiniband_switch_port_rate_bytes_per_second gauge
		infiniband_switch_port_rate_bytes_per_second{guid="0x506b4b03005c2740",port="35",switch="iswr0l1"} 1.25e+10
		infiniband_switch_port_rate_bytes_per_second{guid="0x7cfe9003009ce5b0",port="10",switch="iswr1l1"} 1.25e+10
		infiniband_switch_port_rate_bytes_per_second{guid="0x7cfe9003009ce5b0",port="11",switch="iswr1l1"} 1.25e+10
		infiniband_switch_port_rate_bytes_per_second{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 1.25e+10
		# HELP infiniband_switch_port_raw_rate_bytes_per_second Raw port rate in bytes per second (signaling rate, before encoding overhead).
		# TYPE infiniband_switch_port_raw_rate_bytes_per_second gauge
		infiniband_switch_port_raw_rate_bytes_per_second{guid="0x506b4b03005c2740",port="35",switch="iswr0l1"} 1.2890625e+10
		infiniband_switch_port_raw_rate_bytes_per_second{guid="0x7cfe9003009ce5b0",port="10",switch="iswr1l1"} 1.2890625e+10
		infiniband_switch_port_raw_rate_bytes_per_second{guid="0x7cfe9003009ce5b0",port="11",switch="iswr1l1"} 1.2890625e+10
		infiniband_switch_port_raw_rate_bytes_per_second{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 1.2890625e+10
		# HELP infiniband_switch_port_receive_constraint_errors_total Inbound packets discarded because of a partitioning or rate-limit constraint.
		# TYPE infiniband_switch_port_receive_constraint_errors_total counter
		infiniband_switch_port_receive_constraint_errors_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 0
		infiniband_switch_port_receive_constraint_errors_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 0
		infiniband_switch_port_receive_constraint_errors_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 0
		# HELP infiniband_switch_port_receive_data_bytes_total Total data octets received on this port (perfquery PortRcvData scaled to bytes — IB octets are 4-byte words).
		# TYPE infiniband_switch_port_receive_data_bytes_total counter
		infiniband_switch_port_receive_data_bytes_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 715049367846516
		infiniband_switch_port_receive_data_bytes_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 49116115103004
		infiniband_switch_port_receive_data_bytes_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 156315219973512
		# HELP infiniband_switch_port_receive_errors_total Errors detected on receive packets for any reason (PortRcvErrors).
		# TYPE infiniband_switch_port_receive_errors_total counter
		infiniband_switch_port_receive_errors_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 0
		infiniband_switch_port_receive_errors_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 0
		infiniband_switch_port_receive_errors_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 0
		# HELP infiniband_switch_port_receive_packets_total Total packets received on this port (any size, any traffic class).
		# TYPE infiniband_switch_port_receive_packets_total counter
		infiniband_switch_port_receive_packets_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 387654829341
		infiniband_switch_port_receive_packets_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 32262508468
		infiniband_switch_port_receive_packets_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 93660802641
		# HELP infiniband_switch_port_receive_remote_physical_errors_total Receive errors caused by a remote physical-layer error (e.g. EBP marker).
		# TYPE infiniband_switch_port_receive_remote_physical_errors_total counter
		infiniband_switch_port_receive_remote_physical_errors_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 0
		infiniband_switch_port_receive_remote_physical_errors_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 0
		infiniband_switch_port_receive_remote_physical_errors_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 0
		# HELP infiniband_switch_port_receive_switch_relay_errors_total Packets dropped during switch routing because no relay path was available.
		# TYPE infiniband_switch_port_receive_switch_relay_errors_total counter
		infiniband_switch_port_receive_switch_relay_errors_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 7
		infiniband_switch_port_receive_switch_relay_errors_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 0
		infiniband_switch_port_receive_switch_relay_errors_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 0
		# HELP infiniband_switch_port_symbol_error_total Minor link errors detected on one or more physical lanes (SymbolErrorCounter).
		# TYPE infiniband_switch_port_symbol_error_total counter
		infiniband_switch_port_symbol_error_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 0
		infiniband_switch_port_symbol_error_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 0
		infiniband_switch_port_symbol_error_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 0
		# HELP infiniband_switch_port_transmit_constraint_errors_total Outbound packets discarded because of a partitioning or rate-limit constraint.
		# TYPE infiniband_switch_port_transmit_constraint_errors_total counter
		infiniband_switch_port_transmit_constraint_errors_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 0
		infiniband_switch_port_transmit_constraint_errors_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 0
		infiniband_switch_port_transmit_constraint_errors_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 0
		# HELP infiniband_switch_port_transmit_data_bytes_total Total data octets transmitted on this port (perfquery PortXmitData scaled to bytes — IB octets are 4-byte words).
		# TYPE infiniband_switch_port_transmit_data_bytes_total counter
		infiniband_switch_port_transmit_data_bytes_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 715166628708940
		infiniband_switch_port_transmit_data_bytes_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 145192107443712
		infiniband_switch_port_transmit_data_bytes_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 104026280056104
		# HELP infiniband_switch_port_transmit_discards_total Outbound packets discarded because the port was busy or down (PortXmitDiscards).
		# TYPE infiniband_switch_port_transmit_discards_total counter
		infiniband_switch_port_transmit_discards_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 20046
		infiniband_switch_port_transmit_discards_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 0
		infiniband_switch_port_transmit_discards_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 0
		# HELP infiniband_switch_port_transmit_packets_total Total packets transmitted on this port (any size, any traffic class).
		# TYPE infiniband_switch_port_transmit_packets_total counter
		infiniband_switch_port_transmit_packets_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 393094651266
		infiniband_switch_port_transmit_packets_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 101733204203
		infiniband_switch_port_transmit_packets_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 122978948297
		# HELP infiniband_switch_port_transmit_wait_total Time ticks during which the port had data to transmit but no flow-control credits available — primary congestion signal.
		# TYPE infiniband_switch_port_transmit_wait_total counter
		infiniband_switch_port_transmit_wait_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 41864608
		infiniband_switch_port_transmit_wait_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 22730501
		infiniband_switch_port_transmit_wait_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 36510964
		# HELP infiniband_switch_port_unicast_receive_packets_total Total unicast packets received on this port.
		# TYPE infiniband_switch_port_unicast_receive_packets_total counter
		infiniband_switch_port_unicast_receive_packets_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 387648134400
		infiniband_switch_port_unicast_receive_packets_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 26677661727
		infiniband_switch_port_unicast_receive_packets_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 93660802641
		# HELP infiniband_switch_port_unicast_transmit_packets_total Total unicast packets transmitted on this port.
		# TYPE infiniband_switch_port_unicast_transmit_packets_total counter
		infiniband_switch_port_unicast_transmit_packets_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 387471005571
		infiniband_switch_port_unicast_transmit_packets_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 101708165289
		infiniband_switch_port_unicast_transmit_packets_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 122978948297
		# HELP infiniband_switch_port_vl_mapping_errors_total Inbound packets dropped because the SL→VL mapping was invalid (PortVLMappingErrors).
		# TYPE infiniband_switch_port_vl_mapping_errors_total counter
		infiniband_switch_port_vl_mapping_errors_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 0
		infiniband_switch_port_vl_mapping_errors_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 0
		infiniband_switch_port_vl_mapping_errors_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 0
		# HELP infiniband_switch_port_vl15_dropped_total Subnet management packets (VL15) dropped because of resource limitations.
		# TYPE infiniband_switch_port_vl15_dropped_total counter
		infiniband_switch_port_vl15_dropped_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 0
		infiniband_switch_port_vl15_dropped_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 0
		infiniband_switch_port_vl15_dropped_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 0
		# HELP infiniband_switch_uplink_info Constant 1 describing the device connected to this switch port.
		# TYPE infiniband_switch_uplink_info gauge
		infiniband_switch_uplink_info{guid="0x506b4b03005c2740",port="35",switch="iswr0l1",uplink="p0001 HCA-1",uplink_guid="0x506b4b0300cc02a6",uplink_lid="1432",uplink_port="1",uplink_type="CA"} 1
		infiniband_switch_uplink_info{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1",uplink="ib-i1l2s01",uplink_guid="0x7cfe900300b07320",uplink_lid="1516",uplink_port="1",uplink_type="SW"} 1
		infiniband_switch_uplink_info{guid="0x7cfe9003009ce5b0",port="10",switch="iswr1l1",uplink="o0001 HCA-1",uplink_guid="0x7cfe9003003b4bde",uplink_lid="134",uplink_port="1",uplink_type="CA"} 1
		infiniband_switch_uplink_info{guid="0x7cfe9003009ce5b0",port="11",switch="iswr1l1",uplink="o0002 HCA-1",uplink_guid="0x7cfe9003003b4b96",uplink_lid="133",uplink_port="1",uplink_type="CA"} 1
	`
	collector := NewSwitchCollector(&switchDevices, false, slog.New(slog.DiscardHandler))
	gatherers := setupGatherer(collector)
	if val, err := testutil.GatherAndCount(gatherers); err != nil {
		t.Errorf("Unexpected error: %v", err)
	} else if val != 115 {
		t.Errorf("Unexpected collection count %d, expected 115", val)
	}
	if err := testutil.GatherAndCompare(gatherers, strings.NewReader(expected),
		"infiniband_switch_port_excessive_buffer_overrun_errors_total", "infiniband_switch_port_link_downed_total",
		"infiniband_switch_port_link_error_recovery_total", "infiniband_switch_port_local_link_integrity_errors_total",
		"infiniband_switch_port_multicast_receive_packets_total", "infiniband_switch_port_multicast_transmit_packets_total",
		"infiniband_switch_port_qp1_dropped_total", "infiniband_switch_port_receive_constraint_errors_total",
		"infiniband_switch_port_receive_data_bytes_total", "infiniband_switch_port_receive_errors_total",
		"infiniband_switch_port_receive_packets_total", "infiniband_switch_port_receive_remote_physical_errors_total",
		"infiniband_switch_port_receive_switch_relay_errors_total", "infiniband_switch_port_symbol_error_total",
		"infiniband_switch_port_transmit_constraint_errors_total", "infiniband_switch_port_transmit_data_bytes_total",
		"infiniband_switch_port_transmit_discards_total", "infiniband_switch_port_transmit_packets_total",
		"infiniband_switch_port_transmit_wait_total", "infiniband_switch_port_unicast_receive_packets_total",
		"infiniband_switch_port_unicast_transmit_packets_total", "infiniband_switch_port_vl15_dropped_total",
		"infiniband_switch_port_buffer_overrun_errors_total", "infiniband_switch_port_dlid_mapping_errors_total",
		"infiniband_switch_port_local_physical_errors_total", "infiniband_switch_port_looping_errors_total",
		"infiniband_switch_port_malformed_packet_errors_total", "infiniband_switch_port_vl_mapping_errors_total",
		"infiniband_switch_info", "infiniband_switch_port_rate_bytes_per_second", "infiniband_switch_port_raw_rate_bytes_per_second", "infiniband_switch_uplink_info",
		"infiniband_exporter_collect_errors", "infiniband_exporter_collect_timeouts"); err != nil {
		t.Errorf("unexpected collecting result:\n%s", err)
	}
}

func TestSwitchCollectorNoBase(t *testing.T) {
	if _, err := kingpin.CommandLine.Parse([]string{"--no-collector.switch.base-metrics", "--collector.switch.rcv-err-details"}); err != nil {
		t.Fatal(err)
	}
	SetPerfqueryExecs(t, false, false)
	expected := `
		# HELP infiniband_exporter_collect_errors Number of errors that occurred during collection
		# TYPE infiniband_exporter_collect_errors gauge
		infiniband_exporter_collect_errors{collector="switch"} 0
		# HELP infiniband_exporter_collect_timeouts Number of timeouts that occurred during collection
		# TYPE infiniband_exporter_collect_timeouts gauge
		infiniband_exporter_collect_timeouts{collector="switch"} 0
		# HELP infiniband_switch_port_buffer_overrun_errors_total Inbound packets dropped because the receive buffer overran (PortBufferOverrunErrors).
		# TYPE infiniband_switch_port_buffer_overrun_errors_total counter
		infiniband_switch_port_buffer_overrun_errors_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 0
		infiniband_switch_port_buffer_overrun_errors_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 0
		infiniband_switch_port_buffer_overrun_errors_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 0
		# HELP infiniband_switch_port_dlid_mapping_errors_total Inbound packets dropped because the destination LID had no valid mapping (PortDLIDMappingErrors).
		# TYPE infiniband_switch_port_dlid_mapping_errors_total counter
		infiniband_switch_port_dlid_mapping_errors_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 0
		infiniband_switch_port_dlid_mapping_errors_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 0
		infiniband_switch_port_dlid_mapping_errors_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 0
		# HELP infiniband_switch_port_local_physical_errors_total Local physical-layer errors detected on inbound traffic (PortLocalPhysicalErrors).
		# TYPE infiniband_switch_port_local_physical_errors_total counter
		infiniband_switch_port_local_physical_errors_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 0
		infiniband_switch_port_local_physical_errors_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 0
		infiniband_switch_port_local_physical_errors_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 0
		# HELP infiniband_switch_port_looping_errors_total Inbound packets dropped because they were detected as looping (PortLoopingErrors).
		# TYPE infiniband_switch_port_looping_errors_total counter
		infiniband_switch_port_looping_errors_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 0
		infiniband_switch_port_looping_errors_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 0
		infiniband_switch_port_looping_errors_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 0
		# HELP infiniband_switch_port_malformed_packet_errors_total Inbound packets discarded because they were malformed.
		# TYPE infiniband_switch_port_malformed_packet_errors_total counter
		infiniband_switch_port_malformed_packet_errors_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 0
		infiniband_switch_port_malformed_packet_errors_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 0
		infiniband_switch_port_malformed_packet_errors_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 0
		# HELP infiniband_switch_port_vl_mapping_errors_total Inbound packets dropped because the SL→VL mapping was invalid (PortVLMappingErrors).
		# TYPE infiniband_switch_port_vl_mapping_errors_total counter
		infiniband_switch_port_vl_mapping_errors_total{guid="0x506b4b03005c2740",port="1",switch="iswr0l1"} 0
		infiniband_switch_port_vl_mapping_errors_total{guid="0x7cfe9003009ce5b0",port="1",switch="iswr1l1"} 0
		infiniband_switch_port_vl_mapping_errors_total{guid="0x7cfe9003009ce5b0",port="2",switch="iswr1l1"} 0
	`
	collector := NewSwitchCollector(&switchDevices, false, slog.New(slog.DiscardHandler))
	gatherers := setupGatherer(collector)
	if val, err := testutil.GatherAndCount(gatherers); err != nil {
		t.Errorf("Unexpected error: %v", err)
	} else if val != 27 {
		t.Errorf("Unexpected collection count %d, expected 27", val)
	}
	if err := testutil.GatherAndCompare(gatherers, strings.NewReader(expected),
		"infiniband_switch_port_excessive_buffer_overrun_errors_total", "infiniband_switch_port_link_downed_total",
		"infiniband_switch_port_link_error_recovery_total", "infiniband_switch_port_local_link_integrity_errors_total",
		"infiniband_switch_port_multicast_receive_packets_total", "infiniband_switch_port_multicast_transmit_packets_total",
		"infiniband_switch_port_qp1_dropped_total", "infiniband_switch_port_receive_constraint_errors_total",
		"infiniband_switch_port_receive_data_bytes_total", "infiniband_switch_port_receive_errors_total",
		"infiniband_switch_port_receive_packets_total", "infiniband_switch_port_receive_remote_physical_errors_total",
		"infiniband_switch_port_receive_switch_relay_errors_total", "infiniband_switch_port_symbol_error_total",
		"infiniband_switch_port_transmit_constraint_errors_total", "infiniband_switch_port_transmit_data_bytes_total",
		"infiniband_switch_port_transmit_discards_total", "infiniband_switch_port_transmit_packets_total",
		"infiniband_switch_port_transmit_wait_total", "infiniband_switch_port_unicast_receive_packets_total",
		"infiniband_switch_port_unicast_transmit_packets_total", "infiniband_switch_port_vl15_dropped_total",
		"infiniband_switch_port_buffer_overrun_errors_total", "infiniband_switch_port_dlid_mapping_errors_total",
		"infiniband_switch_port_local_physical_errors_total", "infiniband_switch_port_looping_errors_total",
		"infiniband_switch_port_malformed_packet_errors_total", "infiniband_switch_port_vl_mapping_errors_total",
		"infiniband_switch_info", "infiniband_switch_port_rate_bytes_per_second", "infiniband_switch_raw_port_rate_bytes_per_second", "infiniband_switch_uplink_info",
		"infiniband_exporter_collect_errors", "infiniband_exporter_collect_timeouts"); err != nil {
		t.Errorf("unexpected collecting result:\n%s", err)
	}
}

func TestSwitchCollectorError(t *testing.T) {
	if _, err := kingpin.CommandLine.Parse([]string{}); err != nil {
		t.Fatal(err)
	}
	SetPerfqueryExecs(t, true, false)
	expected := `
		# HELP infiniband_exporter_collect_errors Number of errors that occurred during collection
		# TYPE infiniband_exporter_collect_errors gauge
		infiniband_exporter_collect_errors{collector="switch"} 2
		# HELP infiniband_exporter_collect_timeouts Number of timeouts that occurred during collection
		# TYPE infiniband_exporter_collect_timeouts gauge
		infiniband_exporter_collect_timeouts{collector="switch"} 0
	`
	collector := NewSwitchCollector(&switchDevices, false, slog.New(slog.DiscardHandler))
	gatherers := setupGatherer(collector)
	if val, err := testutil.GatherAndCount(gatherers); err != nil {
		t.Errorf("Unexpected error: %v", err)
	} else if val != 25 {
		t.Errorf("Unexpected collection count %d, expected 25", val)
	}
	if err := testutil.GatherAndCompare(gatherers, strings.NewReader(expected),
		"infiniband_switch_port_excessive_buffer_overrun_errors_total", "infiniband_switch_port_link_downed_total",
		"infiniband_switch_port_link_error_recovery_total", "infiniband_switch_port_local_link_integrity_errors_total",
		"infiniband_exporter_collect_errors", "infiniband_exporter_collect_timeouts"); err != nil {
		t.Errorf("unexpected collecting result:\n%s", err)
	}
}

func TestSwitchCollectorErrorRunonce(t *testing.T) {
	if _, err := kingpin.CommandLine.Parse([]string{}); err != nil {
		t.Fatal(err)
	}
	SetPerfqueryExecs(t, true, false)
	expected := `
		# HELP infiniband_exporter_collect_errors Number of errors that occurred during collection
		# TYPE infiniband_exporter_collect_errors gauge
		infiniband_exporter_collect_errors{collector="switch-runonce"} 2
		# HELP infiniband_exporter_collect_timeouts Number of timeouts that occurred during collection
		# TYPE infiniband_exporter_collect_timeouts gauge
		infiniband_exporter_collect_timeouts{collector="switch-runonce"} 0
	`
	collector := NewSwitchCollector(&switchDevices, true, slog.New(slog.DiscardHandler))
	gatherers := setupGatherer(collector)
	if val, err := testutil.GatherAndCount(gatherers); err != nil {
		t.Errorf("Unexpected error: %v", err)
	} else if val != 26 {
		t.Errorf("Unexpected collection count %d, expected 26", val)
	}
	if err := testutil.GatherAndCompare(gatherers, strings.NewReader(expected),
		"infiniband_switch_port_excessive_buffer_overrun_errors_total", "infiniband_switch_port_link_downed_total",
		"infiniband_switch_port_link_error_recovery_total", "infiniband_switch_port_local_link_integrity_errors_total",
		"infiniband_exporter_collect_errors", "infiniband_exporter_collect_timeouts"); err != nil {
		t.Errorf("unexpected collecting result:\n%s", err)
	}
}

func TestSwitchCollectorTimeout(t *testing.T) {
	if _, err := kingpin.CommandLine.Parse([]string{}); err != nil {
		t.Fatal(err)
	}
	SetPerfqueryExecs(t, false, true)
	expected := `
		# HELP infiniband_exporter_collect_errors Number of errors that occurred during collection
		# TYPE infiniband_exporter_collect_errors gauge
		infiniband_exporter_collect_errors{collector="switch"} 0
		# HELP infiniband_exporter_collect_timeouts Number of timeouts that occurred during collection
		# TYPE infiniband_exporter_collect_timeouts gauge
		infiniband_exporter_collect_timeouts{collector="switch"} 2
	`
	collector := NewSwitchCollector(&switchDevices, false, slog.New(slog.DiscardHandler))
	gatherers := setupGatherer(collector)
	if val, err := testutil.GatherAndCount(gatherers); err != nil {
		t.Errorf("Unexpected error: %v", err)
	} else if val != 25 {
		t.Errorf("Unexpected collection count %d, expected 25", val)
	}
	if err := testutil.GatherAndCompare(gatherers, strings.NewReader(expected),
		"infiniband_switch_port_excessive_buffer_overrun_errors_total", "infiniband_switch_port_link_downed_total",
		"infiniband_switch_port_link_error_recovery_total", "infiniband_switch_port_local_link_integrity_errors_total",
		"infiniband_exporter_collect_errors", "infiniband_exporter_collect_timeouts"); err != nil {
		t.Errorf("unexpected collecting result:\n%s", err)
	}
}
