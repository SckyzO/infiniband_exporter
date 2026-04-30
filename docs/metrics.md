# Metrics reference

All metrics use the `infiniband` namespace. `infiniband_exporter_*` are
exporter-internal; the rest reflect the IB fabric.

Run `curl http://<host>:9315/metrics` against a live exporter for the
authoritative list including HELP texts. This document is the human
walkthrough.

## Per-port counters

Twenty-eight IB Performance Management counters, exposed twice — once
under `infiniband_switch_*` (label set: `guid`, `port`, `switch`) and
once under `infiniband_hca_*` (label set: `guid`, `hca`, `port`,
`switch`). The first 22 come from `perfquery -x`; the last 6 are the
`-E` (PortRcvErrorDetails) set, only emitted when
`--collector.switch.rcv-err-details` (or the HCA equivalent) is enabled.

| Metric base | Type | Notes |
| --- | --- | --- |
| `port_transmit_data_bytes_total` | counter | perfquery `PortXmitData` × 4 (IB octets are 4-byte words) |
| `port_receive_data_bytes_total` | counter | `PortRcvData` × 4 |
| `port_transmit_packets_total` / `port_receive_packets_total` | counter | All packets, all traffic classes |
| `port_unicast_{transmit,receive}_packets_total` | counter | |
| `port_multicast_{transmit,receive}_packets_total` | counter | |
| `port_symbol_error_total` | counter | Lane-level decode errors (link health) |
| `port_link_error_recovery_total` | counter | Recovery attempts that succeeded |
| `port_link_downed_total` | counter | Recovery failures = link actually went down |
| `port_receive_errors_total` | counter | Catch-all rcv error |
| `port_receive_remote_physical_errors_total` | counter | EBP markers — peer signaled an error |
| `port_receive_switch_relay_errors_total` | counter | Relay/routing failure |
| `port_transmit_discards_total` | counter | Drop because target busy/down |
| `port_{transmit,receive}_constraint_errors_total` | counter | Partition / rate-limit violation |
| `port_local_link_integrity_errors_total` | counter | |
| `port_excessive_buffer_overrun_errors_total` | counter | |
| `port_vl15_dropped_total` | counter | SM packets dropped |
| `port_transmit_wait_total` | counter | **Primary congestion signal**: ticks with data to send but no flow-control credits |
| `port_qp1_dropped_total` | counter | SM QP1 drops |
| `port_local_physical_errors_total` | counter | rcv-err-details only |
| `port_malformed_packet_errors_total` | counter | rcv-err-details only |
| `port_buffer_overrun_errors_total` | counter | rcv-err-details only |
| `port_dlid_mapping_errors_total` | counter | rcv-err-details only — DLID has no valid mapping |
| `port_vl_mapping_errors_total` | counter | rcv-err-details only — invalid SL→VL mapping |
| `port_looping_errors_total` | counter | rcv-err-details only |

## Per-port gauges

| Metric | Labels | Notes |
| --- | --- | --- |
| `infiniband_switch_port_rate_bytes_per_second` | guid, port, switch | Effective rate (after IB encoding overhead) |
| `infiniband_switch_port_raw_rate_bytes_per_second` | guid, port, switch | Raw signaling rate |
| `infiniband_hca_port_rate_bytes_per_second` | guid, hca | Same shape, HCA side |
| `infiniband_hca_port_raw_rate_bytes_per_second` | guid, hca | |
| `infiniband_switch_port_state` | guid, port, switch | 1 = link up, 0 = link down. Requires `--collector.switch.port-state`. |

## Identification (info metrics)

Always-1 gauges carrying identifying labels.

| Metric | Labels |
| --- | --- |
| `infiniband_switch_info` | `guid`, `switch`, `lid` |
| `infiniband_hca_info` | `guid`, `hca`, `lid` |
| `infiniband_switch_uplink_info` | `guid`, `port`, `switch`, `uplink`, `uplink_guid`, `uplink_type`, `uplink_port`, `uplink_lid` |
| `infiniband_hca_uplink_info` | symmetric for the HCA side |

## Per-device health

| Metric | Type | Labels | Notes |
| --- | --- | --- | --- |
| `infiniband_switch_up` | gauge | guid, switch | 1 if last perfquery succeeded, 0 on error/timeout |
| `infiniband_hca_up` | gauge | guid, hca | symmetric |
| `infiniband_ibswinfo_up` | gauge | guid, switch | for ibswinfo collection |
| `infiniband_switch_collect_duration_seconds` | gauge | guid, collector | per-device collect time |
| `infiniband_switch_collect_error` / `_timeout` | gauge | guid, collector | per-device error/timeout flag |
| (same for `hca` and `ibswinfo` subsystems) | | | |

## ibswinfo (BETA, requires `--collector.ibswinfo`)

| Metric | Labels | Notes |
| --- | --- | --- |
| `infiniband_switch_hardware_info` | guid, firmware_version, psid, part_number, serial_number, switch | Always-1 info |
| `infiniband_switch_uptime_seconds` | guid, switch | |
| `infiniband_switch_temperature_celsius` | guid, switch | |
| `infiniband_switch_fan_status_info` | guid, status, switch | Always-1, `status` is the firmware string |
| `infiniband_switch_fan_rpm` | guid, fan, switch | |
| `infiniband_switch_power_supply_status_info` | guid, psu, status, switch | |
| `infiniband_switch_power_supply_dc_power_status_info` | guid, psu, status, switch | |
| `infiniband_switch_power_supply_fan_status_info` | guid, psu, status, switch | |
| `infiniband_switch_power_supply_watts` | guid, psu, switch | |

## Exporter internals

| Metric | Notes |
| --- | --- |
| `infiniband_exporter_collector_duration_seconds{collector}` | Collector-level latency |
| `infiniband_exporter_collect_errors{collector}` | Errors during the last scrape |
| `infiniband_exporter_collect_timeouts{collector}` | Timeouts during the last scrape |
| `infiniband_exporter_last_execution{collector}` | Unix timestamp of last successful runonce execution |
| `go_*`, `process_*`, `promhttp_*` | Stdlib/runtime self-metrics |
| `go_build_info` | Always present, exposes the running version + revision |
