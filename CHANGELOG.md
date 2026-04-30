## 1.0.0 / 2026-04-30

First stable release of the SckyzO fork. No new features over v0.19.0
— this tag is the polish pass: PromQL audit, comment dégraissage,
documentation review, dependency tidy.

### Stability commitment from 1.0 onward

* Metric names, label sets, and exit code semantics are part of the
  stable surface. Breaking changes will go through a deprecation
  cycle of at least one minor release.
* Flags follow the same rule. New optional flags are non-breaking;
  removing or reshaping an existing one will be flagged in the
  changelog and announced one minor in advance.
* `examples/grafana/*.json` and `examples/prometheus/rules/*.yml`
  are best-effort references; their content can shift between
  releases without a deprecation cycle. See [docs/dashboards.md](docs/dashboards.md).

### Changes since 0.19.0

* **Bug fix** — `ibswinfo` static-fields cache is now package-global.
  Previously it was a `sync.Map` field on `IbswinfoCollector`. Each
  Prometheus HTTP scrape rebuilds the collector via `setupGathers()`,
  so the per-instance cache was wiped every scrape and the
  `--ibswinfo.static-cache-ttl` knob effectively did nothing — the
  full `ibswinfo -d lid-X` was issued on every scrape regardless of
  TTL. Spotted during pre-1.0 fabric validation; the warm-scrape
  `power_supply_status_info` series (which only exist in the full
  output, not in `-o vitals`) gave it away. The cache now lives at
  the package level (mirrors `ibnetdiscoverCache`). New regression
  test `TestIbswinfoCollectorCacheSurvivesAcrossInstances` exercises
  two consecutive collector instances sharing the cache.
* PromQL audit pass on every recording rule, alert rule, and
  dashboard target. `promtool check rules` is clean. Every
  `infiniband_*` reference has a producer in the code; every
  `node_infiniband_*` reference matches `node_exporter` ≥ 1.5.
* `docs/alerts.md` extended with an "Adapt to your setup" section
  that documents three caveats found during the audit:
  - `IBExporterDown` regex must be tuned to your `job` label
  - `IBSwitchPortDown` requires both `--collector.switch.port-state`
    **and** the recording rules loaded
  - `IBExporterScrapeStale` only fires in `--exporter.runonce` mode
* Code comments tightened. Removed dated "in v0.X.Y" / "before
  v0.X.Y" wording where the documented behaviour is now stable.
* `go mod tidy` clean — no stale dependencies left over from the
  pre-1.0 churn.

### Tooling

* Containers in `Makefile` (build / test / lint / release) now run
  with `--user $(id -u):$(id -g)`, with `HOME` redirected to a
  writable bind-mount and `GOMODCACHE` / `GOCACHE` /
  `GOLANGCI_LINT_CACHE` exported accordingly. Artifacts (`dist/`,
  `infiniband_exporter`, `.build/cache/...`) are owned by the user
  who invoked `make`, not `root`.
* `scripts/testing/test_ib.sh` — fabric-side validation harness
  used during pre-release. Drives nine numbered tests against a
  real exporter binary, including cache-hit verification (warm
  scrape must drop status fields), runonce mode, and a one-liner
  latency comparison summary. Documented under
  [`scripts/testing/README.md`](scripts/testing/README.md). Not
  wired into CI — it needs real IB hardware.

### Highlights of the divergence from upstream

For anyone arriving from `treydock/infiniband_exporter` 0.10.0, the
shape of the tree they see at 1.0:

* Container-only build pipeline (Makefile + GitHub Actions +
  GoReleaser, no CircleCI / promu / Makefile.common).
* Go 1.26 + `log/slog` + `prometheus/common/promslog`.
* `--collector.switch.port-state`, `--ibnetdiscover.cache-ttl`,
  `--ibswinfo.static-cache-ttl` flags. Default
  `--ibswinfo.max-concurrent` raised to 4.
