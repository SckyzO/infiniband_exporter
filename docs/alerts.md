# Alerting guide

The example rules ship in [`examples/prometheus/rules/`](../examples/prometheus/rules/):

* `infiniband_recording.yml` — recording rules (rates, aggregations, the
  "this port has ever been up" predicate). Drop into your Prometheus
  `rule_files`.
* `infiniband_alerts.yml` — alert rules grouped by concern (exporter
  health, fabric health, errors, environment).

The thresholds are starting points based on the IB Performance
Management spec and are deliberately conservative. Tune for your
fabric — a high `SymbolErrorCounter` rate is normal on a freshly-cabled
link for a few minutes after the ASIC syncs.

## Severity convention

Three tiers, following Prometheus alerting docs:

| Severity | When to use | Page someone? |
| --- | --- | --- |
| `critical` | Operator action needed now | Yes |
| `warning`  | Investigate during business hours | Email/ticket |
| `info`     | Capacity-planning / tuning signal | Dashboard, no alert routing |

## Alert catalogue

### Exporter health

| Alert | Severity | Predicate |
| --- | --- | --- |
| `IBExporterDown` | critical | `up{job=~".*infiniband.*"} == 0` for 2 m |
| `IBExporterScrapeStale` | warning | textfile mode: `time() - infiniband_exporter_last_execution > 300` |

### Fabric health

| Alert | Severity | Predicate |
| --- | --- | --- |
| `IBSwitchScrapeFailing` | critical | `infiniband_switch_up == 0` for 5 m |
| `IBHCAScrapeFailing` | warning | `infiniband_hca_up == 0` for 5 m |
| `IBSwitchPortDown` | critical | `infiniband_switch_port_state == 0 and on() infiniband:switch_port_ever_connected` for 5 m |

The `IBSwitchPortDown` alert pairs `port_state == 0` with the
`infiniband:switch_port_ever_connected` recording rule
(`max_over_time(...[7d]) == 1`) so we don't page on ports that have
never been wired. **Requires `--collector.switch.port-state` on the
exporter.**

### Errors

| Alert | Severity | Threshold (tunable) |
| --- | --- | --- |
| `IBPortLinkDownedRising` | warning | LinkDownedCounter incremented in last 5 m |
| `IBPortSymbolErrorBurst` | warning | >100 symbol errors / minute for 10 m |
| `IBPortRcvErrorRate` | warning | >1 PortRcvError / s for 10 m |
| `IBPortXmitDiscardRate` | warning | >1 PortXmitDiscard / s for 10 m |
| `IBPortCongestionElevated` | info | PortXmitWait rate > 1e6 ticks/s for 30 m |

Symbol errors and link-downed counters are a strong predictor of
cable / SFP problems. A page on a single LinkDowned increment is
intentionally conservative — link flaps tend to escalate.

### Environment (require `--collector.ibswinfo`)

| Alert | Severity | Predicate |
| --- | --- | --- |
| `IBSwitchTempHigh` | warning | `temperature_celsius > 80` for 5 m |
| `IBSwitchTempCritical` | critical | `temperature_celsius > 90` for 1 m |
| `IBPSUFailure` | critical | `power_supply_status_info{status!~"OK\|ok"} == 1` |
| `IBSwitchFanFailed` | warning | `fan_status_info{status!~"OK\|ok"} == 1` |

Mellanox switches throttle around 95 °C — paging at 80 °C gives
operations time to react.

## Recording rules used

The dashboards use raw expressions (so they work on a vanilla
Prometheus). The recording rules are an optimization — install them
if your fabric is large enough that the dashboards become slow.

Notable rules:

* `infiniband:switch_transmit_bytes:rate5m` / `:receive_bytes:rate5m` —
  per-switch byte rate (drops the `port` label).
* `infiniband:fabric_{transmit,receive}_bytes:rate5m` — whole-fabric.
* `infiniband:switch_port_total_errors:rate5m` — composite of the
  headline error counters per port.
* `infiniband:switches_up:ratio` — fraction of switches reachable; good
  SLO target.
* `infiniband:switch_port_ever_connected` — used by `IBSwitchPortDown`.

See the comments in `infiniband_recording.yml` for the full list.
