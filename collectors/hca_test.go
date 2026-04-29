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

var (
	hcaDevices = []InfinibandDevice{
		{Type: "CA", LID: "133", GUID: "0x7cfe9003003b4b96", Rate: (25 * 4 * 125000000), RawRate: 1.2890625e+10, Name: "o0002 HCA-1",
			Uplinks: map[string]InfinibandUplink{
				"1": {Type: "SW", LID: "1719", PortNumber: "11", GUID: "0x7cfe9003009ce5b0", Name: "ib-i1l1s01"},
			},
		},
		{Type: "CA", LID: "134", GUID: "0x7cfe9003003b4bde", Rate: (25 * 4 * 125000000), RawRate: 1.2890625e+10, Name: "o0001 HCA-1",
			Uplinks: map[string]InfinibandUplink{
				"1": {Type: "SW", LID: "1719", PortNumber: "10", GUID: "0x7cfe9003009ce5b0", Name: "ib-i1l1s01"},
			},
		},
	}
)

func TestHCACollector(t *testing.T) {
	if _, err := kingpin.CommandLine.Parse([]string{}); err != nil {
		t.Fatal(err)
	}
	SetPerfqueryExecs(t, false, false)
	expected := `
		# HELP infiniband_exporter_collect_errors Number of errors that occurred during collection
		# TYPE infiniband_exporter_collect_errors gauge
		infiniband_exporter_collect_errors{collector="hca"} 0
		# HELP infiniband_exporter_collect_timeouts Number of timeouts that occurred during collection
		# TYPE infiniband_exporter_collect_timeouts gauge
		infiniband_exporter_collect_timeouts{collector="hca"} 0
		# HELP infiniband_hca_info Constant 1 carrying HCA identification labels (lid, guid, hca name).
		# TYPE infiniband_hca_info gauge
		infiniband_hca_info{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",lid="133"} 1
		infiniband_hca_info{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",lid="134"} 1
		# HELP infiniband_hca_port_excessive_buffer_overrun_errors_total Excessive buffer overrun errors — receive buffer overran the configured threshold.
		# TYPE infiniband_hca_port_excessive_buffer_overrun_errors_total counter
		infiniband_hca_port_excessive_buffer_overrun_errors_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch=""} 0
		infiniband_hca_port_excessive_buffer_overrun_errors_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch=""} 0
		# HELP infiniband_hca_port_link_downed_total Times the link error recovery process failed and the link went down.
		# TYPE infiniband_hca_port_link_downed_total counter
		infiniband_hca_port_link_downed_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch=""} 0
		infiniband_hca_port_link_downed_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch=""} 0
		# HELP infiniband_hca_port_link_error_recovery_total Times the link successfully completed the link error recovery process.
		# TYPE infiniband_hca_port_link_error_recovery_total counter
		infiniband_hca_port_link_error_recovery_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch=""} 0
		infiniband_hca_port_link_error_recovery_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch=""} 0
		# HELP infiniband_hca_port_local_link_integrity_errors_total Local link integrity threshold errors (LocalLinkIntegrityErrors).
		# TYPE infiniband_hca_port_local_link_integrity_errors_total counter
		infiniband_hca_port_local_link_integrity_errors_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch=""} 0
		infiniband_hca_port_local_link_integrity_errors_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch=""} 0
		# HELP infiniband_hca_port_multicast_receive_packets_total Total multicast packets received on this port.
		# TYPE infiniband_hca_port_multicast_receive_packets_total counter
		infiniband_hca_port_multicast_receive_packets_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch=""} 3732373137
		infiniband_hca_port_multicast_receive_packets_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch=""} 3732158589
		# HELP infiniband_hca_port_multicast_transmit_packets_total Total multicast packets transmitted on this port.
		# TYPE infiniband_hca_port_multicast_transmit_packets_total counter
		infiniband_hca_port_multicast_transmit_packets_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch=""} 544690
		infiniband_hca_port_multicast_transmit_packets_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch=""} 721488
		# HELP infiniband_hca_port_qp1_dropped_total Subnet management QP1 packets dropped (QP1Dropped).
		# TYPE infiniband_hca_port_qp1_dropped_total counter
		infiniband_hca_port_qp1_dropped_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch=""} 0
		infiniband_hca_port_qp1_dropped_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch=""} 0
		# HELP infiniband_hca_port_receive_constraint_errors_total Inbound packets discarded because of a partitioning or rate-limit constraint.
		# TYPE infiniband_hca_port_receive_constraint_errors_total counter
		infiniband_hca_port_receive_constraint_errors_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch=""} 0
		infiniband_hca_port_receive_constraint_errors_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch=""} 0
		# HELP infiniband_hca_port_receive_data_bytes_total Total data octets received on this port (perfquery PortRcvData scaled to bytes — IB octets are 4-byte words).
		# TYPE infiniband_hca_port_receive_data_bytes_total counter
		infiniband_hca_port_receive_data_bytes_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch=""} 148901607811540
		infiniband_hca_port_receive_data_bytes_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch=""} 39009938353200
		# HELP infiniband_hca_port_receive_errors_total Errors detected on receive packets for any reason (PortRcvErrors).
		# TYPE infiniband_hca_port_receive_errors_total counter
		infiniband_hca_port_receive_errors_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch=""} 0
		infiniband_hca_port_receive_errors_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch=""} 0
		# HELP infiniband_hca_port_receive_packets_total Total packets received on this port (any size, any traffic class).
		# TYPE infiniband_hca_port_receive_packets_total counter
		infiniband_hca_port_receive_packets_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch=""} 100583719365
		infiniband_hca_port_receive_packets_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch=""} 33038722564
		# HELP infiniband_hca_port_receive_remote_physical_errors_total Receive errors caused by a remote physical-layer error (e.g. EBP marker).
		# TYPE infiniband_hca_port_receive_remote_physical_errors_total counter
		infiniband_hca_port_receive_remote_physical_errors_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch=""} 0
		infiniband_hca_port_receive_remote_physical_errors_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch=""} 0
		# HELP infiniband_hca_port_receive_switch_relay_errors_total Packets dropped during switch routing because no relay path was available.
		# TYPE infiniband_hca_port_receive_switch_relay_errors_total counter
		infiniband_hca_port_receive_switch_relay_errors_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch=""} 0
		infiniband_hca_port_receive_switch_relay_errors_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch=""} 0
		# HELP infiniband_hca_port_symbol_error_total Minor link errors detected on one or more physical lanes (SymbolErrorCounter).
		# TYPE infiniband_hca_port_symbol_error_total counter
		infiniband_hca_port_symbol_error_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch=""} 0
		infiniband_hca_port_symbol_error_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch=""} 0
		# HELP infiniband_hca_port_transmit_constraint_errors_total Outbound packets discarded because of a partitioning or rate-limit constraint.
		# TYPE infiniband_hca_port_transmit_constraint_errors_total counter
		infiniband_hca_port_transmit_constraint_errors_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch=""} 0
		infiniband_hca_port_transmit_constraint_errors_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch=""} 0
		# HELP infiniband_hca_port_transmit_data_bytes_total Total data octets transmitted on this port (perfquery PortXmitData scaled to bytes — IB octets are 4-byte words).
		# TYPE infiniband_hca_port_transmit_data_bytes_total counter
		infiniband_hca_port_transmit_data_bytes_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch=""} 148434707415420
		infiniband_hca_port_transmit_data_bytes_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch=""} 36198369975904
		# HELP infiniband_hca_port_transmit_discards_total Outbound packets discarded because the port was busy or down (PortXmitDiscards).
		# TYPE infiniband_hca_port_transmit_discards_total counter
		infiniband_hca_port_transmit_discards_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch=""} 0
		infiniband_hca_port_transmit_discards_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch=""} 0
		# HELP infiniband_hca_port_transmit_packets_total Total packets transmitted on this port (any size, any traffic class).
		# TYPE infiniband_hca_port_transmit_packets_total counter
		infiniband_hca_port_transmit_packets_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch=""} 96917117320
		infiniband_hca_port_transmit_packets_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch=""} 28825338611
		# HELP infiniband_hca_port_transmit_wait_total Time ticks during which the port had data to transmit but no flow-control credits available — primary congestion signal.
		# TYPE infiniband_hca_port_transmit_wait_total counter
		infiniband_hca_port_transmit_wait_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch=""} 0
		infiniband_hca_port_transmit_wait_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch=""} 0
		# HELP infiniband_hca_port_unicast_receive_packets_total Total unicast packets received on this port.
		# TYPE infiniband_hca_port_unicast_receive_packets_total counter
		infiniband_hca_port_unicast_receive_packets_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch=""} 96851346228
		infiniband_hca_port_unicast_receive_packets_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch=""} 29306563974
		# HELP infiniband_hca_port_unicast_transmit_packets_total Total unicast packets transmitted on this port.
		# TYPE infiniband_hca_port_unicast_transmit_packets_total counter
		infiniband_hca_port_unicast_transmit_packets_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch=""} 96916572630
		infiniband_hca_port_unicast_transmit_packets_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch=""} 28824617123
		# HELP infiniband_hca_port_vl15_dropped_total Subnet management packets (VL15) dropped because of resource limitations.
		# TYPE infiniband_hca_port_vl15_dropped_total counter
		infiniband_hca_port_vl15_dropped_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch=""} 0
		infiniband_hca_port_vl15_dropped_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch=""} 0
		# HELP infiniband_hca_port_rate_bytes_per_second Effective HCA port rate in bytes per second (after IB encoding overhead removed).
		# TYPE infiniband_hca_port_rate_bytes_per_second gauge
		infiniband_hca_port_rate_bytes_per_second{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1"} 1.25e+10
		infiniband_hca_port_rate_bytes_per_second{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1"} 1.25e+10
		# HELP infiniband_hca_port_raw_rate_bytes_per_second Raw HCA port rate in bytes per second (signaling rate, before encoding overhead).
		# TYPE infiniband_hca_port_raw_rate_bytes_per_second gauge
		infiniband_hca_port_raw_rate_bytes_per_second{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1"} 1.2890625e+10
		infiniband_hca_port_raw_rate_bytes_per_second{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1"} 1.2890625e+10
		# HELP infiniband_hca_uplink_info Constant 1 describing the switch port this HCA port is connected to.
		# TYPE infiniband_hca_uplink_info gauge
		infiniband_hca_uplink_info{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch="",uplink="ib-i1l1s01",uplink_guid="0x7cfe9003009ce5b0",uplink_lid="1719",uplink_port="11",uplink_type="SW"} 1
		infiniband_hca_uplink_info{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch="",uplink="ib-i1l1s01",uplink_guid="0x7cfe9003009ce5b0",uplink_lid="1719",uplink_port="10",uplink_type="SW"} 1
	`
	collector := NewHCACollector(&hcaDevices, false, slog.New(slog.DiscardHandler))
	gatherers := setupGatherer(collector)
	if val, err := testutil.GatherAndCount(gatherers); err != nil {
		t.Errorf("Unexpected error: %v", err)
	} else if val != 63 {
		t.Errorf("Unexpected collection count %d, expected 63", val)
	}
	if err := testutil.GatherAndCompare(gatherers, strings.NewReader(expected),
		"infiniband_hca_port_excessive_buffer_overrun_errors_total", "infiniband_hca_port_link_downed_total",
		"infiniband_hca_port_link_error_recovery_total", "infiniband_hca_port_local_link_integrity_errors_total",
		"infiniband_hca_port_multicast_receive_packets_total", "infiniband_hca_port_multicast_transmit_packets_total",
		"infiniband_hca_port_qp1_dropped_total", "infiniband_hca_port_receive_constraint_errors_total",
		"infiniband_hca_port_receive_data_bytes_total", "infiniband_hca_port_receive_errors_total",
		"infiniband_hca_port_receive_packets_total", "infiniband_hca_port_receive_remote_physical_errors_total",
		"infiniband_hca_port_receive_switch_relay_errors_total", "infiniband_hca_port_symbol_error_total",
		"infiniband_hca_port_transmit_constraint_errors_total", "infiniband_hca_port_transmit_data_bytes_total",
		"infiniband_hca_port_transmit_discards_total", "infiniband_hca_port_transmit_packets_total",
		"infiniband_hca_port_transmit_wait_total", "infiniband_hca_port_unicast_receive_packets_total",
		"infiniband_hca_port_unicast_transmit_packets_total", "infiniband_hca_port_vl15_dropped_total",
		"infiniband_hca_port_buffer_overrun_errors_total",
		"infiniband_hca_info", "infiniband_hca_port_rate_bytes_per_second", "infiniband_hca_port_raw_rate_bytes_per_second", "infiniband_hca_uplink_info",
		"infiniband_exporter_collect_errors", "infiniband_exporter_collect_timeouts"); err != nil {
		t.Errorf("unexpected collecting result:\n%s", err)
	}
}

