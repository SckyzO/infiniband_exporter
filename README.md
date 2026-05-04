# InfiniBand Prometheus exporter

Collects per-port counters from InfiniBand switches and HCAs through the
standard `infiniband-diags` tools (`ibnetdiscover`, `perfquery`) and the
optional `ibswinfo` helper for unmanaged switches. Exposes them on a
`/metrics` endpoint suitable for Prometheus.

> **Status: stable.** Independent fork of
> [`treydock/infiniband_exporter`](https://github.com/treydock/infiniband_exporter).
> See [`CHANGELOG.md`](CHANGELOG.md) for the divergence history and the
> stability commitment.

## Quick start

> **Recommended command line for a typical fabric.** `--collector.hca`
> and `--collector.switch.port-state` are off by default for upstream
> compatibility but most operators want them; the rest are tuned for a
> ≤50-switch HDR fabric. Adjust the `*.max-concurrent` flags upward on
> bigger sites.

```bash
infiniband_exporter \
    --collector.hca \
    --collector.switch.port-state \
    --collector.ibswinfo \
    --ibswinfo.path=/usr/local/bin/ibswinfo.sh \
    --perfquery.max-concurrent=4 \
    --ibswinfo.max-concurrent=4 \
    --web.listen-address=:9315
```

To deploy:

1. Install `infiniband-diags` on the host (provides `ibnetdiscover`
   and `perfquery`). Optional: install
   [`ibswinfo`](https://github.com/SckyzO/ibswinfo) if you have
   unmanaged Mellanox switches and want PSU/fan/temperature data.
2. Confirm at least one active IB link exists on that host.
3. Drop the `infiniband_exporter` binary in `/usr/local/bin/`.
4. Either install the systemd unit from
   `systemd/infiniband_exporter@.service`, or run the command above
   directly.
5. Scrape `http://<host>:9315/metrics`:
   ```yaml
   scrape_configs:
     - job_name: infiniband
       static_configs:
         - targets: ["<host>:9315"]
   ```
6. Drop the rules from
   [`examples/prometheus/rules/`](examples/prometheus/rules/) into
   Prometheus and import the dashboards from
   [`examples/grafana/`](examples/grafana/).

### Container

Multi-arch images on GitHub Container Registry:
`ghcr.io/sckyzo/infiniband_exporter`.

```bash
docker run --rm \
    --device /dev/infiniband \
    -p 9315:9315 \
    ghcr.io/sckyzo/infiniband_exporter:latest \
    --collector.hca \
    --collector.switch.port-state
```

The container bundles `infiniband-diags` **and** the
[`ibswinfo`](https://github.com/SckyzO/ibswinfo) helper script
(at `/usr/local/bin/ibswinfo.sh`), so `--collector.ibswinfo` works
out-of-the-box. Pass `--device /dev/infiniband` (or `--privileged`
if local permissions block device access) so the exporter can reach
the host's IB stack. Bump the bundled ibswinfo by overriding
`IBSWINFO_VERSION` at build time:

```bash
docker build --build-arg IBSWINFO_VERSION=v0.10.0 -t infiniband_exporter:custom .
```

## Endpoints

| Path | Purpose |
| --- | --- |
| `/metrics` | InfiniBand metrics + Go runtime / process / promhttp self-metrics. `go_build_info` is always present so dashboards can identify the running version. |
| `/healthz` | Returns `200 ok` if the HTTP server is up. Does **not** probe the fabric — pair with metric-based alerts (`infiniband_switch_up` / `infiniband_hca_up`) for that. |

Set `--web.disable-exporter-metrics` to skip registering the Go runtime
and process collectors. Filtering individual `go_*` / `process_*` /
`promhttp_*` series is the responsibility of Prometheus — use
`metric_relabel_configs: drop` if needed.

## Collectors

Enabled or disabled with `--collector.<name>` / `--no-collector.<name>`.

| Collector | Default | Purpose |
| --- | --- | --- |
| `switch` | enabled | Per-port `perfquery` counters for fabric switches |
| `hca` | disabled | Same counters, viewed from each HCA port the host can reach |
| `ibswinfo` | disabled | Hardware info, PSU/fan status, temperature for switches via the [ibswinfo](https://github.com/stanford-rc/ibswinfo) helper |
| `switch.base-metrics` | enabled | Toggles the headline `switch_*` series. Disable with `--no-collector.switch.base-metrics` to run rcv-error-details only |
| `switch.rcv-err-details` | disabled | Adds the slower `-E` perfquery counters (one query per port) |
| `switch.port-state` | disabled | Adds `infiniband_switch_port_state{port}` gauge (1 = up, 0 = down). See [docs/alerts.md](docs/alerts.md) for the alerting recipe. |
| `hca.base-metrics` | enabled | Mirror of `switch.base-metrics` for the HCA collector |
| `hca.rcv-err-details` | disabled | Mirror of `switch.rcv-err-details` for HCA |

## Configuration

Selected flags. Run `infiniband_exporter --help` for the full list.

| Flag | Default | Notes |
| --- | --- | --- |
| `--web.listen-address` | `:9315` | TLS / basic auth via `--web.config.file` (toolkit format) |
| `--sudo` | `false` | Wrap every `ibnetdiscover` / `perfquery` / `ibswinfo` invocation in `sudo`. Sample sudoers below. |
| `--ibnetdiscover.path` | `ibnetdiscover` | Override if not on `$PATH` |
| `--ibnetdiscover.timeout` | `20s` | |
| `--ibnetdiscover.cache-ttl` | `0s` | When >0, reuses the parsed topology between scrapes |
| `--perfquery.max-concurrent` | `1` | Critical for large fabrics. Bump to ~8 on multi-core hosts. |
| `--perfquery.timeout` | `5s` | |
| `--ibswinfo.max-concurrent` | `4` | Increased from 1 in v0.15.0 — see [docs/operations.md](docs/operations.md#sizing-perfquery-and-ibswinfo) |
| `--ibswinfo.static-cache-ttl` | `15m` | Caches PartNumber / SerialNumber / firmware so most scrapes use the lighter `ibswinfo -o vitals` mode. Set to `0` to disable. |
| `--exporter.runonce` | `false` | Single shot, write metrics to `--exporter.output` and exit. Pairs with node_exporter's textfile collector for fabrics where scrape time exceeds Prometheus's scrape timeout. |

### Permissions

The exporter shells out to `ibnetdiscover`, `perfquery`, and `ibswinfo`,
all of which need access to `/dev/infiniband/umad*`. Two options:

* Open the device node (production-grade systems usually do this anyway):
  ```
  $ cat /etc/udev/rules.d/99-ib.rules
  KERNEL=="umad*", NAME="infiniband/%k" MODE="0666"
  ```

* Or run the exporter with `--sudo` and a sudoers entry that whitelists
  exactly the binaries we need:
  ```
  Defaults:infiniband_exporter !syslog,!requiretty
  infiniband_exporter ALL=(ALL) NOPASSWD: /usr/sbin/ibnetdiscover, /usr/sbin/perfquery, /usr/bin/ibswinfo
  ```

### Large fabrics

For fabrics where a single scrape exceeds Prometheus's scrape timeout,
run the exporter with `--exporter.runonce` and have node_exporter pick
up the textfile output. Full guide: [docs/operations.md](docs/operations.md).

## Documentation

* [docs/operations.md](docs/operations.md) — sizing, sudoers, runonce mode, troubleshooting.
* [docs/metrics.md](docs/metrics.md) — full metric reference.
* [docs/alerts.md](docs/alerts.md) — annotated walkthrough of the example alert rules.
* [docs/dashboards.md](docs/dashboards.md) — installing the example Grafana dashboards.
* [scripts/README.md](scripts/README.md) — capturing & anonymizing fabric output (for issues / fixtures).

## Build from source

Every build, test, lint, and release operation runs inside a container
— the Go toolchain is never invoked on the host. You only need
`docker` (or `podman` aliased to `docker`).

| Target | Image |
| --- | --- |
| `make build` | `golang:1.26.2` |
| `make test`, `make test-race`, `make vet`, `make fmt[-check]` | `golang:1.26.2` |
| `make lint` | `golangci/golangci-lint:latest` |
| `make ci-test`, `make ci-lint` | `nektosact/act` if `act` is not on `$PATH` |
| `make smoke` | local binary, exercises `--help`, `--version`, `/healthz`, `/metrics` |
| `make release-snapshot`, `make release-check` | `goreleaser/goreleaser:latest` |

Module and build caches persist under `.build/cache/` for fast
repeat invocations. See [`CONTRIBUTING.md`](CONTRIBUTING.md) for the
expected dev workflow.

## Reporting an issue

For fabric-specific bugs, please attach **anonymized** captures using the
helpers in [`scripts/`](scripts/) — the issue template walks you through it.
This way the bug stays reproducible without leaking your topology.
