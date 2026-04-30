#!/usr/bin/env bash
# infiniband_exporter — pre-1.0 automated validation
#
# Non-disruptive: listens on a high port, never `pkill -f`, never `sudo`,
# all writes confined to ${TEST_DIR}/work, never installs anything.
#
# Layout expected in ${TEST_DIR} (default /tmp/test_ib):
#   infiniband_exporter         (executable, the binary under test)
#   ibswinfo.sh                 (executable, ibswinfo 0.9.0)
#   test_ib.sh                  (this script)
#
# Usage:
#   chmod +x /tmp/test_ib/test_ib.sh
#   cd /tmp/test_ib
#   bash test_ib.sh > out.txt 2>&1
#   # Send out.txt back. That's it.
#
# Override knobs (env vars):
#   TEST_DIR=/tmp/test_ib
#   PORT=19315
#   PERFQUERY_CONCURRENCY=8        # tweak if your fabric is bigger
#   IBSWINFO_CONCURRENCY=8

set -u  # don't set -e — we want to keep going on individual test failures

TEST_DIR="${TEST_DIR:-/tmp/test_ib}"
PORT="${PORT:-19315}"
PERFQUERY_CONCURRENCY="${PERFQUERY_CONCURRENCY:-8}"
IBSWINFO_CONCURRENCY="${IBSWINFO_CONCURRENCY:-8}"

BINARY="$TEST_DIR/infiniband_exporter"
IBSWINFO="$TEST_DIR/ibswinfo.sh"
WORK="$TEST_DIR/work"

# ──────────────────────────────────────────────────────────────────────
# Pre-flight
# ──────────────────────────────────────────────────────────────────────
echo "########################################################################"
echo "# infiniband_exporter automated validation"
echo "# host       : $(hostname)"
echo "# date       : $(date -Iseconds)"
echo "# kernel     : $(uname -r)"
echo "# TEST_DIR   : $TEST_DIR"
echo "# PORT       : $PORT"
echo "# pq concurr : $PERFQUERY_CONCURRENCY"
echo "# ibsw concur: $IBSWINFO_CONCURRENCY"
echo "########################################################################"
echo

if [[ ! -x "$BINARY" ]]; then
    echo "FAIL: $BINARY missing or not executable" >&2
    exit 1
fi
if [[ ! -x "$IBSWINFO" ]]; then
    echo "WARN: $IBSWINFO missing — ibswinfo-related tests will be skipped" >&2
    HAVE_IBSWINFO=0
else
    HAVE_IBSWINFO=1
fi

