#!/bin/sh
set -o pipefail

TEMPLATE="./examples/templates/json/linux_memory_telemetry.json.tmpl"

while true; do
    # 1. Parse /proc/meminfo using awk to strip out whitespace and the "kB" unit suffixes
    MEM_METRICS=$(awk '
        $1 ~ /^(MemTotal|MemFree|MemAvailable|Buffers|Cached|Active):/ {
            sub(/:/, "", $1)
            print $1 "=" $2
        }
    ' /proc/meminfo)

    # 2. Inject the stream into gsub. If any core variable goes missing,
    # execution terminates safely without flooding corrupted JSON to stdout.
    echo "$MEM_METRICS" | \
        ./gsub -e -t "$TEMPLATE" -f - -p "[mem-worker]" | \
        jq --unbuffered -c . || break

    sleep 5
done
