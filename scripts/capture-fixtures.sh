#!/usr/bin/env bash
# capture-fixtures.sh — capture raw `ibnetdiscover` and `perfquery` output
# from a live host so the test suite can replay them.
#
# Run on (or SSHed into) a host that has the InfiniBand diagnostic tools
# and an active fabric link. The captured files are written under a
# user-supplied directory; pipe them through ./scripts/anonymize.sh
# before committing them under collectors/testdata/integration/.
#
# Usage:
#   ./scripts/capture-fixtures.sh <output-dir>
#
# Or via SSH:
#   ssh <host> bash -s <output-dir> < scripts/capture-fixtures.sh
#
# Pick the GUIDs you actually want represented in the fixtures by editing
# the GUIDS array below. Defaults to "the first switch and the first HCA"
# from `ibnetdiscover -p`, which is enough for a small but realistic
# integration test.

set -euo pipefail

if [[ $# -lt 1 ]]; then
	echo "usage: $0 <output-dir>" >&2
	exit 64
fi

outdir=$1
mkdir -p "$outdir"

# 1. Topology dump — drives parsing of the device list and uplinks.
ibnetdiscover -p >"$outdir/ibnetdiscover.out"
echo "wrote $outdir/ibnetdiscover.out ($(wc -l <"$outdir/ibnetdiscover.out") lines)"

# 2. Pick the first SW and the first CA GUID from the topology to
#    illustrate both code paths in tests.
mapfile -t SW_GUIDS < <(awk '$1 == "SW" { print $4; }' "$outdir/ibnetdiscover.out" | sort -u)
mapfile -t CA_GUIDS < <(awk '$1 == "CA" { print $4; }' "$outdir/ibnetdiscover.out" | sort -u)

if [[ ${#SW_GUIDS[@]} -eq 0 || ${#CA_GUIDS[@]} -eq 0 ]]; then
	echo "could not extract switch/HCA GUIDs — is the fabric reachable?" >&2
	exit 1
fi

GUIDS=("${SW_GUIDS[0]}" "${CA_GUIDS[0]}")

for guid in "${GUIDS[@]}"; do
	# port "1" is universal across IB devices; perfquery accepts any.
	# -x: extended counters (the same set the exporter reads).
	out="$outdir/perfquery-${guid}.out"
	perfquery -G "$guid" 1 -x >"$out" || {
		echo "perfquery -G $guid 1 -x failed; continuing" >&2
	}
	echo "wrote $out"
done

echo
echo "next: run ./scripts/anonymize.sh on each captured file before"
echo "committing it under collectors/testdata/integration/."