* Per-device `infiniband_*_up` gauges; `_collect_error` /
  `_collect_timeout` kept for transition.
* `infiniband_ibswinfo_collect_*` carved out from
  `infiniband_switch_collect_*` (label-set conflict fix).
* `port_dlid_mapping_errors_total` typo fixed; `hca_rate_*` moved
  to the `port_*` namespace for symmetry with the switch side.
* Race-free counters, surfaced stderr from ibnetdiscover/perfquery,
  per-iteration cancel of rcv-err contexts.
* `examples/grafana/`: six dashboards for Grafana 10+ (with
  `__inputs` / `__requires` so they import directly and publish on
  grafana.com), including a combo dashboard for management nodes.
* `scripts/anonymize.sh` + `scripts/capture-fixtures.sh` for safe
  bug reporting.
* `docs/{operations,metrics,alerts,dashboards}.md` and
  `CONTRIBUTING.md`.

The full per-lot trail is below.

## 0.19.0 / 2026-04-30

Alerting, dashboards, documentation. Last feature lot before v1.0.0.

### Added

* `--collector.switch.port-state` (default `false`): exposes `infiniband_switch_port_state{guid, switch, port}` gauge (1 = up, 0 = down). When enabled, the parser keeps `???` lines from `ibnetdiscover` instead of dropping them, so port-down alerts can rely on `port_state == 0` instead of fragile `absent()` recipes. Imported and adapted from upstream PR #37 (metfan1981).
* `examples/prometheus/rules/infiniband_recording.yml`: recording rules for byte/packet/error rates per port, per switch, and fabric-wide; plus the `infiniband:switch_port_ever_connected` predicate the `IBSwitchPortDown` alert depends on.
* `examples/prometheus/rules/infiniband_alerts.yml`: alert catalogue grouped by exporter health, fabric health, errors, and environment (PSU / fan / temperature). Severity tiers follow Prometheus alerting docs.
* `examples/grafana/`: six dashboards in Grafana 10+ format with `__inputs` / `__requires` (importable directly, publishable on grafana.com):
  - `infiniband-fabric-overview-small.json` (≤40 switches, per-switch lines)
  - `infiniband-fabric-overview-large.json` (40+ switches, heatmap + topk + min/avg/max)
  - `infiniband-switch-detail.json` and `infiniband-hca-detail.json` for drill-down
  - `infiniband-exporter-internals.json` for the exporter's own health
  - `infiniband_and_node_exporter.json` — combo dashboard for management nodes that run both exporters together (load, CPU, memory, FS, IB driver counters, scrape latency)
* `docs/operations.md`, `docs/metrics.md`, `docs/alerts.md`, `docs/dashboards.md`: operational guide, metric reference, alert walkthrough, dashboard install.
* `CONTRIBUTING.md`: dev workflow, conventions, release process.

### Changed

* `README.md` rewritten around quick-start / endpoints / collectors / configuration / docs sections. Removed the upstream-style multi-paragraph install procedure; refer to `docs/operations.md` instead.
* `examples/grafana-ib-fabric-{health,perf}.json` and `examples/infiniband.rules` removed in favour of the new `examples/grafana/` and `examples/prometheus/` layouts. The new dashboards keep the most useful upstream panels (top congested switches, top by packet rate, failed PSUs/fans) and add several new ones (fabric Tx/Rx aggregate, hardware inventory, port_state table).

## 0.18.0 / 2026-04-30

Test fixtures: convention rename + anonymization tooling + end-to-end test.
No runtime behaviour changes.

### Changed

* `collectors/fixtures/` renamed to `collectors/testdata/`. The Go toolchain
  recognizes `testdata` as a magic directory and skips it during `go build`,
  `go vet`, and most other walks. `ReadFixture` was updated to match.

### Added

* `scripts/anonymize.sh` — strip operational identifiers (GUIDs, switch /
  HCA names, MAC addresses) from `ibnetdiscover -p` and `perfquery -G ... -x`
  output before committing it as a fixture. Replacements are stable inside
  one invocation so the topology relations encoded in the input file are
  preserved.
