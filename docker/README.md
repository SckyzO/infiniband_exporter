Prometheus exporter for [InfiniBand](https://en.wikipedia.org/wiki/InfiniBand)
fabrics, shipped as a self-contained container image with `infiniband-diags`
and the `ibswinfo` helper bundled in.

[![Release](https://img.shields.io/github/v/release/SckyzO/infiniband_exporter?label=release)](https://github.com/SckyzO/infiniband_exporter/releases)
[![Build](https://img.shields.io/github/actions/workflow/status/SckyzO/infiniband_exporter/release.yml?label=build)](https://github.com/SckyzO/infiniband_exporter/actions/workflows/release.yml)
[![Pulls](https://img.shields.io/docker/pulls/sckyzo/infiniband-exporter)](https://hub.docker.com/r/sckyzo/infiniband-exporter)
[![Image size](https://img.shields.io/docker/image-size/sckyzo/infiniband-exporter/latest?label=size)](https://hub.docker.com/r/sckyzo/infiniband-exporter/tags)
[![License](https://img.shields.io/github/license/SckyzO/infiniband_exporter)](https://github.com/SckyzO/infiniband_exporter/blob/main/LICENSE)

## Tags

A single image, published as a **multi-arch manifest** (linux/amd64 +
linux/arm64) to **two registries**:

- `docker.io/sckyzo/infiniband-exporter`
- `ghcr.io/sckyzo/infiniband_exporter` (mirror)

| Tag pattern | Meaning |
|---|---|
| `:X.Y.Z` | Exact release, moving alias (re-pushed weekly with a fresh base) |
| `:X.Y`, `:X` | Latest patch / minor of that line |
| `:latest` | Latest stable release |
| `:X.Y.Z-YYYYMMDD` | Immutable dated rebuild, for bit-for-bit GitOps determinism |

Pre-release tags (`vX.Y.Z-rc1` etc.) push only the pinned version and never
overwrite the floating tags.

> infiniband-diags is only packaged for amd64 and arm64 on Debian, so the
> images cover those two arches. The release tarballs additionally cover
> ppc64le and s390x.

## Quick start

The exporter must run on a host with an **active InfiniBand link**: it shells
out to `ibnetdiscover` / `perfquery` (and optionally `ibswinfo`), which talk
to the local IB stack through `/dev/infiniband/umad*`.

```bash
docker run -d --name infiniband_exporter \
  --device /dev/infiniband \
  -p 9315:9315 \
  sckyzo/infiniband-exporter:latest \
  --collector.ibswinfo

curl -s http://localhost:9315/metrics | head
```

If that returns metrics, you're done. If it doesn't, read on.

## What this image does

The exporter discovers the fabric with `ibnetdiscover`, queries per-port
counters with `perfquery`, and (optionally) collects switch hardware /
PSU / fan / temperature data with `ibswinfo`. It exposes everything as
Prometheus metrics on `/metrics`.

Both `infiniband-diags` and the
[`ibswinfo`](https://github.com/SckyzO/ibswinfo) helper script (at
`/usr/local/bin/ibswinfo.sh`) are baked into the image, so
`--collector.ibswinfo` works out of the box — no host dependency to install.

### Collectors at a glance

| Collector | Default | Purpose |
|---|---|---|
| `switch` | **on** | Per-port `perfquery` counters for fabric switches |
| `switch.port-state` | **on** | `infiniband_switch_port_state` (1=up, 0=down) — powers the port-down alert |
| `hca` | **on** | Same counters from each HCA port the host can reach |
| `ibswinfo` | off | Switch hardware / PSU / fan / temperature (opt-in) |

`--collector.ibswinfo` is the only collector you usually have to turn on.
Run `--help` for the full flag list.

## Device access & permissions

The IB tools need `/dev/infiniband/umad*`. Two paths:

1. **Pass the device** (recommended) and make the umad nodes world-RW on the
   host via a udev rule — production IB hosts usually do this anyway:

   ```bash
   docker run ... --device /dev/infiniband ...
   ```
   ```
   # /etc/udev/rules.d/99-ib.rules
   KERNEL=="umad*", NAME="infiniband/%k" MODE="0666"
   ```

2. **Fallback:** `--privileged` if local permissions block device access.
   Prefer the targeted `--device /dev/infiniband` over `--privileged`.

The image runs as the unprivileged `nobody` user. If your umad nodes are
root-only and you can't relax them, run the exporter with `--sudo` plus a
sudoers entry whitelisting the IB binaries (see the repo docs).

## Compose

```yaml
services:
  infiniband_exporter:
    image: sckyzo/infiniband-exporter:latest
    container_name: infiniband_exporter
    restart: unless-stopped
    command: ["--collector.ibswinfo"]
    ports:
      - "9315:9315"
    devices:
      - /dev/infiniband:/dev/infiniband
    security_opt:
      - no-new-privileges:true
    read_only: true
    tmpfs: [/tmp]
```

## Prometheus scrape config

```yaml
scrape_configs:
  - job_name: infiniband
    static_configs:
      - targets: ["infiniband_exporter:9315"]
    scrape_interval: 30s
    scrape_timeout: 25s   # IB discovery + perfquery can be slow on large fabrics
```

For large fabrics where a scrape can't finish inside Prometheus's timeout,
use `--exporter.runonce` + node_exporter's textfile collector instead. See
[docs/operations.md](https://github.com/SckyzO/infiniband_exporter/blob/main/docs/operations.md).

## Supply chain

Every published artefact ships with two verifiable signals.

### Image signatures (cosign / Sigstore keyless)

Every manifest is signed by the GitHub Actions workflow that built it,
attested by the runner's OIDC token. No keys on either side.

```bash
cosign verify sckyzo/infiniband-exporter:latest \
  --certificate-identity-regexp 'https://github.com/SckyzO/infiniband_exporter/.github/workflows/release.yml@.*' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com
```

Weekly-refreshed images are signed the same way by `docker-refresh.yml`
(swap the workflow name in the identity regexp). The release checksums file
is signed too (`*.pem` + `*.sig` alongside the checksums on the GitHub
release).

### Software Bill of Materials

Each release archive ships with a **CycloneDX SBOM** (`*.sbom.json`) listing
every Go module compiled into the binary, with versions and PURLs.

```bash
gh release download v2.0.1 -p '*sbom.json'
jq '.components[] | {name, version, purl}' infiniband_exporter-2.0.1.linux-amd64.tar.gz.sbom.json
```

## Image freshness

| Event | What happens |
|---|---|
| Release tag pushed (`vX.Y.Z`) | Full GoReleaser run: build, push to both registries, sign, generate SBOM, attach to the GitHub release. |
| Weekly cron (Monday 04:00 UTC) | The last 2 stable lines are rebuilt against the up-to-date debian base, with `infiniband-diags` and `ibswinfo.sh` re-fetched. The moving `:vX.Y.Z` tag is re-pushed with a fresh digest, plus an immutable dated tag `:vX.Y.Z-YYYYMMDD`. |

Pin `:vX.Y.Z-YYYYMMDD` for reproducible deployments. Use the unsuffixed
`:vX.Y.Z` (or `:latest`) to always get the freshest base-image patches for
that version.

## Verifying what you're running

OCI labels carry the build metadata:

```bash
docker inspect sckyzo/infiniband-exporter:latest \
  --format '{{json .Config.Labels}}' | jq
```

Look for `org.opencontainers.image.created` (build timestamp),
`org.opencontainers.image.revision` (git commit) and
`org.opencontainers.image.version`. At runtime, the
`infiniband_exporter_build_info` metric carries the same version stamps.

## Links

- **Source & full documentation**: [github.com/SckyzO/infiniband_exporter](https://github.com/SckyzO/infiniband_exporter)
- **Report a bug / request a feature**: [issue tracker](https://github.com/SckyzO/infiniband_exporter/issues)
- **Release notes**: [CHANGELOG.md](https://github.com/SckyzO/infiniband_exporter/blob/main/CHANGELOG.md)
- **GHCR mirror**: [ghcr.io/sckyzo/infiniband_exporter](https://github.com/users/SckyzO/packages/container/package/infiniband_exporter)

## License

Apache-2.0 — see [LICENSE](https://github.com/SckyzO/infiniband_exporter/blob/main/LICENSE).