func TestHCACollectorFull(t *testing.T) {
	if _, err := kingpin.CommandLine.Parse([]string{"--collector.hca.rcv-err-details"}); err != nil {
		t.Fatal(err)
	}
	SetPerfqueryExecs(t, false, false)
	expected := `
		# HELP infiniband_exporter_collect_errors Number of errors that occurred during collection
		# TYPE infiniband_exporter_collect_errors gauge
		infiniband_exporter_collect_errors{collector="hca"} 0
		# HELP infiniband_exporter_collect_timeouts Number of timeouts that occurred during collection
		# TYPE infiniband_exporter_collect_timeouts gauge
		infiniband_exporter_collect_timeouts{collector="hca"} 0
		# HELP infiniband_hca_info Constant 1 carrying HCA identification labels (lid, guid, hca name).
		# TYPE infiniband_hca_info gauge
		infiniband_hca_info{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",lid="133"} 1
		infiniband_hca_info{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",lid="134"} 1
		# HELP infiniband_hca_port_buffer_overrun_errors_total Inbound packets dropped because the receive buffer overran (PortBufferOverrunErrors).
		# TYPE infiniband_hca_port_buffer_overrun_errors_total counter
		infiniband_hca_port_buffer_overrun_errors_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch=""} 0
		infiniband_hca_port_buffer_overrun_errors_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch=""} 0
		# HELP infiniband_hca_port_dlid_mapping_errors_total Inbound packets dropped because the destination LID had no valid mapping (PortDLIDMappingErrors).
		# TYPE infiniband_hca_port_dlid_mapping_errors_total counter
		infiniband_hca_port_dlid_mapping_errors_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch=""} 0
		infiniband_hca_port_dlid_mapping_errors_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch=""} 0
		# HELP infiniband_hca_port_excessive_buffer_overrun_errors_total Excessive buffer overrun errors — receive buffer overran the configured threshold.
		# TYPE infiniband_hca_port_excessive_buffer_overrun_errors_total counter
		infiniband_hca_port_excessive_buffer_overrun_errors_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch=""} 0
		infiniband_hca_port_excessive_buffer_overrun_errors_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch=""} 0
		# HELP infiniband_hca_port_link_downed_total Times the link error recovery process failed and the link went down.
		# TYPE infiniband_hca_port_link_downed_total counter
		infiniband_hca_port_link_downed_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch=""} 0
		infiniband_hca_port_link_downed_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch=""} 0
		# HELP infiniband_hca_port_link_error_recovery_total Times the link successfully completed the link error recovery process.
		# TYPE infiniband_hca_port_link_error_recovery_total counter
		infiniband_hca_port_link_error_recovery_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch=""} 0
		infiniband_hca_port_link_error_recovery_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch=""} 0
		# HELP infiniband_hca_port_local_link_integrity_errors_total Local link integrity threshold errors (LocalLinkIntegrityErrors).
		# TYPE infiniband_hca_port_local_link_integrity_errors_total counter
		infiniband_hca_port_local_link_integrity_errors_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch=""} 0
		infiniband_hca_port_local_link_integrity_errors_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch=""} 0
		# HELP infiniband_hca_port_local_physical_errors_total Local physical-layer errors detected on inbound traffic (PortLocalPhysicalErrors).
		# TYPE infiniband_hca_port_local_physical_errors_total counter
		infiniband_hca_port_local_physical_errors_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch=""} 0
		infiniband_hca_port_local_physical_errors_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch=""} 0
		# HELP infiniband_hca_port_looping_errors_total Inbound packets dropped because they were detected as looping (PortLoopingErrors).
		# TYPE infiniband_hca_port_looping_errors_total counter
		infiniband_hca_port_looping_errors_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch=""} 0
		infiniband_hca_port_looping_errors_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch=""} 0
		# HELP infiniband_hca_port_malformed_packet_errors_total Inbound packets discarded because they were malformed.
		# TYPE infiniband_hca_port_malformed_packet_errors_total counter
		infiniband_hca_port_malformed_packet_errors_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch=""} 0
		infiniband_hca_port_malformed_packet_errors_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch=""} 0
		# HELP infiniband_hca_port_multicast_receive_packets_total Total multicast packets received on this port.
		# TYPE infiniband_hca_port_multicast_receive_packets_total counter
		infiniband_hca_port_multicast_receive_packets_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch=""} 3732373137
		infiniband_hca_port_multicast_receive_packets_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch=""} 3732158589
		# HELP infiniband_hca_port_multicast_transmit_packets_total Total multicast packets transmitted on this port.
		# TYPE infiniband_hca_port_multicast_transmit_packets_total counter
		infiniband_hca_port_multicast_transmit_packets_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch=""} 544690
		infiniband_hca_port_multicast_transmit_packets_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch=""} 721488
		# HELP infiniband_hca_port_qp1_dropped_total Subnet management QP1 packets dropped (QP1Dropped).
		# TYPE infiniband_hca_port_qp1_dropped_total counter
		infiniband_hca_port_qp1_dropped_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch=""} 0
		infiniband_hca_port_qp1_dropped_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch=""} 0
		# HELP infiniband_hca_port_receive_constraint_errors_total Inbound packets discarded because of a partitioning or rate-limit constraint.
		# TYPE infiniband_hca_port_receive_constraint_errors_total counter
		infiniband_hca_port_receive_constraint_errors_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch=""} 0
		infiniband_hca_port_receive_constraint_errors_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch=""} 0
		# HELP infiniband_hca_port_receive_data_bytes_total Total data octets received on this port (perfquery PortRcvData scaled to bytes — IB octets are 4-byte words).
		# TYPE infiniband_hca_port_receive_data_bytes_total counter
		infiniband_hca_port_receive_data_bytes_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch=""} 148901607811540
		infiniband_hca_port_receive_data_bytes_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch=""} 39009938353200
		# HELP infiniband_hca_port_receive_errors_total Errors detected on receive packets for any reason (PortRcvErrors).
		# TYPE infiniband_hca_port_receive_errors_total counter
		infiniband_hca_port_receive_errors_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch=""} 0
		infiniband_hca_port_receive_errors_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch=""} 0
		# HELP infiniband_hca_port_receive_packets_total Total packets received on this port (any size, any traffic class).
		# TYPE infiniband_hca_port_receive_packets_total counter
		infiniband_hca_port_receive_packets_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch=""} 100583719365
		infiniband_hca_port_receive_packets_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch=""} 33038722564
		# HELP infiniband_hca_port_receive_remote_physical_errors_total Receive errors caused by a remote physical-layer error (e.g. EBP marker).
		# TYPE infiniband_hca_port_receive_remote_physical_errors_total counter
		infiniband_hca_port_receive_remote_physical_errors_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch=""} 0
		infiniband_hca_port_receive_remote_physical_errors_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch=""} 0
		# HELP infiniband_hca_port_receive_switch_relay_errors_total Packets dropped during switch routing because no relay path was available.
		# TYPE infiniband_hca_port_receive_switch_relay_errors_total counter
		infiniband_hca_port_receive_switch_relay_errors_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch=""} 0
		infiniband_hca_port_receive_switch_relay_errors_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch=""} 0
		# HELP infiniband_hca_port_symbol_error_total Minor link errors detected on one or more physical lanes (SymbolErrorCounter).
		# TYPE infiniband_hca_port_symbol_error_total counter
		infiniband_hca_port_symbol_error_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch=""} 0
		infiniband_hca_port_symbol_error_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch=""} 0
		# HELP infiniband_hca_port_transmit_constraint_errors_total Outbound packets discarded because of a partitioning or rate-limit constraint.
		# TYPE infiniband_hca_port_transmit_constraint_errors_total counter
		infiniband_hca_port_transmit_constraint_errors_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch=""} 0
		infiniband_hca_port_transmit_constraint_errors_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch=""} 0
		# HELP infiniband_hca_port_transmit_data_bytes_total Total data octets transmitted on this port (perfquery PortXmitData scaled to bytes — IB octets are 4-byte words).
		# TYPE infiniband_hca_port_transmit_data_bytes_total counter
		infiniband_hca_port_transmit_data_bytes_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch=""} 148434707415420
		infiniband_hca_port_transmit_data_bytes_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch=""} 36198369975904
		# HELP infiniband_hca_port_transmit_discards_total Outbound packets discarded because the port was busy or down (PortXmitDiscards).
		# TYPE infiniband_hca_port_transmit_discards_total counter
		infiniband_hca_port_transmit_discards_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch=""} 0
		infiniband_hca_port_transmit_discards_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch=""} 0
		# HELP infiniband_hca_port_transmit_packets_total Total packets transmitted on this port (any size, any traffic class).
		# TYPE infiniband_hca_port_transmit_packets_total counter
		infiniband_hca_port_transmit_packets_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch=""} 96917117320
		infiniband_hca_port_transmit_packets_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch=""} 28825338611
		# HELP infiniband_hca_port_transmit_wait_total Time ticks during which the port had data to transmit but no flow-control credits available — primary congestion signal.
		# TYPE infiniband_hca_port_transmit_wait_total counter
		infiniband_hca_port_transmit_wait_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch=""} 0
		infiniband_hca_port_transmit_wait_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch=""} 0
		# HELP infiniband_hca_port_unicast_receive_packets_total Total unicast packets received on this port.
		# TYPE infiniband_hca_port_unicast_receive_packets_total counter
		infiniband_hca_port_unicast_receive_packets_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch=""} 96851346228
		infiniband_hca_port_unicast_receive_packets_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch=""} 29306563974
		# HELP infiniband_hca_port_unicast_transmit_packets_total Total unicast packets transmitted on this port.
		# TYPE infiniband_hca_port_unicast_transmit_packets_total counter
		infiniband_hca_port_unicast_transmit_packets_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch=""} 96916572630
		infiniband_hca_port_unicast_transmit_packets_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch=""} 28824617123
		# HELP infiniband_hca_port_vl_mapping_errors_total Inbound packets dropped because the SL→VL mapping was invalid (PortVLMappingErrors).
		# TYPE infiniband_hca_port_vl_mapping_errors_total counter
		infiniband_hca_port_vl_mapping_errors_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch=""} 0
		infiniband_hca_port_vl_mapping_errors_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch=""} 0
		# HELP infiniband_hca_port_vl15_dropped_total Subnet management packets (VL15) dropped because of resource limitations.
		# TYPE infiniband_hca_port_vl15_dropped_total counter
		infiniband_hca_port_vl15_dropped_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch=""} 0
		infiniband_hca_port_vl15_dropped_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch=""} 0
		# HELP infiniband_hca_port_rate_bytes_per_second Effective HCA port rate in bytes per second (after IB encoding overhead removed).
		# TYPE infiniband_hca_port_rate_bytes_per_second gauge
		infiniband_hca_port_rate_bytes_per_second{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1"} 1.25e+10
		infiniband_hca_port_rate_bytes_per_second{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1"} 1.25e+10
		# HELP infiniband_hca_port_raw_rate_bytes_per_second Raw HCA port rate in bytes per second (signaling rate, before encoding overhead).
		# TYPE infiniband_hca_port_raw_rate_bytes_per_second gauge
		infiniband_hca_port_raw_rate_bytes_per_second{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1"} 1.2890625e+10
		infiniband_hca_port_raw_rate_bytes_per_second{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1"} 1.2890625e+10
		# HELP infiniband_hca_uplink_info Constant 1 describing the switch port this HCA port is connected to.
		# TYPE infiniband_hca_uplink_info gauge
		infiniband_hca_uplink_info{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch="",uplink="ib-i1l1s01",uplink_guid="0x7cfe9003009ce5b0",uplink_lid="1719",uplink_port="11",uplink_type="SW"} 1
		infiniband_hca_uplink_info{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch="",uplink="ib-i1l1s01",uplink_guid="0x7cfe9003009ce5b0",uplink_lid="1719",uplink_port="10",uplink_type="SW"} 1
	`
	collector := NewHCACollector(&hcaDevices, false, slog.New(slog.DiscardHandler))
	gatherers := setupGatherer(collector)
	if val, err := testutil.GatherAndCount(gatherers); err != nil {
		t.Errorf("Unexpected error: %v", err)
	} else if val != 81 {
		t.Errorf("Unexpected collection count %d, expected 81", val)
	}
	if err := testutil.GatherAndCompare(gatherers, strings.NewReader(expected),
		"infiniband_hca_port_excessive_buffer_overrun_errors_total", "infiniband_hca_port_link_downed_total",
		"infiniband_hca_port_link_error_recovery_total", "infiniband_hca_port_local_link_integrity_errors_total",
		"infiniband_hca_port_multicast_receive_packets_total", "infiniband_hca_port_multicast_transmit_packets_total",
		"infiniband_hca_port_qp1_dropped_total", "infiniband_hca_port_receive_constraint_errors_total",
		"infiniband_hca_port_receive_data_bytes_total", "infiniband_hca_port_receive_errors_total",
		"infiniband_hca_port_receive_packets_total", "infiniband_hca_port_receive_remote_physical_errors_total",
		"infiniband_hca_port_receive_switch_relay_errors_total", "infiniband_hca_port_symbol_error_total",
		"infiniband_hca_port_transmit_constraint_errors_total", "infiniband_hca_port_transmit_data_bytes_total",
		"infiniband_hca_port_transmit_discards_total", "infiniband_hca_port_transmit_packets_total",
		"infiniband_hca_port_transmit_wait_total", "infiniband_hca_port_unicast_receive_packets_total",
		"infiniband_hca_port_unicast_transmit_packets_total", "infiniband_hca_port_vl15_dropped_total",
		"infiniband_hca_port_buffer_overrun_errors_total", "infiniband_hca_port_dlid_mapping_errors_total",
		"infiniband_hca_port_local_physical_errors_total", "infiniband_hca_port_looping_errors_total",
		"infiniband_hca_port_malformed_packet_errors_total", "infiniband_hca_port_vl_mapping_errors_total",
		"infiniband_hca_info", "infiniband_hca_port_rate_bytes_per_second", "infiniband_hca_port_raw_rate_bytes_per_second", "infiniband_hca_uplink_info",
		"infiniband_exporter_collect_errors", "infiniband_exporter_collect_timeouts"); err != nil {
		t.Errorf("unexpected collecting result:\n%s", err)
	}
}

