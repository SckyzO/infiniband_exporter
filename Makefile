# Container-only build/test/lint surface.
# Nothing in this Makefile must execute the Go toolchain on the host.

GO_VERSION   ?= 1.26.2
# Debian-based image: ships gcc + cgo so `go test -race` works out of the box.
GO_IMAGE     ?= golang:$(GO_VERSION)
LINT_IMAGE   ?= golangci/golangci-lint:latest
RELEASE_IMAGE ?= goreleaser/goreleaser:latest

DOCKER       ?= docker
PKG          ?= ./...
BIN          ?= infiniband_exporter

# Mount the working tree, persist module + build caches under .build/cache so
# repeat runs don't pay the dependency download tax.
RUN_GO = $(DOCKER) run --rm \
	-v "$(CURDIR)":/src \
	-v "$(CURDIR)/.build/cache/go-mod":/go/pkg/mod \
	-v "$(CURDIR)/.build/cache/go-build":/root/.cache/go-build \
	-w /src \
	$(GO_IMAGE)

RUN_LINT = $(DOCKER) run --rm \
	-v "$(CURDIR)":/src \
	-v "$(CURDIR)/.build/cache/golangci":/root/.cache/golangci-lint \
	-w /src \
	$(LINT_IMAGE)

.PHONY: all
all: fmt-check vet test lint build

.PHONY: build
build:
	# -buildvcs=false: container has the source tree but no `git` to stamp VCS info.
	# GoReleaser handles version stamping at release time.
	$(RUN_GO) go build -trimpath -buildvcs=false -o $(BIN) .

.PHONY: test
test:
	$(RUN_GO) go test -short $(PKG)

.PHONY: test-race
test-race:
	$(RUN_GO) go test -race -coverprofile=coverage.txt -covermode=atomic $(PKG)

.PHONY: vet
vet:
	$(RUN_GO) go vet $(PKG)

.PHONY: fmt
fmt:
	$(RUN_GO) sh -c 'gofmt -s -w $$(find . -type f -name "*.go" -not -path "./.build/*")'

.PHONY: fmt-check
fmt-check:
	@out=$$($(RUN_GO) sh -c 'gofmt -l $$(find . -type f -name "*.go" -not -path "./.build/*")'); \
	if [ -n "$$out" ]; then \
		echo "gofmt needed on:"; echo "$$out"; exit 1; \
	fi

.PHONY: lint
lint:
	$(RUN_LINT) golangci-lint run $(PKG)

.PHONY: tidy
tidy:
	$(RUN_GO) go mod tidy

.PHONY: release-snapshot
release-snapshot:
	$(DOCKER) run --rm \
		-v "$(CURDIR)":/src \
		-w /src \
		$(RELEASE_IMAGE) release --snapshot --clean

.PHONY: release-check
release-check:
	$(DOCKER) run --rm \
		-v "$(CURDIR)":/src \
		-w /src \
		$(RELEASE_IMAGE) check

# Local validation of the GitHub Actions workflow via `act`.
# If `act` is on PATH it is used directly; otherwise we fall back to the
# `nektosact/act` container image. Either way, the heavy lifting happens
# inside containers — no Go toolchain runs on the host.
ACT_IMAGE   ?= nektosact/act:latest
ACT_LOCAL   := $(shell command -v act 2>/dev/null)

# Container fallback: mount the Docker socket so `act` can orchestrate its
# own job containers via the host daemon (Docker-out-of-Docker). Without
# this, `act` cannot launch the runner image.
ACT_CONTAINER = $(DOCKER) run --rm \
	-v /var/run/docker.sock:/var/run/docker.sock \
	-v "$(CURDIR)":/src \
	-w /src \
	$(ACT_IMAGE)

ifdef ACT_LOCAL
ACT := act
else
ACT := $(ACT_CONTAINER)
endif

.PHONY: ci-test
ci-test:
	@if [ -z "$(ACT_LOCAL)" ]; then echo "act not found locally, using $(ACT_IMAGE)"; fi
	$(ACT) -W .github/workflows/test.yml --container-architecture linux/amd64 -j test

.PHONY: ci-lint
ci-lint:
	@if [ -z "$(ACT_LOCAL)" ]; then echo "act not found locally, using $(ACT_IMAGE)"; fi
	$(ACT) -W .github/workflows/test.yml --container-architecture linux/amd64 -j lint

.PHONY: clean
clean:
	rm -rf $(BIN) coverage.txt dist/ .build/