* `scripts/capture-fixtures.sh` — capture raw `ibnetdiscover` and
  `perfquery` output from a fabric host. Pipe the result through
  `anonymize.sh` before committing under `collectors/testdata/integration/`.
* `scripts/README.md` documents both scripts and explicitly covers the
  bug-reporting workflow: a user opening an issue can capture, anonymize,
  and attach a reproducer without leaking their fabric topology.
* New `TestEndToEndPipeline` drives the parsing and emission pipeline
  through the existing fixtures. Asserts the expected metric families
  (`infiniband_switch_info`, `infiniband_*_up`, …) appear when every
  collector is wired into the same registry.

## 0.17.0 / 2026-04-29

Internal refactor: switch / HCA factorization, ibnetdiscover topology cache.
Zero externally-visible behaviour changes; metric names, labels, HELP texts,
and flag defaults are unchanged.

### Changed (internal)

* New `collectors/port_counters.go` carries the single-source-of-truth list of all 28 perfquery-derived port counters (`portCounterDefs`). Both `SwitchCollector` and `HCACollector` now build their `*prometheus.Desc` map via `buildPortCounterDescs(subsystem, labels)` and emit samples through `emitPortCounters(...)`. Adding a new IB counter is now one `portCounterDef` entry instead of four edits across two files.
* `switch.go` 405 → 228 lines, `hca.go` 420 → 225 lines (≈ 370 lines removed, replaced by 160 lines in the shared file).

### Added

* `--ibnetdiscover.cache-ttl` (default `0` — disabled, identical to pre-v0.17 behaviour). Setting a non-zero TTL caches the parsed topology process-globally so subsequent scrapes reuse it instead of re-running `ibnetdiscover`. Useful on large fabrics where `ibnetdiscover` takes seconds; perfquery counters are still re-collected on every scrape. New tests `TestIbnetdiscoverCache{Disabled,Hit,Expired}` cover the cache paths.

## 0.16.0 / 2026-04-29

Prometheus naming and HELP cleanup; per-device `_up`. Last batch of breaking
metric renames before v1.0.0.

### ⚠️ Breaking — metric renames

* `infiniband_switch_port_dli_mapping_errors_total` → `infiniband_switch_port_dlid_mapping_errors_total` (typo fix; same applies to the HCA collector).
* `infiniband_hca_rate_bytes_per_second` → `infiniband_hca_port_rate_bytes_per_second` (matches the `port_*` namespace already used everywhere else and mirrors the switch-side name).
* `infiniband_hca_raw_rate_bytes_per_second` → `infiniband_hca_port_raw_rate_bytes_per_second` (same reason).

Update any dashboards/alerting rules referencing the old names. The `port_dli_mapping_errors_total` typo had been there since the upstream introduction of the metric.

### Added

* `infiniband_switch_up{guid, switch}`, `infiniband_hca_up{guid, hca}`, `infiniband_ibswinfo_up{guid, switch}`. Each emits `1` if the latest scrape of that device succeeded and `0` if it timed out or errored. These are the idiomatic Prometheus signal for per-device health and let alerting rules use the `up == 0` pattern. They complement the existing `_collect_error{guid}` / `_collect_timeout{guid}` series — those are kept for now to avoid forcing an alerting rewrite at the same time as everything else.

### Changed — HELP texts

Every metric's `# HELP` is now a description of *what is being measured* rather than the underlying perfquery / ibswinfo identifier name. Example: `port_link_downed_total` was `"Infiniband switch port LinkDownedCounter"`, it is now `"Times the link error recovery process failed and the link went down."`. `port_transmit_wait_total` is annotated as the primary congestion signal. The label sets are unchanged.

## 0.15.0 / 2026-04-29

ibswinfo: parallelism + static-fields cache.

### Changed