func TestHCACollectorError(t *testing.T) {
	if _, err := kingpin.CommandLine.Parse([]string{}); err != nil {
		t.Fatal(err)
	}
	SetPerfqueryExecs(t, true, false)
	expected := `
		# HELP infiniband_exporter_collect_errors Number of errors that occurred during collection
		# TYPE infiniband_exporter_collect_errors gauge
		infiniband_exporter_collect_errors{collector="hca"} 2
		# HELP infiniband_exporter_collect_timeouts Number of timeouts that occurred during collection
		# TYPE infiniband_exporter_collect_timeouts gauge
		infiniband_exporter_collect_timeouts{collector="hca"} 0
	`
	collector := NewHCACollector(&hcaDevices, false, slog.New(slog.DiscardHandler))
	gatherers := setupGatherer(collector)
	if val, err := testutil.GatherAndCount(gatherers); err != nil {
		t.Errorf("Unexpected error: %v", err)
	} else if val != 19 {
		t.Errorf("Unexpected collection count %d, expected 19", val)
	}
	if err := testutil.GatherAndCompare(gatherers, strings.NewReader(expected),
		"infiniband_hca_port_excessive_buffer_overrun_errors_total", "infiniband_hca_port_link_downed_total",
		"infiniband_hca_port_link_error_recovery_total", "infiniband_hca_port_local_link_integrity_errors_total",
		"infiniband_exporter_collect_errors", "infiniband_exporter_collect_timeouts"); err != nil {
		t.Errorf("unexpected collecting result:\n%s", err)
	}
}

