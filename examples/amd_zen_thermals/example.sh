#!/bin/sh
set -o pipefail

TEMPLATE="./examples/templates/json/amd_zen_thermals.json.tmpl"

while true; do
    # 1. Capture system hardware temps
    THERMAL_METRICS=$(awk '
        FILENAME ~ /temp1_input/ { tctl = $1/1000 }
        FILENAME ~ /temp3_input/ { tccd = $1/1000 }
        END {
            print "Tctl=" tctl
            print "Tccd=" tccd
        }
    ' /sys/class/hwmon/hwmon3/temp1_input /sys/class/hwmon/hwmon3/temp3_input)

    # 2. Direct streaming pipeline.
    # gsub blocks malformed output natively. If it exits 1, pipefail triggers the '|| break'
    echo -e "$THERMAL_METRICS" | \
        ./gsub -e -t "$TEMPLATE" -f - -p "[worker-1]" | \
        jq --unbuffered -c . || break

    sleep 5
done