* `--ibswinfo.max-concurrent` default raised from `1` to `4`. On a 23-switch HDR400 fabric, sequential scrape was ~32 s (≈1.4 s per call); 4 concurrent callers cut that to ~8 s without saturating the SMA on the switches. Override via `--ibswinfo.max-concurrent=N` if your fabric has different characteristics.

### Added

* `--ibswinfo.static-cache-ttl` (default `15m`). Caches the rarely-changing fields (`PartNumber`, `SerialNumber`, `PSID`, `FirmwareVersion`) per GUID. While the entry is fresh, the collector calls `ibswinfo -d lid-X -o vitals` instead of the full output — reads only the dynamic registers (`MGIR`/`MGPIR`/`MSPS`/`MTMP`/`MFCR`), much faster. Cached static fields are merged back into the emitted `infiniband_switch_hardware_info` metric so it stays available across scrapes.
* Set `--ibswinfo.static-cache-ttl=0` to disable the cache and reproduce pre-v0.15.0 behaviour exactly (every scrape is a full call).
* New parser `parseIbswinfoVitals` for the `-o vitals` output format (separator `:`, key set `uptime (sec)` / `psu<N>.power (W)` / `cur.temp (C)` / `fan#<N>.speed (rpm)`). The pre-existing `parse_ibswinfo` handles the full output.

### Caveats

* Status fields (`infiniband_switch_power_supply_status_info`, `infiniband_switch_power_supply_dc_power_status_info`, `infiniband_switch_power_supply_fan_status_info`, `infiniband_switch_fan_status_info`) are emitted **only on full scrapes** — they are absent from the `vitals` output and not stored in the cache (a status that changes mid-cycle should not be hidden by a 15 min cache). Between full scrapes these series go stale in Prometheus. To alert on PSU/fan failure with sub-15 min latency, lower `--ibswinfo.static-cache-ttl` or set it to `0`.
* The existing five `TestIbswinfoCollector*` tests now run with `--ibswinfo.static-cache-ttl=0` to keep their assertions stable. New tests (`TestParseIbswinfoVitals`, `TestIbswinfoCollectorCacheMissFull`, `TestIbswinfoCollectorCacheHitVitals`, `TestIbswinfoCollectorCacheExpired`, `TestIbswinfoCollectorCacheDisabled`) cover the cache paths.

## 0.14.0 / 2026-04-29

Modernization: Go 1.26, slog, dependency bumps, alignment with sibling
exporters' conventions.

### ⚠️ Breaking — endpoint convention

* `/internal/metrics` is removed. Go runtime / process / promhttp self-metrics return to `/metrics`, gated by `--web.disable-exporter-metrics` (which now skips registering `GoCollector` and `ProcessCollector` instead of disabling a route). This aligns with `node_exporter`, `mysqld_exporter`, and the in-house `slurm_exporter`. If a Prometheus job was scraping `/internal/metrics`, repoint it to `/metrics` and use `metric_relabel_configs` if you need to filter `go_*` / `process_*` / `promhttp_*`.
* `go_build_info` is now always exposed (via `prometheus.NewBuildInfoCollector`), even when `--web.disable-exporter-metrics` is set. Surfaces the running version and revision without a separate scrape job.
* New endpoint `/healthz` (returns `200 ok`) for Kubernetes / systemd liveness probes. It does *not* check fabric reachability — only that the HTTP server is up.

### Changed

* Go module bumped: `1.22` → `1.26`.
* Logging migrated from `github.com/go-kit/log` to the stdlib `log/slog`. `--log.level` and `--log.format` are now provided by `prometheus/common/promslog`. Output format changes slightly: each line is a single key-value record, with `level=` instead of `level=…`; downstream log parsers may need adjusting.
* Dependency bumps to latest stable:
  - `prometheus/client_golang` 1.19.1 → 1.23.2
  - `prometheus/common` 0.53.0 → 0.67.5 (was blocking Dependabot — `promlog` removed in 0.67, hence the `slog` migration)
  - `prometheus/exporter-toolkit` 0.11.0 → 0.16.0
  - `gofrs/flock` 0.8.1 → 0.13.0
  - `golang.org/x/{crypto,net,oauth2,sync,sys,text}` to current
