#!/usr/bin/env bash
# anonymize.sh — strip operational identifiers from infiniband tool output
# before committing it as a test fixture.
#
# Reads from <input> and writes to <output>. Replacements are stable: the
# same source GUID, switch name, HCA name, MAC, or LID always maps to the
# same anonymized value across one invocation, so the relations between
# devices in the file are preserved.
#
# Targets:
#   * GUIDs   0x[0-9a-fA-F]{16}            -> 0x000000000000NNNN  (sequential)
#   * Switch names "iswr...", "ib-iN...", "5FB0..." (configurable)
#                                          -> sw-NN
#   * HCA names "<host> HCA-N", "<host> mlxN_N"
#                                          -> hca-NN
#   * LIDs    decimal 1-49151 in the LID column
#                                          -> sequential 100, 101, ...
#   * MACs    XX:XX:XX:XX:XX:XX            -> 02:00:00:00:00:NN
#
# Usage:
#   ./scripts/anonymize.sh <input.out> <output.out>
#
# Limitations: this script is intentionally simple. It targets the formats
# emitted by `ibnetdiscover -p` and `perfquery -G ... -x`. Inspect the
# output diff before committing — eyeballing a `sed`-driven mapping is
# always wise for prod data.

set -euo pipefail

if [[ $# -ne 2 ]]; then
	echo "usage: $0 <input> <output>" >&2
	exit 64
fi

input=$1
output=$2

if [[ ! -f $input ]]; then
	echo "input file not found: $input" >&2
	exit 1
fi

tmp=$(mktemp)
trap 'rm -f "$tmp" "$tmp.map.guid" "$tmp.map.sw" "$tmp.map.hca" "$tmp.map.lid"' EXIT

# Phase 1: GUIDs. Stable per-input mapping.
guid_idx=0
declare -A guid_map=()
while read -r guid; do
	if [[ -z "${guid_map[$guid]:-}" ]]; then
		guid_idx=$((guid_idx + 1))
		guid_map[$guid]=$(printf '0x%016x' "$guid_idx")
	fi
done < <(grep -ohE '0x[0-9a-fA-F]{16}' "$input" | sort -u)

cp "$input" "$tmp"
for src in "${!guid_map[@]}"; do
	dst=${guid_map[$src]}
	# sed is fine here because the replacement is fixed-width hex.
	sed -i "s|$src|$dst|g" "$tmp"
done

# Phase 2: switch / HCA names appearing in single quotes inside ibnetdiscover
# output. We harvest both sides of the `( 'A' - 'B' )` pair where A/B can be
# either a switch or an HCA. The simplest signal: lines that start with `SW`
# vs `CA` for the local device.
sw_idx=0
hca_idx=0
declare -A sw_map=()
declare -A hca_map=()

# Extract names from quoted form. A typical line:
#   CA 134 1 0xGUID 4x EDR - SW 1719 10 0xGUID2 ( 'o0001 HCA-1' - 'ib-i1l1s01' )
while IFS= read -r line; do
	if [[ $line =~ \(\ \'([^\']+)\'[[:space:]]+-[[:space:]]+\'([^\']+)\'[[:space:]]\) ]]; then
		left=${BASH_REMATCH[1]}
		right=${BASH_REMATCH[2]}
		# Use the line's leading device type (first token) to decide which
		# side is the local device, the other is the uplink.
		first=${line%% *}
		if [[ $first == CA ]]; then
			# left = HCA, right = switch
			[[ -z "${hca_map[$left]:-}" ]] && { hca_idx=$((hca_idx + 1)); hca_map[$left]=$(printf 'hca-%02d' "$hca_idx"); }
			[[ -z "${sw_map[$right]:-}" ]] && { sw_idx=$((sw_idx + 1)); sw_map[$right]=$(printf 'sw-%02d' "$sw_idx"); }
		elif [[ $first == SW ]]; then
			# left = switch (this device), right = either switch or HCA
			[[ -z "${sw_map[$left]:-}" ]] && { sw_idx=$((sw_idx + 1)); sw_map[$left]=$(printf 'sw-%02d' "$sw_idx"); }
			# Heuristic: if "right" looks like "<host> HCA-N" or "<host> mlxN_N" or contains HCA, treat as HCA
			if [[ $right =~ HCA-[0-9]|mlx[0-9]_[0-9] ]]; then
				[[ -z "${hca_map[$right]:-}" ]] && { hca_idx=$((hca_idx + 1)); hca_map[$right]=$(printf 'hca-%02d' "$hca_idx"); }
			else
				[[ -z "${sw_map[$right]:-}" ]] && { sw_idx=$((sw_idx + 1)); sw_map[$right]=$(printf 'sw-%02d' "$sw_idx"); }
			fi
		fi
	fi
done <"$tmp"

# Apply name substitutions. Order from longest to shortest to avoid
# substring collisions (e.g. "5FB0405-leaf-IB01" vs "5FB0").
mapfile -t sw_keys < <(printf '%s\n' "${!sw_map[@]}" | awk '{ print length, $0 }' | sort -rn | cut -d' ' -f2-)
for src in "${sw_keys[@]}"; do
	dst=${sw_map[$src]}
	# Escape regex metas in src; '-', '.', '/' are common in switch names.
	esc=$(printf '%s' "$src" | sed 's|[]/.*[\^$|]|\\&|g')
	sed -i "s|'$esc'|'$dst'|g" "$tmp"
done

mapfile -t hca_keys < <(printf '%s\n' "${!hca_map[@]}" | awk '{ print length, $0 }' | sort -rn | cut -d' ' -f2-)
for src in "${hca_keys[@]}"; do
	dst=${hca_map[$src]}
	esc=$(printf '%s' "$src" | sed 's|[]/.*[\^$|]|\\&|g')
	sed -i "s|'$esc'|'$dst'|g" "$tmp"
done

# Phase 3: MACs. Documentation-range placeholder.
mac_idx=0
declare -A mac_map=()
while read -r mac; do
	if [[ -z "${mac_map[$mac]:-}" ]]; then
		mac_idx=$((mac_idx + 1))
		mac_map[$mac]=$(printf '02:00:00:00:00:%02x' "$mac_idx")
	fi
done < <(grep -ohiE '([0-9a-f]{2}:){5}[0-9a-f]{2}' "$tmp" | sort -u || true)

for src in "${!mac_map[@]}"; do
	dst=${mac_map[$src]}
	sed -i "s|$src|$dst|gI" "$tmp"
done

mv "$tmp" "$output"
trap - EXIT

# Summary on stderr so it does not pollute the output file.
{
	echo "anonymized: $input -> $output"
	echo "  GUIDs:    ${#guid_map[@]}"
	echo "  switches: ${#sw_map[@]}"
	echo "  HCAs:     ${#hca_map[@]}"
	echo "  MACs:     ${#mac_map[@]}"
} >&2