mkdir -p "$WORK"
rm -f "$WORK"/*.log "$WORK"/*.metrics 2>/dev/null

# Confirm the listen port is free; refuse to clobber an existing listener.
if ss -lnt 2>/dev/null | awk '{print $4}' | grep -qE "[:.]${PORT}\$"; then
    echo "FAIL: port $PORT already in use; pick another via PORT=… and rerun" >&2
    exit 2
fi

# ──────────────────────────────────────────────────────────────────────
# Helpers
# ──────────────────────────────────────────────────────────────────────
EXPORTER_PID=""

start_exporter() {
    local logfile="$1"; shift
    "$BINARY" "$@" --web.listen-address="0.0.0.0:$PORT" >"$logfile" 2>&1 &
    EXPORTER_PID=$!
    # Wait for the listener (max 10 s).
    for _ in $(seq 1 50); do
        if ss -lnt 2>/dev/null | awk '{print $4}' | grep -qE "[:.]${PORT}\$"; then
            return 0
        fi
        sleep 0.2
    done
    echo "FAIL: exporter did not bind on :$PORT within 10s. Log:" >&2
    tail -30 "$logfile" >&2
    return 1
}

stop_exporter() {
    if [[ -n "${EXPORTER_PID:-}" ]] && kill -0 "$EXPORTER_PID" 2>/dev/null; then
        kill "$EXPORTER_PID" 2>/dev/null
        wait "$EXPORTER_PID" 2>/dev/null
    fi
    EXPORTER_PID=""
}

# Cleanup if the script is interrupted.
on_exit() { stop_exporter; }
trap on_exit EXIT INT TERM

scrape_timed() {
    local out="$1"
    local label="$2"
    local s e dur lines
    s=$(date +%s.%N)
    curl -fsS -o "$out" "http://127.0.0.1:$PORT/metrics"
    e=$(date +%s.%N)
    dur=$(awk -v s="$s" -v e="$e" 'BEGIN{printf "%.3f", e-s}')
    lines=$(wc -l <"$out")
    printf "%-30s %s s   %s lines\n" "$label" "$dur" "$lines"
}

section() {
    echo
    echo "================================================================"
    echo "=== $* ==="
    echo "================================================================"
}

# ──────────────────────────────────────────────────────────────────────
# 0. Binary identity
# ──────────────────────────────────────────────────────────────────────
section "TEST 0 — binary identity"
sha256sum "$BINARY"
echo "--- --version ---"
"$BINARY" --version 2>&1
if [[ "$HAVE_IBSWINFO" == "1" ]]; then
    echo "--- ibswinfo helper ---"
    head -2 "$IBSWINFO" 2>&1
    sha256sum "$IBSWINFO"
fi

# ──────────────────────────────────────────────────────────────────────
# 1. --help (smoke check on flag surface)
# ──────────────────────────────────────────────────────────────────────
section "TEST 1 — --help / flag surface"
"$BINARY" --help 2>&1 | head -80

# ──────────────────────────────────────────────────────────────────────
# 2. Baseline: switch + HCA, no ibswinfo
# ──────────────────────────────────────────────────────────────────────
section "TEST 2 — baseline (switch + HCA, no ibswinfo)"
if start_exporter "$WORK/t2.log" \
    --collector.switch --collector.hca \
    --perfquery.max-concurrent="$PERFQUERY_CONCURRENCY" \
    --log.level=info; then
    echo "--- /healthz ---"
    curl -fsS "http://127.0.0.1:$PORT/healthz"; echo
    echo
    echo "--- 3 consecutive scrapes ---"
    for i in 1 2 3; do
        scrape_timed "$WORK/t2.metrics" "scrape #$i"
    done
    echo
    echo "--- metric family count ---"
    awk '/^# TYPE/{print $3}' "$WORK/t2.metrics" | sort -u | wc -l
    echo "--- exporter log: any error/warn? ---"
    grep -E 'level=(error|warn)' "$WORK/t2.log" | head -10 || echo "(none)"
    stop_exporter
fi

# ──────────────────────────────────────────────────────────────────────
# 3. + ibswinfo, default static-cache-ttl (15m)
# ──────────────────────────────────────────────────────────────────────
if [[ "$HAVE_IBSWINFO" == "1" ]]; then
    section "TEST 3 — switch + HCA + ibswinfo, cache 15m (default)"
    if start_exporter "$WORK/t3.log" \
        --collector.switch --collector.hca --collector.ibswinfo \
        --ibswinfo.path="$IBSWINFO" \
        --perfquery.max-concurrent="$PERFQUERY_CONCURRENCY" \
        --ibswinfo.max-concurrent="$IBSWINFO_CONCURRENCY" \
        --log.level=info; then
        for i in 1 2 3; do
            label="scrape #$i"
            [[ $i == 1 ]] && label="scrape #$i (cold cache → full ibswinfo)"
            [[ $i -gt 1 ]] && label="scrape #$i (warm cache → vitals)"
            scrape_timed "$WORK/t3.$i.metrics" "$label"
        done
        echo
        echo "--- hardware_info series count per scrape (must be identical) ---"
        for f in "$WORK"/t3.*.metrics; do
            n=$(grep -c '^infiniband_switch_hardware_info' "$f")
            echo "$(basename "$f"): $n"
        done
        echo
        echo "--- ibswinfo collector duration sample (last scrape) ---"
        grep '^infiniband_ibswinfo_collect_duration_seconds' "$WORK/t3.3.metrics" | head -3
        echo
        echo "--- exporter log: any error/warn? ---"
        grep -E 'level=(error|warn)' "$WORK/t3.log" | head -10 || echo "(none)"
        stop_exporter
    fi
fi

# ──────────────────────────────────────────────────────────────────────
# 4. ibswinfo cache disabled (regression baseline)
# ──────────────────────────────────────────────────────────────────────
if [[ "$HAVE_IBSWINFO" == "1" ]]; then
    section "TEST 4 — ibswinfo cache disabled (--ibswinfo.static-cache-ttl=0)"
    if start_exporter "$WORK/t4.log" \
        --collector.switch --collector.hca --collector.ibswinfo \
        --ibswinfo.path="$IBSWINFO" \
        --perfquery.max-concurrent="$PERFQUERY_CONCURRENCY" \
        --ibswinfo.max-concurrent="$IBSWINFO_CONCURRENCY" \
        --ibswinfo.static-cache-ttl=0 \
        --log.level=info; then
        for i in 1 2 3; do
            scrape_timed "$WORK/t4.metrics" "scrape #$i (cache off → full every time)"
        done
        echo
        echo "--- exporter log: any error/warn? ---"
        grep -E 'level=(error|warn)' "$WORK/t4.log" | head -10 || echo "(none)"
        stop_exporter
    fi
fi

# ──────────────────────────────────────────────────────────────────────
# 5. ibnetdiscover cache enabled
# ──────────────────────────────────────────────────────────────────────
section "TEST 5 — ibnetdiscover cache 5m"
if start_exporter "$WORK/t5.log" \
    --collector.switch --collector.hca \
    --perfquery.max-concurrent="$PERFQUERY_CONCURRENCY" \
    --ibnetdiscover.cache-ttl=5m \
    --log.level=info; then
    for i in 1 2 3; do
        label="scrape #$i"
        [[ $i == 1 ]] && label="scrape #$i (cold: ibnetdiscover runs)"
        [[ $i -gt 1 ]] && label="scrape #$i (warm: topology cached)"
        scrape_timed "$WORK/t5.$i.metrics" "$label"
    done
    echo
    echo "--- ibnetdiscover collector duration across scrapes ---"
    for f in "$WORK"/t5.*.metrics; do
        v=$(grep '^infiniband_exporter_collector_duration_seconds{collector="ibnetdiscover"' "$f" | awk '{print $NF}')
        echo "$(basename "$f"): $v s"
    done
    stop_exporter
fi

# ──────────────────────────────────────────────────────────────────────
# 6. port-state
# ──────────────────────────────────────────────────────────────────────
section "TEST 6 — --collector.switch.port-state"
if start_exporter "$WORK/t6.log" \
    --collector.switch --collector.hca \
    --collector.switch.port-state \
    --perfquery.max-concurrent="$PERFQUERY_CONCURRENCY" \
    --log.level=info; then
    curl -fsS "http://127.0.0.1:$PORT/metrics" >"$WORK/t6.metrics"
    echo "--- port_state series count, value distribution ---"
    grep '^infiniband_switch_port_state' "$WORK/t6.metrics" | awk '{print $NF}' | sort | uniq -c
    echo
    echo "--- sample down ports (value=0), if any ---"
    grep '^infiniband_switch_port_state' "$WORK/t6.metrics" | awk '$NF==0' | head -5 || echo "(none — fabric all up)"
    stop_exporter
fi

# ──────────────────────────────────────────────────────────────────────
# 7. runonce / textfile mode
# ──────────────────────────────────────────────────────────────────────
section "TEST 7 — --exporter.runonce"
rm -f "$WORK/runonce.prom" "$WORK/runonce.lock"
"$BINARY" \
    --collector.switch --collector.hca \
    --perfquery.max-concurrent="$PERFQUERY_CONCURRENCY" \
    --exporter.runonce \
    --exporter.lockfile="$WORK/runonce.lock" \
    --exporter.output="$WORK/runonce.prom" \
    --log.level=info \
    >"$WORK/t7.log" 2>&1
rc=$?
echo "exit code: $rc"
echo "--- output file ---"
ls -la "$WORK/runonce.prom" 2>&1
wc -l "$WORK/runonce.prom" 2>&1
echo
echo "--- last_execution present ---"
grep '^infiniband_exporter_last_execution' "$WORK/runonce.prom" || echo "(MISSING — bug)"
echo
echo "--- a few representative metrics ---"
grep -E '^infiniband_switch_(info|up|port_transmit_data_bytes_total)' "$WORK/runonce.prom" | head -10

# ──────────────────────────────────────────────────────────────────────
# 8. Shape sanity (everything wired together)
# ──────────────────────────────────────────────────────────────────────
section "TEST 8 — full configuration shape sanity"
if [[ "$HAVE_IBSWINFO" == "1" ]]; then
    EXTRA=(--collector.ibswinfo --ibswinfo.path="$IBSWINFO" --ibswinfo.max-concurrent="$IBSWINFO_CONCURRENCY")
else
    EXTRA=()
fi
if start_exporter "$WORK/t8.log" \
    --collector.switch --collector.hca \
    --collector.switch.port-state \
    --perfquery.max-concurrent="$PERFQUERY_CONCURRENCY" \
    "${EXTRA[@]}" \
    --log.level=info; then
    curl -fsS "http://127.0.0.1:$PORT/metrics" >"$WORK/t8.metrics"
    echo "--- HELP vs TYPE counts (should match) ---"
    echo "  # HELP : $(grep -c '^# HELP' "$WORK/t8.metrics")"
    echo "  # TYPE : $(grep -c '^# TYPE' "$WORK/t8.metrics")"
    echo
    echo "--- # TYPE by family (truncated) ---"
    awk '/^# TYPE/{print $3,$4}' "$WORK/t8.metrics" | sort -u | head -40
    echo
    echo "--- families with no samples observed ---"
    awk '
        /^# HELP/ { name=$3; help[name]=1 }
        /^# TYPE/ {}
        /^[^#]/ {
            for (n in help) {
                if (index($0, n)==1 && (substr($0,length(n)+1,1) == "{" || substr($0,length(n)+1,1) == " ")) {
                    sample[n]=1
                }
            }
        }
        END { for (n in help) if (!sample[n]) print "  " n }
    ' "$WORK/t8.metrics" | head -20
    echo
    echo "--- total samples count ---"
    grep -cv '^#' "$WORK/t8.metrics"
    stop_exporter
fi

# ──────────────────────────────────────────────────────────────────────
# 9. Latency comparison summary
# ──────────────────────────────────────────────────────────────────────
section "TEST 9 — latency comparison summary"
declare -a CFGS
CFGS+=("baseline                     |--collector.switch --collector.hca --perfquery.max-concurrent=$PERFQUERY_CONCURRENCY")
CFGS+=("ibnetdiscover-cache=5m       |--collector.switch --collector.hca --perfquery.max-concurrent=$PERFQUERY_CONCURRENCY --ibnetdiscover.cache-ttl=5m")
if [[ "$HAVE_IBSWINFO" == "1" ]]; then
    CFGS+=("+ibswinfo (cache 15m)        |--collector.switch --collector.hca --collector.ibswinfo --ibswinfo.path=$IBSWINFO --perfquery.max-concurrent=$PERFQUERY_CONCURRENCY --ibswinfo.max-concurrent=$IBSWINFO_CONCURRENCY")
    CFGS+=("+ibswinfo (cache=0)          |--collector.switch --collector.hca --collector.ibswinfo --ibswinfo.path=$IBSWINFO --perfquery.max-concurrent=$PERFQUERY_CONCURRENCY --ibswinfo.max-concurrent=$IBSWINFO_CONCURRENCY --ibswinfo.static-cache-ttl=0")
fi

printf "%-30s %-12s %-12s\n" "config" "cold scrape" "warm scrape"
echo "--------------------------------------------------------------------"
for cfg in "${CFGS[@]}"; do
    label="${cfg%%|*}"
    args="${cfg#*|}"
    if start_exporter "$WORK/t9.log" $args --log.level=info >/dev/null 2>&1; then
        cold=$(curl -fsS -w '%{time_total}\n' -o /dev/null "http://127.0.0.1:$PORT/metrics")
        warm=$(curl -fsS -w '%{time_total}\n' -o /dev/null "http://127.0.0.1:$PORT/metrics")
        printf "%-30s %-12s %-12s\n" "$label" "${cold}s" "${warm}s"
        stop_exporter
    else
        printf "%-30s %s\n" "$label" "FAIL TO START"
        stop_exporter
    fi
done

# ──────────────────────────────────────────────────────────────────────
# Done
# ──────────────────────────────────────────────────────────────────────
section "DONE"
echo "Logs and metric dumps preserved under: $WORK"
echo "Send the entire stdout of this script back. If anything looks"
echo "suspicious, also send the relevant $WORK/*.log and $WORK/*.metrics."
