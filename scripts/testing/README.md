# scripts/testing/ — fabric-side validation harness

A drop-in shell harness to exercise a built `infiniband_exporter` binary
against a real fabric. Used during pre-release validation; also useful
for "does my packaged build still work" smoke testing.

## What it does

`test_ib.sh` runs nine numbered tests covering:

| Test | Configuration | Verifies |
| --- | --- | --- |
| 0 | binary identity | `--version`, sha256 |
| 1 | `--help` | flag surface |
| 2 | switch + HCA, no ibswinfo | baseline scrape latency, `/healthz` |
| 3 | + ibswinfo, default cache (15 m) | static cache hit ⇒ vitals path |
| 4 | + ibswinfo, `--ibswinfo.static-cache-ttl=0` | regression baseline (every scrape full) |
| 5 | `--ibnetdiscover.cache-ttl=5m` | topology cache hit ⇒ `collector_duration` drops to 0 |
| 6 | `--collector.switch.port-state` | up/down series count distribution |
| 7 | `--exporter.runonce` | textfile mode + `last_execution` |
| 8 | full configuration | HELP/TYPE counts match, no metric family without samples |
| 9 | latency summary | one-line cold-vs-warm scrape comparison across configs |

## Non-disruptive guarantees

* Listens on `:19315` by default (overridable via `PORT=`); refuses to
  start if the port is already in use.
* Never `pkill`/`killall` — kills only PIDs it spawned.
* No `sudo`, no install, no system writes. All output under
  `${TEST_DIR}/work/`.
* `--ibswinfo.path` is set explicitly; the harness does not modify
  `$PATH`.
* `set -u` but not `-e`: an individual failing test does not stop the
  rest.

## Usage

On the host with InfiniBand tooling and an active fabric link:

```bash
mkdir -p /tmp/test_ib
# Drop these three files into /tmp/test_ib/:
#   * infiniband_exporter   (the binary you want to validate)
#   * ibswinfo.sh           (helper from stanford-rc/ibswinfo)
#   * test_ib.sh            (this script)

cd /tmp/test_ib
chmod +x test_ib.sh ibswinfo.sh infiniband_exporter
bash test_ib.sh > output.log 2>&1
```

`output.log` is the captured stdout/stderr you'd typically attach to a
release-validation issue. The `work/` directory holds per-test logs and
metric dumps:

```
work/
├── tN.log         # exporter logs per test
├── tN.metrics     # /metrics dump per test (or t3.1.metrics, t3.2.metrics, …)
├── runonce.prom   # textfile output of TEST 7
└── runonce.lock   # released after TEST 7
```

## Knobs

Override via environment variables:

| Var | Default | Use |
| --- | --- | --- |
| `TEST_DIR` | `/tmp/test_ib` | layout root |
| `PORT` | `19315` | listen port |
| `PERFQUERY_CONCURRENCY` | `8` | `--perfquery.max-concurrent` |
| `IBSWINFO_CONCURRENCY` | `8` | `--ibswinfo.max-concurrent` |

Bigger fabric? Bump the concurrencies:

```bash
PERFQUERY_CONCURRENCY=16 IBSWINFO_CONCURRENCY=8 bash test_ib.sh > out.log 2>&1
```

## Reading the output

* **Cache validation** (TEST 3 vs TEST 4): the per-switch
  `infiniband_ibswinfo_collect_duration_seconds` value should be lower
  on the warm scrape (TEST 3 scrape #2/#3) than on TEST 4. Even more
  diagnostic: `infiniband_switch_power_supply_status_info`,
  `infiniband_switch_power_supply_dc_power_status_info`,
  `infiniband_switch_power_supply_fan_status_info`, and
  `infiniband_switch_fan_status_info` **must drop to 0 series** on warm
  scrapes — the vitals output does not carry status fields. If those
  series are still present on warm scrapes, the cache is broken.

* **TEST 5 ibnetdiscover cache**:
  `infiniband_exporter_collector_duration_seconds{collector="ibnetdiscover"}`
  goes from sub-second on the cold scrape to literally `0` on warm
  scrapes.

* **TEST 8 shape**: the section "families with no samples observed"
  should be empty. Anything listed there means the exporter declared a
  metric (`# HELP`/`# TYPE`) but never emitted a sample for it during
  the scrape — usually a bug.

## Ownership / who runs this

This is **not** part of CI. CI in `.github/workflows/test.yml` runs the
unit-test suite inside containers and never has fabric access.

`test_ib.sh` is for the maintainer (or a fabric operator wishing to
self-validate a build) to run on a bench server with real IB hardware
before tagging a release. The `make` targets do not invoke it.
