# Migration guide — 1.x → 2.0

2.0 is the first release that intentionally breaks 1.x compatibility.
Three flavours of breaking change, all of them small, all of them
mechanical.

## 1. Renamed metrics (Counter type)

| 1.x (Gauge) | 2.0 (Counter) |
|---|---|
| `infiniband_exporter_collect_errors` | `infiniband_exporter_collect_errors_total` |
| `infiniband_exporter_collect_timeouts` | `infiniband_exporter_collect_timeouts_total` |

The old metrics were gauges that held the per-scrape count of errors /
timeouts. Each scrape reset them. The names suggested "counter", which
made `rate()` and `increase()` silently return wrong numbers — and
Prometheus printed a warning on every query.

In 2.0 they are true counters: cumulative since process start, suffix
`_total`, type `counter`. `rate()` and `increase()` work as you'd
expect.

### What you need to update

**Queries that used `> 0` to "match scrapes with errors" no longer
work** because a counter only goes up; once any error has ever
occurred it stays `> 0` forever. Rewrite them:

```promql
# Before (1.x, gauge semantics):
infiniband_exporter_collect_errors{collector="hca"} > 0

# After (2.0, counter semantics):
increase(infiniband_exporter_collect_errors_total{collector="hca"}[5m]) > 0
```

**Queries that used `sum_over_time(...)` to count errors over a
window become a single `increase()`:**

```promql
# Before:
sum_over_time(infiniband_exporter_collect_errors[1h])

# After:
increase(infiniband_exporter_collect_errors_total[1h])
```

The alert pack shipped in `examples/prometheus/rules/infiniband_alerts.yml`
has already been migrated. If you imported your own custom alerts
based on 1.x metric names, do a find-and-replace.

## 2. Default flag changes

Two flags flipped from `false` to `true`:

| Flag | 1.x default | 2.0 default |
|---|---|---|
| `--collector.hca` | `false` | **`true`** |
| `--collector.switch.port-state` | `false` | **`true`** |

### What you'll observe

* New metric series start appearing the moment you upgrade:
  - `infiniband_hca_*` (HCA counters)
  - `infiniband_switch_port_state{port=...}` (1=up, 0=down per port)
* Scrape duration grows by the time `perfquery` takes per HCA. With
  `--perfquery.max-concurrent=4` (the 2.0 default), 50 HCAs add
  about 1–2 seconds.
* `IBSwitchPortDown` and `IBPortLinkDownedRising`-style alerts that
  shipped in the rule pack but were silently inoperative now fire
  on real fabric events.

### How to disable (revert pre-2.0 behaviour)

Add the explicit `--no-` form to your systemd unit:

```bash
infiniband_exporter \
  --no-collector.hca \
  --no-collector.switch.port-state \
  ...
```

This is supported and will not be deprecated — flipping the default
just made the common case implicit.

## 3. Other notes that are not breaking but worth knowing

* `--perfquery.max-concurrent` default went from `1` to `4` in 1.1.
  If you set it explicitly, no change.
* `--ibnetdiscover.cache-ttl` default went from `0` to `5m` in 1.1.
* `--ibswinfo.static-cache-ttl` default went from `15m` to `5m` in 1.1.
* New flag `--perfquery.retries` (default `0`) lets you absorb
  transient `_do_madrpc: recv failed` errors. Recommended on noisy
  fabrics: `--perfquery.retries=1`.
* New metric `infiniband_exporter_collect_retries_total` (Counter)
  counts how often the retry path triggered, so you can tell
  whether your fabric is actually noisy.

## TL;DR

```bash
# Find queries that need rewriting:
grep -rE 'infiniband_exporter_collect_(errors|timeouts)\b' \
  prometheus-rules/ grafana-dashboards/

# Find queries that need a `> 0` rewrite:
grep -rE 'collect_errors\b.*>' prometheus-rules/

# Find users still relying on the old default flags being off:
grep -E '\-\-no\-collector\.(hca|switch\.port-state)' systemd/

# Find the new flag you might want to use:
echo "Consider: --perfquery.retries=1"
```
