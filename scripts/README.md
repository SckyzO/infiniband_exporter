# Scripts

Utility scripts that live outside the build pipeline. They do not run as
part of CI; they are operator tools.

## `capture-fixtures.sh`

Captures raw `ibnetdiscover -p` and `perfquery -G <GUID> 1 -x` output from
a host with InfiniBand diagnostic tools and an active fabric link.

```bash
# Local
./scripts/capture-fixtures.sh /tmp/ib-raw

# Remote (the script is shipped over stdin so the host needs no checkout)
ssh fabric-host bash -s /tmp/ib-raw < scripts/capture-fixtures.sh
scp -r fabric-host:/tmp/ib-raw .
```

Output is **NOT yet anonymized** — it contains real GUIDs, switch and HCA
names, and possibly MACs.

## `anonymize.sh`

Reads a captured `ibnetdiscover` or `perfquery` output and writes an
anonymized copy. Replacements are stable inside a single invocation: the
same source GUID maps to the same destination GUID, so the
switch ↔ HCA relations encoded in the original file are preserved.

```bash
./scripts/anonymize.sh /tmp/ib-raw/ibnetdiscover.out collectors/testdata/integration/ibnetdiscover.out
./scripts/anonymize.sh /tmp/ib-raw/perfquery-0xabc.out collectors/testdata/integration/perfquery-sw01.out
```

The script targets:

* GUIDs `0x[0-9a-f]{16}` → `0x000000000000NNNN`
* Switch / HCA names quoted in `ibnetdiscover` output → `sw-NN` / `hca-NN`
* MAC addresses → `02:00:00:00:00:NN` (documentation range)

It does **not** touch LIDs (small integer, low signal). Inspect the diff
manually before committing — anonymization is best-effort.

## Workflow for adding integration fixtures

1. Run `capture-fixtures.sh` on a host with fabric access.
2. Copy the output back to your workstation.
3. Run `anonymize.sh` on each file.
4. Inspect the result — `grep` for any leftover hostname or GUID prefix
   you recognize; the script may miss formats it was not written for.
5. Commit under `collectors/testdata/integration/`.

## Reporting a bug

When a fabric-specific bug is suspected (e.g. "switch model X drops counter
Y", "the parser breaks on this `ibnetdiscover` line"), the same two scripts
are the right tool to attach a reproducer without leaking topology.

```bash
mkdir -p /tmp/ib-issue
./scripts/capture-fixtures.sh /tmp/ib-issue                    # raw, sensitive
./scripts/anonymize.sh /tmp/ib-issue/ibnetdiscover.out  ibnetdiscover.out
for f in /tmp/ib-issue/perfquery-*.out; do
    ./scripts/anonymize.sh "$f" "$(basename "$f")"
done
# Eyeball the anonymized files before attaching them.
grep -E '0x[0-9a-f]{16}|HCA-[0-9]|mlx[0-9]|hostname' ibnetdiscover.out perfquery-*.out
```

Attach the **anonymized** files to the GitHub issue. Do not attach the raw
captures — they contain real GUIDs and switch names. Maintainers can then
drop the files into `collectors/testdata/integration/` and write a
regression test against them.
