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
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"log/slog"

	kingpin "github.com/alecthomas/kingpin/v2"

	"github.com/SckyzO/infiniband_exporter/collectors"
)

const (
	address = "localhost:19315"
)

var (
	outputPath     string
	expectedSwitch = `# HELP infiniband_switch_info Constant 1 carrying switch identification labels (lid, guid, switch name)
# TYPE infiniband_switch_info gauge
infiniband_switch_info{guid="0x506b4b03005c2740",lid="2052",switch="ib-i4l1s01"} 1
infiniband_switch_info{guid="0x7cfe9003009ce5b0",lid="1719",switch="ib-i1l1s01"} 1
# HELP infiniband_switch_port_excessive_buffer_overrun_errors_total Excessive buffer overrun errors — receive buffer overran the configured threshold.
# TYPE infiniband_switch_port_excessive_buffer_overrun_errors_total counter
infiniband_switch_port_excessive_buffer_overrun_errors_total{guid="0x506b4b03005c2740",port="1",switch="ib-i4l1s01"} 0
infiniband_switch_port_excessive_buffer_overrun_errors_total{guid="0x7cfe9003009ce5b0",port="1",switch="ib-i1l1s01"} 0
infiniband_switch_port_excessive_buffer_overrun_errors_total{guid="0x7cfe9003009ce5b0",port="2",switch="ib-i1l1s01"} 0
# HELP infiniband_switch_port_link_downed_total Times the link error recovery process failed and the link went down.
# TYPE infiniband_switch_port_link_downed_total counter
infiniband_switch_port_link_downed_total{guid="0x506b4b03005c2740",port="1",switch="ib-i4l1s01"} 1
infiniband_switch_port_link_downed_total{guid="0x7cfe9003009ce5b0",port="1",switch="ib-i1l1s01"} 0
infiniband_switch_port_link_downed_total{guid="0x7cfe9003009ce5b0",port="2",switch="ib-i1l1s01"} 0
# HELP infiniband_switch_port_link_error_recovery_total Times the link successfully completed the link error recovery process.
# TYPE infiniband_switch_port_link_error_recovery_total counter
infiniband_switch_port_link_error_recovery_total{guid="0x506b4b03005c2740",port="1",switch="ib-i4l1s01"} 0
infiniband_switch_port_link_error_recovery_total{guid="0x7cfe9003009ce5b0",port="1",switch="ib-i1l1s01"} 0
infiniband_switch_port_link_error_recovery_total{guid="0x7cfe9003009ce5b0",port="2",switch="ib-i1l1s01"} 0
# HELP infiniband_switch_port_local_link_integrity_errors_total Local link integrity threshold errors (LocalLinkIntegrityErrors).
# TYPE infiniband_switch_port_local_link_integrity_errors_total counter
infiniband_switch_port_local_link_integrity_errors_total{guid="0x506b4b03005c2740",port="1",switch="ib-i4l1s01"} 0
infiniband_switch_port_local_link_integrity_errors_total{guid="0x7cfe9003009ce5b0",port="1",switch="ib-i1l1s01"} 0
infiniband_switch_port_local_link_integrity_errors_total{guid="0x7cfe9003009ce5b0",port="2",switch="ib-i1l1s01"} 0
# HELP infiniband_switch_port_multicast_receive_packets_total Total multicast packets received on this port.
# TYPE infiniband_switch_port_multicast_receive_packets_total counter
infiniband_switch_port_multicast_receive_packets_total{guid="0x506b4b03005c2740",port="1",switch="ib-i4l1s01"} 6.69494e+06
infiniband_switch_port_multicast_receive_packets_total{guid="0x7cfe9003009ce5b0",port="1",switch="ib-i1l1s01"} 5.584846741e+09
infiniband_switch_port_multicast_receive_packets_total{guid="0x7cfe9003009ce5b0",port="2",switch="ib-i1l1s01"} 0
# HELP infiniband_switch_port_multicast_transmit_packets_total Total multicast packets transmitted on this port.
# TYPE infiniband_switch_port_multicast_transmit_packets_total counter
infiniband_switch_port_multicast_transmit_packets_total{guid="0x506b4b03005c2740",port="1",switch="ib-i4l1s01"} 5.623645694e+09
infiniband_switch_port_multicast_transmit_packets_total{guid="0x7cfe9003009ce5b0",port="1",switch="ib-i1l1s01"} 2.5038914e+07
infiniband_switch_port_multicast_transmit_packets_total{guid="0x7cfe9003009ce5b0",port="2",switch="ib-i1l1s01"} 0
# HELP infiniband_switch_port_qp1_dropped_total Subnet management QP1 packets dropped (QP1Dropped).
# TYPE infiniband_switch_port_qp1_dropped_total counter
infiniband_switch_port_qp1_dropped_total{guid="0x506b4b03005c2740",port="1",switch="ib-i4l1s01"} 0
infiniband_switch_port_qp1_dropped_total{guid="0x7cfe9003009ce5b0",port="1",switch="ib-i1l1s01"} 0
infiniband_switch_port_qp1_dropped_total{guid="0x7cfe9003009ce5b0",port="2",switch="ib-i1l1s01"} 0
# HELP infiniband_switch_port_rate_bytes_per_second Effective port rate in bytes per second (after IB encoding overhead removed).
# TYPE infiniband_switch_port_rate_bytes_per_second gauge
infiniband_switch_port_rate_bytes_per_second{guid="0x506b4b03005c2740",port="35",switch="ib-i4l1s01"} 1.25e+10
infiniband_switch_port_rate_bytes_per_second{guid="0x7cfe9003009ce5b0",port="1",switch="ib-i1l1s01"} 1.25e+10
infiniband_switch_port_rate_bytes_per_second{guid="0x7cfe9003009ce5b0",port="10",switch="ib-i1l1s01"} 1.25e+10
infiniband_switch_port_rate_bytes_per_second{guid="0x7cfe9003009ce5b0",port="11",switch="ib-i1l1s01"} 1.25e+10
# HELP infiniband_switch_port_raw_rate_bytes_per_second Raw port rate in bytes per second (signaling rate, before encoding overhead).
# TYPE infiniband_switch_port_raw_rate_bytes_per_second gauge
infiniband_switch_port_raw_rate_bytes_per_second{guid="0x506b4b03005c2740",port="35",switch="ib-i4l1s01"} 1.2890625e+10
infiniband_switch_port_raw_rate_bytes_per_second{guid="0x7cfe9003009ce5b0",port="1",switch="ib-i1l1s01"} 1.2890625e+10
infiniband_switch_port_raw_rate_bytes_per_second{guid="0x7cfe9003009ce5b0",port="10",switch="ib-i1l1s01"} 1.2890625e+10
infiniband_switch_port_raw_rate_bytes_per_second{guid="0x7cfe9003009ce5b0",port="11",switch="ib-i1l1s01"} 1.2890625e+10
# HELP infiniband_switch_port_receive_constraint_errors_total Inbound packets discarded because of a partitioning or rate-limit constraint.
# TYPE infiniband_switch_port_receive_constraint_errors_total counter
infiniband_switch_port_receive_constraint_errors_total{guid="0x506b4b03005c2740",port="1",switch="ib-i4l1s01"} 0
infiniband_switch_port_receive_constraint_errors_total{guid="0x7cfe9003009ce5b0",port="1",switch="ib-i1l1s01"} 0
infiniband_switch_port_receive_constraint_errors_total{guid="0x7cfe9003009ce5b0",port="2",switch="ib-i1l1s01"} 0
# HELP infiniband_switch_port_receive_data_bytes_total Total data octets received on this port (perfquery PortRcvData scaled to bytes — IB octets are 4-byte words).
# TYPE infiniband_switch_port_receive_data_bytes_total counter
infiniband_switch_port_receive_data_bytes_total{guid="0x506b4b03005c2740",port="1",switch="ib-i4l1s01"} 7.15049367846516e+14
infiniband_switch_port_receive_data_bytes_total{guid="0x7cfe9003009ce5b0",port="1",switch="ib-i1l1s01"} 4.9116115103004e+13
infiniband_switch_port_receive_data_bytes_total{guid="0x7cfe9003009ce5b0",port="2",switch="ib-i1l1s01"} 1.56315219973512e+14
# HELP infiniband_switch_port_receive_errors_total Errors detected on receive packets for any reason (PortRcvErrors).
# TYPE infiniband_switch_port_receive_errors_total counter
infiniband_switch_port_receive_errors_total{guid="0x506b4b03005c2740",port="1",switch="ib-i4l1s01"} 0
infiniband_switch_port_receive_errors_total{guid="0x7cfe9003009ce5b0",port="1",switch="ib-i1l1s01"} 0
infiniband_switch_port_receive_errors_total{guid="0x7cfe9003009ce5b0",port="2",switch="ib-i1l1s01"} 0
# HELP infiniband_switch_port_receive_packets_total Total packets received on this port (any size, any traffic class).
# TYPE infiniband_switch_port_receive_packets_total counter
infiniband_switch_port_receive_packets_total{guid="0x506b4b03005c2740",port="1",switch="ib-i4l1s01"} 3.87654829341e+11
infiniband_switch_port_receive_packets_total{guid="0x7cfe9003009ce5b0",port="1",switch="ib-i1l1s01"} 3.2262508468e+10
infiniband_switch_port_receive_packets_total{guid="0x7cfe9003009ce5b0",port="2",switch="ib-i1l1s01"} 9.3660802641e+10
# HELP infiniband_switch_port_receive_remote_physical_errors_total Receive errors caused by a remote physical-layer error (e.g. EBP marker).
# TYPE infiniband_switch_port_receive_remote_physical_errors_total counter
infiniband_switch_port_receive_remote_physical_errors_total{guid="0x506b4b03005c2740",port="1",switch="ib-i4l1s01"} 0
infiniband_switch_port_receive_remote_physical_errors_total{guid="0x7cfe9003009ce5b0",port="1",switch="ib-i1l1s01"} 0
infiniband_switch_port_receive_remote_physical_errors_total{guid="0x7cfe9003009ce5b0",port="2",switch="ib-i1l1s01"} 0
# HELP infiniband_switch_port_receive_switch_relay_errors_total Packets dropped during switch routing because no relay path was available.
# TYPE infiniband_switch_port_receive_switch_relay_errors_total counter
infiniband_switch_port_receive_switch_relay_errors_total{guid="0x506b4b03005c2740",port="1",switch="ib-i4l1s01"} 7
infiniband_switch_port_receive_switch_relay_errors_total{guid="0x7cfe9003009ce5b0",port="1",switch="ib-i1l1s01"} 0
infiniband_switch_port_receive_switch_relay_errors_total{guid="0x7cfe9003009ce5b0",port="2",switch="ib-i1l1s01"} 0
# HELP infiniband_switch_port_symbol_error_total Minor link errors detected on one or more physical lanes (SymbolErrorCounter).
# TYPE infiniband_switch_port_symbol_error_total counter
infiniband_switch_port_symbol_error_total{guid="0x506b4b03005c2740",port="1",switch="ib-i4l1s01"} 0
infiniband_switch_port_symbol_error_total{guid="0x7cfe9003009ce5b0",port="1",switch="ib-i1l1s01"} 0
infiniband_switch_port_symbol_error_total{guid="0x7cfe9003009ce5b0",port="2",switch="ib-i1l1s01"} 0
# HELP infiniband_switch_port_transmit_constraint_errors_total Outbound packets discarded because of a partitioning or rate-limit constraint.
# TYPE infiniband_switch_port_transmit_constraint_errors_total counter
infiniband_switch_port_transmit_constraint_errors_total{guid="0x506b4b03005c2740",port="1",switch="ib-i4l1s01"} 0
infiniband_switch_port_transmit_constraint_errors_total{guid="0x7cfe9003009ce5b0",port="1",switch="ib-i1l1s01"} 0
infiniband_switch_port_transmit_constraint_errors_total{guid="0x7cfe9003009ce5b0",port="2",switch="ib-i1l1s01"} 0
# HELP infiniband_switch_port_transmit_data_bytes_total Total data octets transmitted on this port (perfquery PortXmitData scaled to bytes — IB octets are 4-byte words).
# TYPE infiniband_switch_port_transmit_data_bytes_total counter
infiniband_switch_port_transmit_data_bytes_total{guid="0x506b4b03005c2740",port="1",switch="ib-i4l1s01"} 7.1516662870894e+14
infiniband_switch_port_transmit_data_bytes_total{guid="0x7cfe9003009ce5b0",port="1",switch="ib-i1l1s01"} 1.45192107443712e+14
infiniband_switch_port_transmit_data_bytes_total{guid="0x7cfe9003009ce5b0",port="2",switch="ib-i1l1s01"} 1.04026280056104e+14
# HELP infiniband_switch_port_transmit_discards_total Outbound packets discarded because the port was busy or down (PortXmitDiscards).
# TYPE infiniband_switch_port_transmit_discards_total counter
infiniband_switch_port_transmit_discards_total{guid="0x506b4b03005c2740",port="1",switch="ib-i4l1s01"} 20046
infiniband_switch_port_transmit_discards_total{guid="0x7cfe9003009ce5b0",port="1",switch="ib-i1l1s01"} 0
infiniband_switch_port_transmit_discards_total{guid="0x7cfe9003009ce5b0",port="2",switch="ib-i1l1s01"} 0
# HELP infiniband_switch_port_transmit_packets_total Total packets transmitted on this port (any size, any traffic class).
# TYPE infiniband_switch_port_transmit_packets_total counter
infiniband_switch_port_transmit_packets_total{guid="0x506b4b03005c2740",port="1",switch="ib-i4l1s01"} 3.93094651266e+11
infiniband_switch_port_transmit_packets_total{guid="0x7cfe9003009ce5b0",port="1",switch="ib-i1l1s01"} 1.01733204203e+11
infiniband_switch_port_transmit_packets_total{guid="0x7cfe9003009ce5b0",port="2",switch="ib-i1l1s01"} 1.22978948297e+11
# HELP infiniband_switch_port_transmit_wait_total Time ticks during which the port had data to transmit but no flow-control credits available — primary congestion signal.
# TYPE infiniband_switch_port_transmit_wait_total counter
infiniband_switch_port_transmit_wait_total{guid="0x506b4b03005c2740",port="1",switch="ib-i4l1s01"} 4.1864608e+07
infiniband_switch_port_transmit_wait_total{guid="0x7cfe9003009ce5b0",port="1",switch="ib-i1l1s01"} 2.2730501e+07
infiniband_switch_port_transmit_wait_total{guid="0x7cfe9003009ce5b0",port="2",switch="ib-i1l1s01"} 3.6510964e+07
# HELP infiniband_switch_port_unicast_receive_packets_total Total unicast packets received on this port.
# TYPE infiniband_switch_port_unicast_receive_packets_total counter
infiniband_switch_port_unicast_receive_packets_total{guid="0x506b4b03005c2740",port="1",switch="ib-i4l1s01"} 3.876481344e+11
infiniband_switch_port_unicast_receive_packets_total{guid="0x7cfe9003009ce5b0",port="1",switch="ib-i1l1s01"} 2.6677661727e+10
infiniband_switch_port_unicast_receive_packets_total{guid="0x7cfe9003009ce5b0",port="2",switch="ib-i1l1s01"} 9.3660802641e+10
# HELP infiniband_switch_port_unicast_transmit_packets_total Total unicast packets transmitted on this port.
# TYPE infiniband_switch_port_unicast_transmit_packets_total counter
infiniband_switch_port_unicast_transmit_packets_total{guid="0x506b4b03005c2740",port="1",switch="ib-i4l1s01"} 3.87471005571e+11
infiniband_switch_port_unicast_transmit_packets_total{guid="0x7cfe9003009ce5b0",port="1",switch="ib-i1l1s01"} 1.01708165289e+11
infiniband_switch_port_unicast_transmit_packets_total{guid="0x7cfe9003009ce5b0",port="2",switch="ib-i1l1s01"} 1.22978948297e+11
# HELP infiniband_switch_port_vl15_dropped_total Subnet management packets (VL15) dropped because of resource limitations.
# TYPE infiniband_switch_port_vl15_dropped_total counter
infiniband_switch_port_vl15_dropped_total{guid="0x506b4b03005c2740",port="1",switch="ib-i4l1s01"} 0
infiniband_switch_port_vl15_dropped_total{guid="0x7cfe9003009ce5b0",port="1",switch="ib-i1l1s01"} 0
infiniband_switch_port_vl15_dropped_total{guid="0x7cfe9003009ce5b0",port="2",switch="ib-i1l1s01"} 0
# HELP infiniband_switch_up 1 if the latest perfquery scrape of this switch succeeded, 0 otherwise (timeout or error).
# TYPE infiniband_switch_up gauge
infiniband_switch_up{guid="0x506b4b03005c2740",switch="ib-i4l1s01"} 1
infiniband_switch_up{guid="0x7cfe9003009ce5b0",switch="ib-i1l1s01"} 1
# HELP infiniband_switch_uplink_info Constant 1 describing the device connected to this switch port.
# TYPE infiniband_switch_uplink_info gauge
infiniband_switch_uplink_info{guid="0x506b4b03005c2740",port="35",switch="ib-i4l1s01",uplink="p0001 HCA-1",uplink_guid="0x506b4b0300cc02a6",uplink_lid="1432",uplink_port="1",uplink_type="CA"} 1
infiniband_switch_uplink_info{guid="0x7cfe9003009ce5b0",port="1",switch="ib-i1l1s01",uplink="ib-i1l2s01",uplink_guid="0x7cfe900300b07320",uplink_lid="1516",uplink_port="1",uplink_type="SW"} 1
infiniband_switch_uplink_info{guid="0x7cfe9003009ce5b0",port="10",switch="ib-i1l1s01",uplink="o0001 HCA-1",uplink_guid="0x7cfe9003003b4bde",uplink_lid="134",uplink_port="1",uplink_type="CA"} 1
infiniband_switch_uplink_info{guid="0x7cfe9003009ce5b0",port="11",switch="ib-i1l1s01",uplink="o0002 HCA-1",uplink_guid="0x7cfe9003003b4b96",uplink_lid="133",uplink_port="1",uplink_type="CA"} 1`
	expectedIbswinfo = `# HELP infiniband_switch_fan_rpm Switch fan rotation speed in RPM (one series per fan).
# TYPE infiniband_switch_fan_rpm gauge
infiniband_switch_fan_rpm{fan="1",guid="0x506b4b03005c2740",switch="ib-i4l1s01"} 6125
infiniband_switch_fan_rpm{fan="1",guid="0x7cfe9003009ce5b0",switch="ib-i1l1s01"} 8493
infiniband_switch_fan_rpm{fan="2",guid="0x506b4b03005c2740",switch="ib-i4l1s01"} 5251
infiniband_switch_fan_rpm{fan="2",guid="0x7cfe9003009ce5b0",switch="ib-i1l1s01"} 7349
infiniband_switch_fan_rpm{fan="3",guid="0x506b4b03005c2740",switch="ib-i4l1s01"} 6013
infiniband_switch_fan_rpm{fan="3",guid="0x7cfe9003009ce5b0",switch="ib-i1l1s01"} 8441
infiniband_switch_fan_rpm{fan="4",guid="0x506b4b03005c2740",switch="ib-i4l1s01"} 5335
infiniband_switch_fan_rpm{fan="4",guid="0x7cfe9003009ce5b0",switch="ib-i1l1s01"} 7270
infiniband_switch_fan_rpm{fan="5",guid="0x506b4b03005c2740",switch="ib-i4l1s01"} 6068
infiniband_switch_fan_rpm{fan="5",guid="0x7cfe9003009ce5b0",switch="ib-i1l1s01"} 8337
infiniband_switch_fan_rpm{fan="6",guid="0x506b4b03005c2740",switch="ib-i4l1s01"} 5423
infiniband_switch_fan_rpm{fan="6",guid="0x7cfe9003009ce5b0",switch="ib-i1l1s01"} 7156
infiniband_switch_fan_rpm{fan="7",guid="0x506b4b03005c2740",switch="ib-i4l1s01"} 5854
infiniband_switch_fan_rpm{fan="7",guid="0x7cfe9003009ce5b0",switch="ib-i1l1s01"} 8441
infiniband_switch_fan_rpm{fan="8",guid="0x506b4b03005c2740",switch="ib-i4l1s01"} 5467
infiniband_switch_fan_rpm{fan="8",guid="0x7cfe9003009ce5b0",switch="ib-i1l1s01"} 7232
infiniband_switch_fan_rpm{fan="9",guid="0x506b4b03005c2740",switch="ib-i4l1s01"} 5906
# HELP infiniband_switch_fan_status_info Constant 1 with the current overall fan status string label.
# TYPE infiniband_switch_fan_status_info gauge
infiniband_switch_fan_status_info{guid="0x506b4b03005c2740",status="OK",switch="ib-i4l1s01"} 1
infiniband_switch_fan_status_info{guid="0x7cfe9003009ce5b0",status="ERROR",switch="ib-i1l1s01"} 1
# HELP infiniband_switch_hardware_info Constant 1 carrying switch hardware identification labels (firmware version, PSID, part/serial numbers).
# TYPE infiniband_switch_hardware_info gauge
infiniband_switch_hardware_info{firmware_version="11.2008.2102",guid="0x7cfe9003009ce5b0",part_number="MSB7790-ES2F",psid="MT_1880110032",serial_number="MT1943X00498",switch="ib-i1l1s01"} 1
infiniband_switch_hardware_info{firmware_version="27.2010.3118",guid="0x506b4b03005c2740",part_number="MQM8790-HS2F",psid="MT_0000000063",serial_number="MT2152T10239",switch="ib-i4l1s01"} 1
# HELP infiniband_switch_power_supply_dc_power_status_info Constant 1 with the current DC power status string label (1 series per PSU per state).
# TYPE infiniband_switch_power_supply_dc_power_status_info gauge
infiniband_switch_power_supply_dc_power_status_info{guid="0x506b4b03005c2740",psu="0",status="OK",switch="ib-i4l1s01"} 1
infiniband_switch_power_supply_dc_power_status_info{guid="0x506b4b03005c2740",psu="1",status="OK",switch="ib-i4l1s01"} 1
infiniband_switch_power_supply_dc_power_status_info{guid="0x7cfe9003009ce5b0",psu="0",status="OK",switch="ib-i1l1s01"} 1
infiniband_switch_power_supply_dc_power_status_info{guid="0x7cfe9003009ce5b0",psu="1",status="OK",switch="ib-i1l1s01"} 1
# HELP infiniband_switch_power_supply_fan_status_info Constant 1 with the current PSU fan status string label (1 series per PSU per state).
# TYPE infiniband_switch_power_supply_fan_status_info gauge
infiniband_switch_power_supply_fan_status_info{guid="0x506b4b03005c2740",psu="0",status="OK",switch="ib-i4l1s01"} 1
infiniband_switch_power_supply_fan_status_info{guid="0x506b4b03005c2740",psu="1",status="OK",switch="ib-i4l1s01"} 1
infiniband_switch_power_supply_fan_status_info{guid="0x7cfe9003009ce5b0",psu="0",status="OK",switch="ib-i1l1s01"} 1
infiniband_switch_power_supply_fan_status_info{guid="0x7cfe9003009ce5b0",psu="1",status="OK",switch="ib-i1l1s01"} 1
# HELP infiniband_switch_power_supply_status_info Constant 1 with the current PSU status string label (1 series per PSU per state).
# TYPE infiniband_switch_power_supply_status_info gauge
infiniband_switch_power_supply_status_info{guid="0x506b4b03005c2740",psu="0",status="OK",switch="ib-i4l1s01"} 1
infiniband_switch_power_supply_status_info{guid="0x506b4b03005c2740",psu="1",status="OK",switch="ib-i4l1s01"} 1
infiniband_switch_power_supply_status_info{guid="0x7cfe9003009ce5b0",psu="0",status="OK",switch="ib-i1l1s01"} 1
infiniband_switch_power_supply_status_info{guid="0x7cfe9003009ce5b0",psu="1",status="OK",switch="ib-i1l1s01"} 1
# HELP infiniband_switch_power_supply_watts Power drawn by the PSU in watts.
# TYPE infiniband_switch_power_supply_watts gauge
infiniband_switch_power_supply_watts{guid="0x506b4b03005c2740",psu="0",switch="ib-i4l1s01"} 154
infiniband_switch_power_supply_watts{guid="0x506b4b03005c2740",psu="1",switch="ib-i4l1s01"} 134
infiniband_switch_power_supply_watts{guid="0x7cfe9003009ce5b0",psu="0",switch="ib-i1l1s01"} 72
infiniband_switch_power_supply_watts{guid="0x7cfe9003009ce5b0",psu="1",switch="ib-i1l1s01"} 71
# HELP infiniband_switch_temperature_celsius Switch ASIC temperature in degrees Celsius.
# TYPE infiniband_switch_temperature_celsius gauge
infiniband_switch_temperature_celsius{guid="0x506b4b03005c2740",switch="ib-i4l1s01"} 53
infiniband_switch_temperature_celsius{guid="0x7cfe9003009ce5b0",switch="ib-i1l1s01"} 45`
	expectedHCA = `# HELP infiniband_hca_info Constant 1 carrying HCA identification labels (lid, guid, hca name).
# TYPE infiniband_hca_info gauge
infiniband_hca_info{guid="0x506b4b0300cc02a6",hca="p0001 HCA-1",lid="1432"} 1
infiniband_hca_info{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",lid="133"} 1
infiniband_hca_info{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",lid="134"} 1
# HELP infiniband_hca_port_excessive_buffer_overrun_errors_total Excessive buffer overrun errors — receive buffer overran the configured threshold.
# TYPE infiniband_hca_port_excessive_buffer_overrun_errors_total counter
infiniband_hca_port_excessive_buffer_overrun_errors_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch="ib-i1l1s01"} 0
infiniband_hca_port_excessive_buffer_overrun_errors_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch="ib-i1l1s01"} 0
# HELP infiniband_hca_port_link_downed_total Times the link error recovery process failed and the link went down.
# TYPE infiniband_hca_port_link_downed_total counter
infiniband_hca_port_link_downed_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch="ib-i1l1s01"} 0
infiniband_hca_port_link_downed_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch="ib-i1l1s01"} 0
# HELP infiniband_hca_port_link_error_recovery_total Times the link successfully completed the link error recovery process.
# TYPE infiniband_hca_port_link_error_recovery_total counter
infiniband_hca_port_link_error_recovery_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch="ib-i1l1s01"} 0
infiniband_hca_port_link_error_recovery_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch="ib-i1l1s01"} 0
# HELP infiniband_hca_port_local_link_integrity_errors_total Local link integrity threshold errors (LocalLinkIntegrityErrors).
# TYPE infiniband_hca_port_local_link_integrity_errors_total counter
infiniband_hca_port_local_link_integrity_errors_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch="ib-i1l1s01"} 0
infiniband_hca_port_local_link_integrity_errors_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch="ib-i1l1s01"} 0
# HELP infiniband_hca_port_multicast_receive_packets_total Total multicast packets received on this port.
# TYPE infiniband_hca_port_multicast_receive_packets_total counter
infiniband_hca_port_multicast_receive_packets_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch="ib-i1l1s01"} 3.732373137e+09
infiniband_hca_port_multicast_receive_packets_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch="ib-i1l1s01"} 3.732158589e+09
# HELP infiniband_hca_port_multicast_transmit_packets_total Total multicast packets transmitted on this port.
# TYPE infiniband_hca_port_multicast_transmit_packets_total counter
infiniband_hca_port_multicast_transmit_packets_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch="ib-i1l1s01"} 544690
infiniband_hca_port_multicast_transmit_packets_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch="ib-i1l1s01"} 721488
# HELP infiniband_hca_port_qp1_dropped_total Subnet management QP1 packets dropped (QP1Dropped).
# TYPE infiniband_hca_port_qp1_dropped_total counter
infiniband_hca_port_qp1_dropped_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch="ib-i1l1s01"} 0
infiniband_hca_port_qp1_dropped_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch="ib-i1l1s01"} 0
# HELP infiniband_hca_port_rate_bytes_per_second Effective HCA port rate in bytes per second (after IB encoding overhead removed).
# TYPE infiniband_hca_port_rate_bytes_per_second gauge
infiniband_hca_port_rate_bytes_per_second{guid="0x506b4b0300cc02a6",hca="p0001 HCA-1"} 1.25e+10
infiniband_hca_port_rate_bytes_per_second{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1"} 1.25e+10
infiniband_hca_port_rate_bytes_per_second{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1"} 1.25e+10
# HELP infiniband_hca_port_raw_rate_bytes_per_second Raw HCA port rate in bytes per second (signaling rate, before encoding overhead).
# TYPE infiniband_hca_port_raw_rate_bytes_per_second gauge
infiniband_hca_port_raw_rate_bytes_per_second{guid="0x506b4b0300cc02a6",hca="p0001 HCA-1"} 1.2890625e+10
infiniband_hca_port_raw_rate_bytes_per_second{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1"} 1.2890625e+10
infiniband_hca_port_raw_rate_bytes_per_second{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1"} 1.2890625e+10
# HELP infiniband_hca_port_receive_constraint_errors_total Inbound packets discarded because of a partitioning or rate-limit constraint.
# TYPE infiniband_hca_port_receive_constraint_errors_total counter
infiniband_hca_port_receive_constraint_errors_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch="ib-i1l1s01"} 0
infiniband_hca_port_receive_constraint_errors_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch="ib-i1l1s01"} 0
# HELP infiniband_hca_port_receive_data_bytes_total Total data octets received on this port (perfquery PortRcvData scaled to bytes — IB octets are 4-byte words).
# TYPE infiniband_hca_port_receive_data_bytes_total counter
infiniband_hca_port_receive_data_bytes_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch="ib-i1l1s01"} 1.4890160781154e+14
infiniband_hca_port_receive_data_bytes_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch="ib-i1l1s01"} 3.90099383532e+13
# HELP infiniband_hca_port_receive_errors_total Errors detected on receive packets for any reason (PortRcvErrors).
# TYPE infiniband_hca_port_receive_errors_total counter
infiniband_hca_port_receive_errors_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch="ib-i1l1s01"} 0
infiniband_hca_port_receive_errors_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch="ib-i1l1s01"} 0
# HELP infiniband_hca_port_receive_packets_total Total packets received on this port (any size, any traffic class).
# TYPE infiniband_hca_port_receive_packets_total counter
infiniband_hca_port_receive_packets_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch="ib-i1l1s01"} 1.00583719365e+11
infiniband_hca_port_receive_packets_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch="ib-i1l1s01"} 3.3038722564e+10
# HELP infiniband_hca_port_receive_remote_physical_errors_total Receive errors caused by a remote physical-layer error (e.g. EBP marker).
# TYPE infiniband_hca_port_receive_remote_physical_errors_total counter
infiniband_hca_port_receive_remote_physical_errors_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch="ib-i1l1s01"} 0
infiniband_hca_port_receive_remote_physical_errors_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch="ib-i1l1s01"} 0
# HELP infiniband_hca_port_receive_switch_relay_errors_total Packets dropped during switch routing because no relay path was available.
# TYPE infiniband_hca_port_receive_switch_relay_errors_total counter
infiniband_hca_port_receive_switch_relay_errors_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch="ib-i1l1s01"} 0
infiniband_hca_port_receive_switch_relay_errors_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch="ib-i1l1s01"} 0
# HELP infiniband_hca_port_symbol_error_total Minor link errors detected on one or more physical lanes (SymbolErrorCounter).
# TYPE infiniband_hca_port_symbol_error_total counter
infiniband_hca_port_symbol_error_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch="ib-i1l1s01"} 0
infiniband_hca_port_symbol_error_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch="ib-i1l1s01"} 0
# HELP infiniband_hca_port_transmit_constraint_errors_total Outbound packets discarded because of a partitioning or rate-limit constraint.
# TYPE infiniband_hca_port_transmit_constraint_errors_total counter
infiniband_hca_port_transmit_constraint_errors_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch="ib-i1l1s01"} 0
infiniband_hca_port_transmit_constraint_errors_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch="ib-i1l1s01"} 0
# HELP infiniband_hca_port_transmit_data_bytes_total Total data octets transmitted on this port (perfquery PortXmitData scaled to bytes — IB octets are 4-byte words).
# TYPE infiniband_hca_port_transmit_data_bytes_total counter
infiniband_hca_port_transmit_data_bytes_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch="ib-i1l1s01"} 1.4843470741542e+14
infiniband_hca_port_transmit_data_bytes_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch="ib-i1l1s01"} 3.6198369975904e+13
# HELP infiniband_hca_port_transmit_discards_total Outbound packets discarded because the port was busy or down (PortXmitDiscards).
# TYPE infiniband_hca_port_transmit_discards_total counter
infiniband_hca_port_transmit_discards_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch="ib-i1l1s01"} 0
infiniband_hca_port_transmit_discards_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch="ib-i1l1s01"} 0
# HELP infiniband_hca_port_transmit_packets_total Total packets transmitted on this port (any size, any traffic class).
# TYPE infiniband_hca_port_transmit_packets_total counter
infiniband_hca_port_transmit_packets_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch="ib-i1l1s01"} 9.691711732e+10
infiniband_hca_port_transmit_packets_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch="ib-i1l1s01"} 2.8825338611e+10
# HELP infiniband_hca_port_transmit_wait_total Time ticks during which the port had data to transmit but no flow-control credits available — primary congestion signal.
# TYPE infiniband_hca_port_transmit_wait_total counter
infiniband_hca_port_transmit_wait_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch="ib-i1l1s01"} 0
infiniband_hca_port_transmit_wait_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch="ib-i1l1s01"} 0
# HELP infiniband_hca_port_unicast_receive_packets_total Total unicast packets received on this port.
# TYPE infiniband_hca_port_unicast_receive_packets_total counter
infiniband_hca_port_unicast_receive_packets_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch="ib-i1l1s01"} 9.6851346228e+10
infiniband_hca_port_unicast_receive_packets_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch="ib-i1l1s01"} 2.9306563974e+10
# HELP infiniband_hca_port_unicast_transmit_packets_total Total unicast packets transmitted on this port.
# TYPE infiniband_hca_port_unicast_transmit_packets_total counter
infiniband_hca_port_unicast_transmit_packets_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch="ib-i1l1s01"} 9.691657263e+10
infiniband_hca_port_unicast_transmit_packets_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch="ib-i1l1s01"} 2.8824617123e+10
# HELP infiniband_hca_port_vl15_dropped_total Subnet management packets (VL15) dropped because of resource limitations.
# TYPE infiniband_hca_port_vl15_dropped_total counter
infiniband_hca_port_vl15_dropped_total{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch="ib-i1l1s01"} 0
infiniband_hca_port_vl15_dropped_total{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch="ib-i1l1s01"} 0
# HELP infiniband_hca_up 1 if the latest perfquery scrape of this HCA succeeded, 0 otherwise (timeout or error).
# TYPE infiniband_hca_up gauge
infiniband_hca_up{guid="0x506b4b0300cc02a6",hca="p0001 HCA-1"} 1
infiniband_hca_up{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1"} 1
infiniband_hca_up{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1"} 1
# HELP infiniband_hca_uplink_info Constant 1 describing the switch port this HCA port is connected to.
# TYPE infiniband_hca_uplink_info gauge
infiniband_hca_uplink_info{guid="0x506b4b0300cc02a6",hca="p0001 HCA-1",port="1",switch="ib-i4l1s01",uplink="ib-i4l1s01",uplink_guid="0x506b4b03005c2740",uplink_lid="2052",uplink_port="35",uplink_type="SW"} 1
infiniband_hca_uplink_info{guid="0x7cfe9003003b4b96",hca="o0002 HCA-1",port="1",switch="ib-i1l1s01",uplink="ib-i1l1s01",uplink_guid="0x7cfe9003009ce5b0",uplink_lid="1719",uplink_port="11",uplink_type="SW"} 1
infiniband_hca_uplink_info{guid="0x7cfe9003003b4bde",hca="o0001 HCA-1",port="1",switch="ib-i1l1s01",uplink="ib-i1l1s01",uplink_guid="0x7cfe9003009ce5b0",uplink_lid="1719",uplink_port="10",uplink_type="SW"} 1`
	expectedSwitchNoError = `# HELP infiniband_exporter_collect_errors Number of errors that occurred during collection
# TYPE infiniband_exporter_collect_errors gauge
infiniband_exporter_collect_errors{collector="ibnetdiscover-runonce"} 0
infiniband_exporter_collect_errors{collector="switch-runonce"} 0
# HELP infiniband_exporter_collect_timeouts Number of timeouts that occurred during collection
# TYPE infiniband_exporter_collect_timeouts gauge
infiniband_exporter_collect_timeouts{collector="ibnetdiscover-runonce"} 0
infiniband_exporter_collect_timeouts{collector="switch-runonce"} 0`
	expectedFullNoError = `# HELP infiniband_exporter_collect_errors Number of errors that occurred during collection
# TYPE infiniband_exporter_collect_errors gauge
infiniband_exporter_collect_errors{collector="hca-runonce"} 0
infiniband_exporter_collect_errors{collector="ibnetdiscover-runonce"} 0
infiniband_exporter_collect_errors{collector="switch-runonce"} 0
# HELP infiniband_exporter_collect_timeouts Number of timeouts that occurred during collection
# TYPE infiniband_exporter_collect_timeouts gauge
infiniband_exporter_collect_timeouts{collector="hca-runonce"} 0
infiniband_exporter_collect_timeouts{collector="ibnetdiscover-runonce"} 0
infiniband_exporter_collect_timeouts{collector="switch-runonce"} 0`
	expectedIbnetdiscoverError = `# HELP infiniband_exporter_collect_errors Number of errors that occurred during collection
# TYPE infiniband_exporter_collect_errors gauge
infiniband_exporter_collect_errors{collector="ibnetdiscover-runonce"} 1
# HELP infiniband_exporter_collect_timeouts Number of timeouts that occurred during collection
# TYPE infiniband_exporter_collect_timeouts gauge
infiniband_exporter_collect_timeouts{collector="ibnetdiscover-runonce"} 0`
)

