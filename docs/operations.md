# Operations guide

Operational guide for someone running `infiniband_exporter` in production.
For metrics reference see [metrics.md](metrics.md); for alerts and
dashboards see [alerts.md](alerts.md) and [dashboards.md](dashboards.md).

## Sizing perfquery and ibswinfo

The two flags that matter most on a non-trivial fabric:

* `--perfquery.max-concurrent` (default `1`). On a multi-core management
  host, **bump this to 8**. Leaving it at 1 makes the switch and HCA
  collectors run sequentially, which is fine on a 3-switch lab and
  catastrophic on a 100-switch fabric.
* `--ibswinfo.max-concurrent` (default `4`, raised from `1` in v0.15.0).
  ibswinfo takes ≈1.4 s per switch on HDR fabrics. With 4 in flight a
  20-switch fabric drops from ~32 s to ~8 s. Bump higher if your fabric
  is bigger and the SMA can take it.

Pair concurrency with the cache flag:

* `--ibswinfo.static-cache-ttl` (default `15m`). Keeps PartNumber /
  SerialNumber / PSID / FirmwareVersion in memory; while fresh, the
  exporter switches to `ibswinfo -d lid-X -o vitals` which only reads
  the dynamic registers. Set to `0` to reproduce pre-v0.15.0 behaviour.
  PSU / fan **status** strings are not cached — see the caveat in
  CHANGELOG v0.15.0.

* `--ibnetdiscover.cache-ttl` (default `0`, disabled). On fabrics where
  `ibnetdiscover` itself takes seconds, set to ~5 minutes so the parsed
  topology is reused between scrapes. perfquery counters are still
  re-collected every scrape.

## Permissions

The exporter shells out to `ibnetdiscover`, `perfquery`, and `ibswinfo`
— all three need access to `/dev/infiniband/umad*`. Two options:

### Option A — open the umad device (recommended)

```
$ cat /etc/udev/rules.d/99-ib.rules
KERNEL=="umad*", NAME="infiniband/%k" MODE="0666"
```

### Option B — wrap with sudo

Run with `--sudo` and a sudoers entry that whitelists exactly the
binaries the exporter calls:

```
Defaults:infiniband_exporter !syslog,!requiretty
infiniband_exporter ALL=(ALL) NOPASSWD: /usr/sbin/ibnetdiscover, /usr/sbin/perfquery, /usr/bin/ibswinfo
```

If the diagnostic tools are not on `$PATH`, point at them with
`--ibnetdiscover.path`, `--perfquery.path`, `--ibswinfo.path`.

## Runonce / textfile mode

Two cases where the HTTP scrape model breaks down:

1. The fabric is large enough that a full collection exceeds Prometheus's
   scrape timeout (typically 10–30 s).
2. You want `--collector.switch.rcv-err-details` (slow — one perfquery
   per port) without blocking every scrape.

Solution: run the exporter periodically in `--exporter.runonce` mode
and let `node_exporter`'s textfile collector serve the resulting file.

```
--exporter.runonce
--exporter.output=/var/lib/node_exporter/textfile_collector/infiniband_exporter.prom
```

Schedule via systemd timer or cron. The exporter takes a lock file
(`--exporter.lockfile`, default `/tmp/infiniband_exporter.lock`) so
overlapping runs are safely declined.

You can split base metrics and rcv-err details into two cron jobs:

```
# every minute, base metrics
infiniband_exporter --exporter.runonce \
    --exporter.output=/var/lib/node_exporter/textfile_collector/infiniband_exporter.prom \
    --collector.switch --collector.hca --perfquery.max-concurrent=8

# every 5 minutes, rcv-err details only
infiniband_exporter --exporter.runonce \
    --exporter.output=/var/lib/node_exporter/textfile_collector/infiniband_exporter_rcverr.prom \
    --no-collector.switch.base-metrics \
    --collector.switch.rcv-err-details \
    --perfquery.max-concurrent=8
```

## Troubleshooting

| Symptom | Likely cause |
| --- | --- |
| `infiniband_exporter_collect_errors{collector="ibnetdiscover"} > 0` | umad permissions, or `ibnetdiscover` not on `$PATH`. Check exporter logs for stderr from the binary (we surface it from v0.13.0 onward). |
| `infiniband_switch_up == 0` for a single switch | Management link to that switch is down, or perfquery times out for it. The other switches are unaffected. |
| Scrape time exceeds `scrape_timeout` | Bump `--perfquery.max-concurrent`; if still too slow, switch to runonce mode. |
| `ibswinfo` errors on some switches | Some switch firmware versions don't support all MFT registers. `--ibswinfo.exclude-node-name` (planned, see roadmap) lets you skip them. Until then, disable `--collector.ibswinfo` or live with the per-switch error metric. |
| `infiniband_switch_port_state` always returns no series | The flag is `--collector.switch.port-state` (default off) — enable it to surface up/down state. |

For fabric-specific bugs, please attach **anonymized** captures via
[`scripts/anonymize.sh`](../scripts/README.md). The bug-report issue
template walks you through it.
