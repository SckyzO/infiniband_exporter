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

## Required exporter flags

Each alert in the catalogue depends on a specific source metric, which
in turn requires a particular collector to be enabled. Cross-check
your exporter command line against this matrix before importing the
rules — an alert whose source metric does not exist will silently
never fire.

| Alert(s) | Source metric | Required exporter flag(s) |
| --- | --- | --- |
| `IBSwitchScrapeFailing` | `infiniband_switch_up` | `--collector.switch` (default: enabled) |
| `IBHCAScrapeFailing` | `infiniband_hca_up` | `--collector.hca` (default: enabled since 2.0) |
| `IBSwitchPortDown` | `infiniband_switch_port_state` | `--collector.switch.port-state` (default: enabled since 2.0) |
| `IBPortStateMetricMissing` | absence of the above | none — meta-alert that catches the case where the flag is missing |
| `IBPortLinkDownedRising`, `IBPortSymbolErrorBurst`, `IBPortRcvErrorRate`, `IBPortXmitDiscardRate`, `IBPortCongestionElevated` | per-port perfquery counters | `--collector.switch` + `--collector.switch.base-metrics` (both default: enabled) |
| `IBSwitchTempHigh`, `IBSwitchTempCritical`, `IBPSUFailure`, `IBSwitchFanFailed` | `infiniband_switch_*` from ibswinfo | `--collector.ibswinfo` (default: disabled) and `ibswinfo.sh` resolvable via `--ibswinfo.path` |
| `IBExporterDown` | `up` | none — Prometheus-side meta-metric |
| `IBExporterScrapeStale` | `infiniband_exporter_last_execution` | `--exporter.runonce` mode only |

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
| `IBPortStateMetricMissing` | warning | `absent_over_time(infiniband_switch_port_state[30m])` for 30 m |

The `IBSwitchPortDown` alert pairs `port_state == 0` with the
`infiniband:switch_port_ever_connected` recording rule
(`max_over_time(...[7d]) == 1`) so we don't page on ports that have
never been wired. Requires `--collector.switch.port-state` on the
exporter (default enabled since 2.0).

`IBPortStateMetricMissing` (added in 1.1) is the safety net for that
last requirement: it fires when the `port_state` series has not been
seen anywhere in the fabric for 30 minutes, which is what happens if
operators deploy these rules but forget the flag — without this
catch, `IBSwitchPortDown` is silently inoperative.

| `IBHCAScrapeErrorRateElevated` | info | `increase(infiniband_exporter_collect_errors_total{collector="hca"}[1h]) > 5` for 30 m |

`IBHCAScrapeErrorRateElevated` (added with the v2.0 alert pack) is
an info-level signal that more than 1 % of HCA scrapes over the
last hour reported errors. Most often these are transient
`_do_madrpc: recv failed` MAD timeouts. The alert is meant to
surface drift, not to page. Mitigate with `--perfquery.retries=1`
on the exporter (added in 2.0).

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

## Adapt to your setup before deploying

A few alerts encode assumptions about how Prometheus is configured. Read
this section before importing.

### `IBExporterDown` and the `job` label

The default expression is `up{job=~".*infiniband.*"} == 0`. The regex
matches any Prometheus job whose name contains the string `infiniband`
— common names like `infiniband`, `infiniband-exporter`, `ib-fabric`
are caught.

If your job is named differently (e.g. just `ib`), edit the alert to
use the exact label value:

```yaml
expr: up{job="ib"} == 0
```

### `IBSwitchPortDown` requires the recording rule **and** the flag

This alert pairs `infiniband_switch_port_state == 0` with
`infiniband:switch_port_ever_connected` (a recording rule that flips
on once the port has been seen up). It needs **both**:

* The exporter started with `--collector.switch.port-state` (default
  enabled since 2.0; pass `--no-collector.switch.port-state` to opt
  out and the alert becomes inoperative).
* `infiniband_recording.yml` loaded by Prometheus — without
  `infiniband:switch_port_ever_connected` the right-hand side of the
  `and` is empty and the alert never fires.

If you forget either of these, alerts silently never trigger. There
is no log message — that is the point of the design (no false
positives on never-cabled ports), but it does mean you should sanity
check by querying both names in Prometheus once during setup.

### `IBExporterScrapeStale` only fires in runonce mode

`infiniband_exporter_last_execution` is only exposed when the
exporter runs as `--exporter.runonce`. In HTTP scrape mode the
metric is absent, so `time() - last_execution > 300` evaluates to
no data and the alert never fires.

That is fine — in HTTP mode `IBExporterDown` covers the same
failure (Prometheus can't scrape it). Keep both rules; they
target different operational modes.

### ibswinfo and node_exporter alerts

* The environment alerts (`IBSwitchTempHigh`, `IBPSUFailure`,
  `IBSwitchFanFailed`, …) require `--collector.ibswinfo`.
* The `node_exporter`-driven combo dashboard requires
  `node_exporter` ≥ 1.5 with `--collector.infiniband` enabled on the
  same host. None of the exporter's *own* alerts depend on
  `node_*` metrics — those are dashboard-only.
