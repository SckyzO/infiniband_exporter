# Contributing

## Dev environment

You only need `docker` (or `podman` aliased to `docker`). The Go
toolchain, golangci-lint and goreleaser all run inside containers
driven by the [`Makefile`](Makefile). Nothing touches the host.

```bash
# happy path — all of fmt-check + vet + test + lint + build
make

# faster individual targets
make test           # go test -short ./...
make test-race      # go test -race + coverage profile
make lint           # golangci-lint run
make smoke          # build + bind /metrics + /healthz briefly
```

The `Makefile` exposes one target per stage of CI; pick the smallest
one that exercises what you changed.

## Validating GitHub Actions locally

`make ci-test` and `make ci-lint` run the GitHub Actions workflow via
[`act`](https://nektosact.com). If `act` is not on `$PATH` they fall
back to the `nektosact/act` container — you still need Docker but
nothing else.

## Code conventions

* Format: `gofmt -s` + `goimports` (handled by `make fmt` /
  `golangci-lint run --fix`).
* Lint: the baseline (`govet`, `ineffassign`, `staticcheck`, `unused`,
  `unconvert`, `misspell`, `errcheck`) must pass clean. Stricter
  checks (`gocritic`, `prealloc`, `unparam`, `gosec`, `revive`) are
  intentionally deferred.
* Comments: write the *why*, not the *what*. Identifier names
  document the *what*. Reference upstream PRs and the IB-PM spec
  where it helps a reader who lands on the file cold.

## Tests

Test fixtures live under `collectors/testdata/` (Go's magic name —
the toolchain skips them in build/vet). Keep new fixtures small and
self-explanatory. For real-fabric output, **anonymize** with the
[`scripts/anonymize.sh`](scripts/anonymize.sh) helper before
committing — see [`scripts/README.md`](scripts/README.md).

## Releasing

Tags `v*.*.*` trigger
[`.github/workflows/release.yml`](.github/workflows/release.yml) —
GoReleaser produces multi-arch tarballs (linux amd64 / arm64 /
ppc64le / s390x), SHA-256 checksums, and a GitHub Release with an
auto-generated changelog. Cut a release with:

```bash
git tag -a vX.Y.Z -m "release notes…"
git push origin main
git push origin vX.Y.Z
```

Conventional commit prefixes (`feat:`, `fix:`, `perf:`,
`refactor:`) are recognized by the changelog grouping; everything
else lands in *Other*.

Bump strategy across the lots leading to v1.0.0 has been one minor
version per "lot" of work — see `CHANGELOG.md` for the trail.
