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

