# Grafana dashboards

The exporter ships with nine dashboards in
[`examples/grafana/`](../examples/grafana/) — seven operational views
plus two fabric-overview variants (small and large fabrics). All of
them target
**Grafana 10+** and use the `${DS_PROMETHEUS}` placeholder so they
import cleanly through the standard *Dashboards → Import → Upload JSON*
flow.

| File | UID | When to use |
| --- | --- | --- |
| `00-infiniband-fabric-overview-small.json` | `ib-fabric-overview-small` | 2–40 switches. Per-switch lines remain readable. |
| `00-infiniband-fabric-overview-large.json` | `ib-fabric-overview-large` | 40+ switches. Heatmap for temperatures, `topk(20)` for ranked panels, min/avg/max stats. |
| `01-infiniband-switch-fleet.json` | `ib-switch-fleet` | Tabular view of every switch with throughput, error rate and ibswinfo health columns. Use to scan the whole fabric in one page. |
| `02-infiniband-switch-detail.json` | `ib-switch-detail` | Drill-down per switch (variable `$switch`). Same dashboard regardless of fabric size — already scoped. |
| `03-infiniband-hca-detail.json` | `ib-hca-detail` | Drill-down per HCA (variable `$hca`). |
| `04-infiniband-health.json` | `ib-health` | "What's wrong" dashboard. Down switches/HCAs/ports, error trends, link-downed events, top noisy ports. Open this when an alert fires. |
| `05-infiniband-environmental.json` | `ib-environmental` | Temperature, fan RPM, PSU watts, hardware inventory. Requires `--collector.ibswinfo`. |
| `06-infiniband-and-node-exporter.json` | `ib-and-node` | **Management node combo.** Combines node_exporter (CPU, RAM, load, filesystem, IB driver counters) with the exporter's own scrape metrics. Useful when both run on the same management host (typical setup). |
| `07-infiniband-exporter-internals.json` | `ib-exporter-internals` | Health of the exporter process itself — collector latency, scrape errors, Go runtime. |

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
expressions (`rate(infiniband_..._total[$__rate_interval])`). They
work on a vanilla Prometheus that does not yet have the recording
rules from `examples/prometheus/rules/` loaded. The recording rules
exist primarily for the alert pack — alertmanager evaluates them
continuously regardless of whether dashboards are open, so the
compute cost is amortized there.

For very large fabrics (>200 switches) where panel refresh times
become noticeable, swap the heaviest panel queries (fabric-overview
top-N error/xmit_wait) for their `infiniband:*:rate5m` recording
rule equivalents — see [alerts.md](alerts.md) for the list.

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
