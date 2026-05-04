# Distribution image for infiniband_exporter.
# GoReleaser builds the binary on the host and copies it in via the
# `dockers:` step in .goreleaser.yaml — there is no go toolchain stage
# here.

FROM debian:bookworm-slim

LABEL org.opencontainers.image.source="https://github.com/SckyzO/infiniband_exporter"
LABEL org.opencontainers.image.description="Prometheus exporter for InfiniBand fabrics"
LABEL org.opencontainers.image.licenses="Apache-2.0"

# infiniband-diags ships ibnetdiscover, perfquery, ibstat — the tools the
# exporter shells out to. ca-certificates so HTTPS scrapes against
# --web.config.file targets work. No --no-install-recommends suggests
# pulling in suggested helpers; we explicitly avoid them.
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        infiniband-diags \
        ca-certificates && \
    rm -rf /var/lib/apt/lists/*

COPY infiniband_exporter /usr/local/bin/infiniband_exporter
COPY LICENSE /licenses/LICENSE

EXPOSE 9315
USER nobody

ENTRYPOINT ["/usr/local/bin/infiniband_exporter"]
