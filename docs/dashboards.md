# Grafana dashboards

The exporter ships with six dashboards in
[`examples/grafana/`](../examples/grafana/). All of them target
**Grafana 10+** and use the `${DS_PROMETHEUS}` placeholder so they
import cleanly through the standard *Dashboards → Import → Upload JSON*
flow.

| File | UID | When to use |
| --- | --- | --- |
| `infiniband-fabric-overview-small.json` | `ib-fabric-overview-small` | 2–40 switches. Per-switch lines remain readable. |
| `infiniband-fabric-overview-large.json` | `ib-fabric-overview-large` | 40+ switches. Heatmap for temperatures, `topk(20)` for ranked panels, min/avg/max stats. |
| `infiniband-switch-detail.json` | `ib-switch-detail` | Drill-down per switch (variable `$switch`). Same dashboard regardless of fabric size — already scoped. |
| `infiniband-hca-detail.json` | `ib-hca-detail` | Drill-down per HCA (variable `$hca`). |
| `infiniband-exporter-internals.json` | `ib-exporter-internals` | Health of the exporter process itself — collector latency, scrape errors, Go runtime. |
| `infiniband_and_node_exporter.json` | `ib-and-node` | **Management node combo.** Combines node_exporter (CPU, RAM, load, filesystem, IB driver counters) with the exporter's own scrape metrics. Useful when both run on the same management host (typical setup). |

## Importing

In Grafana: **Dashboards → New → Import → Upload JSON file**.

Grafana picks up `__inputs` and prompts you for the Prometheus data
source — pick the one scraping `infiniband_exporter`.

Each dashboard has a stable `uid`, so cross-dashboard drill-down links
work out of the box once they are imported (the *Fabric Overview*
panels link to *Switch detail* and *HCA detail*).

## Provisioning

For provisioning via Grafana's file-based datasource layer, drop the
JSON files under your dashboards provider path; no edits needed.

## Publishing on grafana.com

Each dashboard is shipped in the format expected by
[Grafana Labs' dashboard library](https://grafana.com/grafana/dashboards/):

* `__inputs` describes the placeholder data sources required at import.
* `__requires` declares the minimum Grafana version (10.0.0) and the
  panel plugins used.
* `id` is `null` so Grafana assigns one at upload.

Upload steps (per dashboard) on grafana.com:

1. Sign in with your Grafana Cloud / grafana.com account.
2. Go to **Dashboards → Upload dashboard**.
3. Paste or upload the JSON.
4. Fill the metadata — title, description, tags `infiniband`,
   `prometheus`, `hpc`, screenshots.

## Dashboards rely on the exporter, not on the recording rules

To keep grafana.com import clean, all dashboards use raw Prometheus
expressions (`rate(infiniband_..._total[5m])`). They work on a vanilla
Prometheus that does not yet have the recording rules from
`examples/prometheus/rules/` loaded.

If you do install the recording rules, you can swap the dashboards'
expressions for the cheaper recording-rule names — see
[alerts.md](alerts.md) for the list.

## Caveats

* The **ibswinfo panels** (PSU watts, temperatures, fan RPM, hardware
  inventory) need `--collector.ibswinfo` on the exporter side. If
  ibswinfo is disabled, those panels show "No data" — that is correct
  behaviour, not a dashboard bug.
* The **port_state table** in *Switch detail* needs
  `--collector.switch.port-state`.
* `infiniband_and_node_exporter` assumes node_exporter's `infiniband`
  collector is enabled (`--collector.infiniband` on node_exporter ≥ 1.5).
  Without it, the *driver-level* row stays empty.