func TestMain(m *testing.M) {
	w := os.Stderr
	logger := slog.New(slog.NewTextHandler(w, nil))
	collectors.IbnetdiscoverExec = func(ctx context.Context) (string, error) {
		out, err := collectors.ReadFixture("ibnetdiscover", "test")
		if err != nil {
			logger.Error("error", "err", err)
			os.Exit(1)
		}
		return out, nil
	}
	collectors.PerfqueryExec = func(guid string, port string, extraArgs []string, ctx context.Context) (string, error) {
		out, err := collectors.ReadFixture("perfquery", guid)
		if err != nil {
			logger.Error("error", "err", err)
			os.Exit(1)
		}
		return out, nil
	}
	collectors.IbswinfoExec = func(lid string, vitals bool, ctx context.Context) (string, error) {
		if lid == "1719" {
			out, err := collectors.ReadFixture("ibswinfo", "test1")
			return out, err
		} else if lid == "2052" {
			out, err := collectors.ReadFixture("ibswinfo", "test2")
			return out, err
		} else {
			return "", nil
		}
	}
	exitVal := m.Run()
	os.Exit(exitVal)
}

func TestCollectToFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp(os.TempDir(), "output")
	if err != nil {
		os.Exit(1)
	}
	outputPath = tmpDir + "/output"
	defer os.RemoveAll(tmpDir)
	if _, err := kingpin.CommandLine.Parse([]string{fmt.Sprintf("--exporter.output=%s", outputPath), "--exporter.runonce"}); err != nil {
		t.Fatal(err)
	}
	err = run(slog.New(slog.DiscardHandler))
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
		return
	}
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
		return
	}
	if !strings.Contains(string(content), expectedSwitch) {
		t.Errorf("Unexpected content:\nExpected:\n%s\nGot:\n%s", expectedSwitch, string(content))
	}
	if !strings.Contains(string(content), expectedSwitchNoError) {
		t.Errorf("Unexpected error content:\nExpected:\n%s\nGot:\n%s", expectedSwitchNoError, string(content))
	}
	if !strings.Contains(string(content), "infiniband_exporter_last_execution") {
		t.Errorf("Unexpected error content:\nExpected: infiniband_exporter_last_execution\nGot:\n%s", string(content))
	}
}