func TestHCACollectorErrorRunonce(t *testing.T) {
	if _, err := kingpin.CommandLine.Parse([]string{}); err != nil {
		t.Fatal(err)
	}
	SetPerfqueryExecs(t, true, false)
	expected := `
		# HELP infiniband_exporter_collect_errors Number of errors that occurred during collection
		# TYPE infiniband_exporter_collect_errors gauge
		infiniband_exporter_collect_errors{collector="hca-runonce"} 2
		# HELP infiniband_exporter_collect_timeouts Number of timeouts that occurred during collection
		# TYPE infiniband_exporter_collect_timeouts gauge
		infiniband_exporter_collect_timeouts{collector="hca-runonce"} 0
	`
	collector := NewHCACollector(&hcaDevices, true, slog.New(slog.DiscardHandler))
	gatherers := setupGatherer(collector)
	if val, err := testutil.GatherAndCount(gatherers); err != nil {
		t.Errorf("Unexpected error: %v", err)
	} else if val != 20 {
		t.Errorf("Unexpected collection count %d, expected 20", val)
	}
	if err := testutil.GatherAndCompare(gatherers, strings.NewReader(expected),
		"infiniband_hca_port_excessive_buffer_overrun_errors_total", "infiniband_hca_port_link_downed_total",
		"infiniband_hca_port_link_error_recovery_total", "infiniband_hca_port_local_link_integrity_errors_total",
		"infiniband_exporter_collect_errors", "infiniband_exporter_collect_timeouts"); err != nil {
		t.Errorf("unexpected collecting result:\n%s", err)
	}
}

