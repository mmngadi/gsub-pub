
# `gsub` Integration Examples

This directory contains standalone telemetry orchestration scripts and data templates designed to demonstrate how `gsub` behaves as a fail-safe, atomic validation component within production streaming pipelines.

---

##  Prerequisites & Build Step

Before executing any of these automation examples, you **must compile the `gsub` binary locally** at the root folder of this repository. The shell scripts rely on path alignment to target your compiled Go binary.

Navigate to the repository root directory and run:

```bash
# Move to the repository root and compile the binary
go build -o gsub main.go

```

To make testing easier inside isolated environments, you can link the binary globally inside your terminal frame:

```bash
sudo ln -s $(pwd)/gsub /usr/local/bin/gsub

```

---

## Directory Structure

```text
./examples/
├── README.md
├── amd_zen_thermals
│   └── example.sh            # Streams CPU hardware metrics as structured JSON
├── linux_memory_telemetry
│   └── example.sh            # Streams kernel RAM statistics as structured JSON
└── templates
    └── json
        ├── amd_zen_thermals.json.tmpl
        └── linux_memory_telemetry.json.tmpl

```

---

## Running the Telemetry Monitors

The automation scripts in this suite leverage native `awk` engines to harvest real-time data fields directly from Linux kernel subsystems (`/sys/class` and `/proc`), feed them into `gsub` for line-by-line template substitution, and pass them downstream to `jq` for compact Newline Delimited JSON (NDJSON) rendering.

### 1. AMD Zen Thermals Stream

```bash
cd ./examples/amd_zen_thermals
chmod +x example.sh
./example.sh

```

**Expected Output (STDOUT):**

```json
{"host":"alpine","resource":"amd_zen_thermals","status":"operational","metrics":{"control_peak_celsius":"36.125","core_die_celsius":"25.625"}}
{"host":"alpine","resource":"amd_zen_thermals","status":"operational","metrics":{"control_peak_celsius":"35.875","core_die_celsius":"26"}}

```

### 2. Linux Memory Allocation Monitor

```bash
cd ./examples/linux_memory_telemetry
chmod +x example.sh
./example.sh

```

---

## Atomic Fail-Safe Architecture

Every orchestration script in this suite is initialized with `set -o pipefail` and strictly runs in a loop that checks the validation health of the data engine.

Because `gsub` handles **atomic parsing**, if a critical system variable definition disappears, an environment key drops, or data becomes corrupted mid-stream, `gsub` will:

1. **Halt execution instantly** before flushing any broken text downstream.
2. **Safeguard your data pipelines.** It discards the incomplete block so that downstream analytics collectors or log aggregators never ingest malformed payloads.
3. **Emit logfmt metrics to `STDERR`.** It dumps a clean, cloud-native error trace (prefixed with its `-p [worker]` tracing header) directly to `STDERR`, allowing you to catch processing errors without contaminating your valid `STDOUT` log streams.