* `make smoke` updated to probe `/healthz` and the new layout.

## 0.13.0 / 2026-04-28

Critical bug fixes, endpoint split, errcheck enabled.

### ⚠️ Breaking

* The IbswinfoCollector's three "collect" metrics have been renamed from `infiniband_switch_collect_*` to `infiniband_ibswinfo_collect_*`. Before this change, both collectors tried to register metrics with the same fully-qualified name but different label sets, which `client_golang` rejects at `MustRegister`. The label set on the renamed metrics is unchanged (`guid`, `collector`, `switch`). Update any alerting rules / dashboards that filtered by `infiniband_switch_collect_*{collector="ibswinfo"}`.
* `/metrics` no longer exposes `go_*`, `process_*`, `promhttp_*` self-metrics. Those moved to a separate endpoint `/internal/metrics` (still gated by `--web.disable-exporter-metrics`). If a Prometheus scrape job needs both, point it at `/internal/metrics` as a second target — typically with a longer scrape interval.

### Fixed

* Data race in `SwitchCollector.collect` and `HCACollector.collect`: the `errors` and `timeouts` counters were mutated from concurrent goroutines (capped by `--perfquery.max-concurrent`) without synchronization. Both are now `atomic.Uint64`. `go test -race` is now clean.
* Goroutine-scoped context leak in the rcv-err loop: each iteration's `defer cancelRcvErr()` accumulated until the goroutine returned. Wrapped in a per-iteration closure so the context cancels at the end of each port query.
* `hca.go`: arguments to `h.Uplink` were already reordered in v0.11.0; this lot adds a regression test (`TestCollectorsCoexist`) that exercises every collector against the same registry, which would have caught the original label-set conflict.
* `ibnetdiscover` and `perfquery` now capture and surface stderr in their wrapped errors. Previously, IB fabric failures and `mad_rpc` timeouts were silently dropped on the floor.

### Quality

* `errcheck` re-enabled in `.golangci.yml` after a one-pass audit. Functions that documented "always returns nil" (`bytes.Buffer.Write*`, `http.ResponseWriter.Write`, `go-kit/log.Logger.Log`, `context.CancelFunc`) are excluded with rationale.
* `queryExporter` test helper retries briefly to absorb the race between `go run(...)` startup and the first HTTP probe.

## 0.12.0 / 2026-04-28

GitHub Actions CI/CD, GoReleaser, lint baseline.

* `.github/workflows/test.yml` runs gofmt, `go vet`, `go test -race`, and `go build` inside the `golang:1.26.2` container, plus a separate `golangci-lint` job using the `golangci/golangci-lint:latest` container image.
* `.github/workflows/release.yml` triggers on tags matching `v*.*.*` and produces multi-arch (amd64/arm64/ppc64le/s390x) Linux tarballs with SHA-256 checksums via GoReleaser. Releases publish automatically on tag push.
* `.goreleaser.yaml` configures the build/archive/release pipeline. Conventional-commit prefixes (`feat:`, `fix:`, `perf:`, `refactor:`) are grouped in the auto-generated changelog.
* `.golangci.yml` (v2 schema) enables a deliberately conservative baseline (`govet`, `ineffassign`, `staticcheck`, `unconvert`, `unused`, `misspell`). `errcheck` is deferred to the next release where ignored returns will be audited; stylistic checks (`ST1000`/`ST1003`/`ST1005`/`QF1003`) and the wider quality set (`gocritic`, `prealloc`, `unparam`, `gosec`, `revive`) are tracked for follow-up releases.
* `.github/dependabot.yml` opens weekly grouped PRs for `gomod` (Prometheus stack and `golang.org/x` clustered) and `github-actions`.
* New `Makefile` targets: `lint`, `release-snapshot`, `release-check`, `ci-test`, `ci-lint`. The `ci-*` targets run the GitHub workflow locally via `act`, falling back to the `nektosact/act` container if `act` is not on `$PATH`.
* Auto-fixes applied by goimports / `golangci-lint --fix`:
  - `math.Pow(1000, 3)` → `1000*1000*1000` in `parseRate`
  - `strings.Replace(..., -1)` → `strings.ReplaceAll(...)` in `perfqueryParse`
  - import grouping with `github.com/SckyzO/...` as local prefix