func TestCollect(t *testing.T) {
	var err error
	if _, err = kingpin.CommandLine.Parse([]string{fmt.Sprintf("--web.listen-address=%s", address)}); err != nil {
		t.Fatal(err)
	}
	go func() {
		err = run(slog.New(slog.DiscardHandler))
	}()
	if err != nil {
		t.Fatal(err)
	}
	body, err := queryExporter(metricsEndpoint)
	if err != nil {
		t.Fatalf("Unexpected error GET %s: %s", metricsEndpoint, err.Error())
	}
	if !strings.Contains(body, expectedSwitch) {
		t.Errorf("Unexpected body\nExpected:\n%s\nGot:\n%s\n", expectedSwitch, body)
	}
	// remove -runonce collector suffix
	runonceRe := regexp.MustCompile("-runonce")
	expectedSwitchNoError = runonceRe.ReplaceAllString(expectedSwitchNoError, "")
	if !strings.Contains(body, expectedSwitchNoError) {
		t.Errorf("Unexpected body\nExpected:\n%s\nGot:\n%s\n", expectedSwitchNoError, body)
	}
	if _, err = kingpin.CommandLine.Parse([]string{"--no-collector.switch", "--collector.ibswinfo", fmt.Sprintf("--web.listen-address=%s", address)}); err != nil {
		t.Fatal(err)
	}
	body, err = queryExporter(metricsEndpoint)
	if err != nil {
		t.Fatalf("Unexpected error GET %s: %s", metricsEndpoint, err.Error())
	}
	if !strings.Contains(body, expectedIbswinfo) {
		t.Errorf("Unexpected body\nExpected:\n%s\nGot:\n%s\n", expectedIbswinfo, body)
	}
	if _, err = kingpin.CommandLine.Parse([]string{"--collector.hca", fmt.Sprintf("--web.listen-address=%s", address)}); err != nil {
		t.Fatal(err)
	}
	body, err = queryExporter(metricsEndpoint)
	if err != nil {
		t.Fatalf("Unexpected error GET %s: %s", metricsEndpoint, err.Error())
	}
	if !strings.Contains(body, expectedHCA) {
		t.Errorf("Unexpected body\nExpected:\n%s\nGot:\n%s\n", expectedHCA, body)
	}
	expectedFullNoError = runonceRe.ReplaceAllString(expectedFullNoError, "")
	if !strings.Contains(body, expectedFullNoError) {
		t.Errorf("Unexpected body\nExpected:\n%s\nGot:\n%s\n", expectedFullNoError, body)
	}
	collectors.IbnetdiscoverExec = func(ctx context.Context) (string, error) {
		return "", fmt.Errorf("Error")
	}
	if _, err = kingpin.CommandLine.Parse([]string{"--web.disable-exporter-metrics", fmt.Sprintf("--web.listen-address=%s", address)}); err != nil {
		t.Fatal(err)
	}
	body, err = queryExporter(metricsEndpoint)
	if err != nil {
		t.Fatalf("Unexpected error GET %s: %s", metricsEndpoint, err.Error())
	}
	// Strip noise we cannot pin: per-scrape duration, and the always-on
	// build_info collector (registered as part of the standard exporter
	// surface from v0.14.0 onward).
	re := regexp.MustCompile(`(?m)^.*infiniband_exporter_collector_duration_seconds.*$`)
	body = re.ReplaceAllString(body, "")
	buildInfoRe := regexp.MustCompile(`(?m)^.*go_build_info.*$`)
	body = buildInfoRe.ReplaceAllString(body, "")
	body = strings.TrimSpace(body)
	body = regexp.MustCompile(`\n{2,}`).ReplaceAllString(body, "\n")
	expectedIbnetdiscoverError = runonceRe.ReplaceAllString(expectedIbnetdiscoverError, "")
	if body != expectedIbnetdiscoverError {
		t.Errorf("Unexpected body\nExpected:\n%s\nGot:\n%s\n", expectedIbnetdiscoverError, body)
	}
}

func TestBaseURL(t *testing.T) {
	body, err := queryExporter("")
	if err != nil {
		t.Fatalf("Unexpected error GET base URL: %s", err.Error())
	}
	if !strings.Contains(body, metricsEndpoint) {
		t.Errorf("Unexpected body\nExpected: /metrics\nGot:\n%s\n", body)
	}
}

func queryExporter(path string) (string, error) {
	// run() is started in a goroutine by TestCollect with no readiness
	// signal, so the HTTP server may not be bound yet on the first call.
	// Retry briefly to absorb that race.
	var resp *http.Response
	var err error
	url := fmt.Sprintf("http://%s%s", address, path)
	for i := 0; i < 20; i++ {
		resp, err = http.Get(url)
		if err == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err != nil {
		return "", err
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if err := resp.Body.Close(); err != nil {
		return "", err
	}
	if want, have := http.StatusOK, resp.StatusCode; want != have {
		return "", fmt.Errorf("want /metrics status code %d, have %d. Body:\n%s", want, have, b)
	}
	return string(b), nil
}
