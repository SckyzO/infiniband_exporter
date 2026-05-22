# Distribution image for infiniband_exporter.
# GoReleaser builds the binary on the host and copies it in via the
# `dockers:` step in .goreleaser.yaml — there is no go toolchain stage
# here.

FROM debian:bookworm-slim

# Pinned ibswinfo helper version. Bump when upstream cuts a new release.
ARG IBSWINFO_VERSION=v0.9.0

# Build metadata. GoReleaser sets these via build_args at release time; the
# weekly docker-refresh workflow passes them via --build-arg. They feed the
# OCI version/revision/created labels below so the published manifest carries
# accurate provenance regardless of which path built it.
ARG VERSION=dev
ARG COMMIT=
ARG BUILD_DATE=

LABEL org.opencontainers.image.title="infiniband_exporter" \
      org.opencontainers.image.description="Prometheus exporter for InfiniBand fabrics" \
      org.opencontainers.image.licenses="Apache-2.0" \
      org.opencontainers.image.source="https://github.com/SckyzO/infiniband_exporter" \
      org.opencontainers.image.documentation="https://github.com/SckyzO/infiniband_exporter/blob/main/docker/README.md" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${COMMIT}" \
      org.opencontainers.image.created="${BUILD_DATE}"

# infiniband-diags ships ibnetdiscover, perfquery, ibstat, smpquery —
# the tools the exporter (and ibswinfo) shell out to. bash is
# required by ibswinfo.sh; curl is used to fetch ibswinfo.sh during
# build. ca-certificates so HTTPS scrapes against --web.config.file
# targets work.
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        infiniband-diags \
        ca-certificates \
        bash \
        curl && \
    curl -fsSL "https://raw.githubusercontent.com/SckyzO/ibswinfo/${IBSWINFO_VERSION}/ibswinfo.sh" \
        -o /usr/local/bin/ibswinfo.sh && \
    chmod +x /usr/local/bin/ibswinfo.sh && \
    apt-get purge -y curl && \
    apt-get autoremove -y && \
    rm -rf /var/lib/apt/lists/*

COPY infiniband_exporter /usr/local/bin/infiniband_exporter
COPY LICENSE /licenses/LICENSE

EXPOSE 9315
USER nobody

ENTRYPOINT ["/usr/local/bin/infiniband_exporter"]