## 0.11.0 / 2026-04-28

Independent fork — repository now lives at `github.com/SckyzO/infiniband_exporter`.

* Module path renamed `github.com/treydock/infiniband_exporter` → `github.com/SckyzO/infiniband_exporter`.
* Removed upstream Prometheus build scaffolding: CircleCI config, `Makefile.common`, `.promu.yml`, RPM spec, Helm chart, and the upstream `Dockerfile`.
* New container-only `Makefile`: every target (`build`, `test`, `vet`, `lint`, `release-snapshot`) runs inside `golang:1.26.2-alpine` / `golangci/golangci-lint` / `goreleaser` images. The Go toolchain is never invoked on the host.
* GitHub Actions CI/CD and a refreshed `.golangci.yml` arrive in 0.12.0.

## 0.10.0 / 2025-01-12

* Support TLS and basic auth (#25)
* Address issues where ibswinfo does not work (#32)
* Add Helm chart (#28)
* [BREAKING] Associate switch rate with port (#31)
* Ensure proper value for data rates from perfquery (#34)

## 0.10.0-rc.1 / 2024-09-06

* Support TLS and basic auth (#25)
* Address issues where ibswinfo does not work (#32)

## 0.10.0-rc.0 / 2024-06-04

* Support TLS and basic auth (#25)

## 0.9.0 / 2024-05-13

* Update to Go 1.22 and update dependencies (#23)
* Add metrics for per-device collection duration, error and timeout indicators (#22)

## 0.8.0 / 2024-02-27

* Ensure the full HCA name is included in "hca" and "uplink" labels (#21)

## 0.7.0 / 2023-12-21

* parseNames support for unconnected non-SDR lines (#18)
* Add infiniband_switch_uptime_seconds from ibswinfo (#19)

## 0.6.0 / 2023-12-03

* feat:device add raw rate & FDR effective lane rate accurate to 13.64 (#16)

## 0.5.2 / 2023-05-22

* Do not generate ibswinfo metrics for things that do not return values (#15)

## 0.5.1 / 2023-05-21

* Fix ibswinfo parsing when a PSU loses power on a switch (#14)

## 0.5.0 / 2023-05-06

* Update to Go 1.20 and update Go module dependencies (#13)

## 0.4.2 / 2022-12-07

* Rename infiniband_switch_fan_status to infiniband_switch_fan_status_info (#11)
* Include switch name with infiniband_switch_hardware_info (#11)

## 0.4.1 / 2022-12-07

* Ensure ibswinfo respects --sudo flag (#10)

## 0.4.0 / 2022-12-07

* Collect information from unmanaged switches using ibswinfo (BETA) (#9)

## 0.3.1 / 2022-08-24

* Handle switches with split mode enabled (#8)

## 0.3.0 / 2022-03-23

* Update to Go 1.17 and update Go module dependencies

## 0.2.0 / 2021-07-03

* Add `infiniband_exporter_last_execution` metric when exporter is run with `--exporter.runonce`

## 0.1.0 / 2021-07-03

* Add `--no-collector.hca.base-metrics` flag to disable collecting base HCA metrics
* Add `--no-collector.switch.base-metrics` flag to disable collecting base switch metrics
* When run with `--exporter.runonce`, the `collector` label will now have `-runonce` suffix to avoid conflicts with possible Prometheus scrape metrics

## 0.0.1 / 2021-04-27

### Changes

* Initial Release