func TestHCACollectorTimeout(t *testing.T) {
	if _, err := kingpin.CommandLine.Parse([]string{}); err != nil {
		t.Fatal(err)
	}
	SetPerfqueryExecs(t, false, true)
	expected := `
		# HELP infiniband_exporter_collect_errors Number of errors that occurred during collection
		# TYPE infiniband_exporter_collect_errors gauge
		infiniband_exporter_collect_errors{collector="hca"} 0
		# HELP infiniband_exporter_collect_timeouts Number of timeouts that occurred during collection
		# TYPE infiniband_exporter_collect_timeouts gauge
		infiniband_exporter_collect_timeouts{collector="hca"} 2
	`
	collector := NewHCACollector(&hcaDevices, false, slog.New(slog.DiscardHandler))
	gatherers := setupGatherer(collector)
	if val, err := testutil.GatherAndCount(gatherers); err != nil {
		t.Errorf("Unexpected error: %v", err)
	} else if val != 19 {
		t.Errorf("Unexpected collection count %d, expected 19", val)
	}
	if err := testutil.GatherAndCompare(gatherers, strings.NewReader(expected),
		"infiniband_hca_port_excessive_buffer_overrun_errors_total", "infiniband_hca_port_link_downed_total",
		"infiniband_hca_port_link_error_recovery_total", "infiniband_hca_port_local_link_integrity_errors_total",
		"infiniband_exporter_collect_errors", "infiniband_exporter_collect_timeouts"); err != nil {
		t.Errorf("unexpected collecting result:\n%s", err)
	}
}
