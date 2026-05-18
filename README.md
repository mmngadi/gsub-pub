# gsub

**gsub** is an ultra-fast, stream-oriented template engine written in Go. Operating natively inside Unix pipelines, it parses `{{PLACEHOLDERS}}` on the fly and safely injects variables sourced from system environments, `.env` files, or standard input.

Featuring an **instant-flush streaming architecture**, `gsub` evaluates inputs line-by-line. This ensures a remarkably low memory footprint capable of handling continuous telemetry streams, immense configuration structures, and high-frequency automation pipelines without blocking or buffering data.

---

## Key Features

* **Zero Dependencies**: Compiles into a single, isolated, resource-lean binary (~2MB).
* **Format Agnostic**: Strictly preserves original spacing, indentation, and structure across **JSON, YAML, HTML, Markdown, or plain-text**.
* **Dual Operational Modes**: Runs as a one-shot configuration compiler or shifts into a persistent, open-ended streaming ingestion daemon.
* **Strict Safety First**: Operates on a zero-trust model. It fails loud and fast if a template variable is completely missing from your environment.
* **Inline Fallbacks**: Built-in default string syntax (`{{VAR || 'fallback'}}`) allows graceful operational fallback values without risking pipeline drops.
* **Differentiates Empty vs. Absent**: Explicitly treats empty strings (`VAR=""`) as valid configurations, while throwing failures only for completely omitted variables.
* **Granular Scoping**: Explicit namespaces (`{{$env.VAR}}` or `{{$file.VAR}}`) completely eliminate value collisions between system contexts and custom data streams.
* **Transparent Escaping**: Prefix placeholders with a backslash `\{{LITERAL}}` to skip engine parsing entirely.

---

## Installation

### Build from Source

Ensure you have a modern Go toolchain installed on your system, then compile:

```bash
go build -o gsub

```

Move the compiled binary to your local execution path to make it globally accessible:

```bash
sudo mv gsub /usr/local/bin/

```

---

## Reference Layouts

To use the examples below, assume the following sample assets are present in your workspace:

### `vars.env`

```text
USER=alpine
TIMESTAMP=1748365200

```

### `payload.json.tmpl`

```json
{
  "user": "{{USER}}",
  "generated_at": "{{TIMESTAMP}}"
}

```

---

## Basic Usage

### 1. Feed Key-Value Pairs via STDIN

Stream variables directly down standard input to render your target template:

```bash
echo "USER=alpine\nTIMESTAMP=$(date +%s)" | gsub -t payload.json.tmpl -f -

```

### 2. Read Variables from an Env File

Pass static paths explicitly, or use standard Unix redirection to push structural templates through:

```bash
gsub -t payload.json.tmpl -f vars.env
# or
gsub -f vars.env < payload.json.tmpl

```

### 3. Source from System Environment Variables

Instruct `gsub` via `-e` to capture contexts native to your active terminal session or process:

```bash
export USER=alpine
export TIMESTAMP=$(date +%s)

gsub -e < payload.json.tmpl
# or
gsub -e -t payload.json.tmpl

```

### 4. Layer Cascading Overrides

Combine system environments alongside file defaults.

> **Precedence Rule:** Un-namespaced system environment flags (`-e`) permanently take priority over values read inside static asset scopes (`-f`). Missing variables fall back to the file context cleanly.

```bash
export USER=alpine
export TIMESTAMP=$(date +%s)

gsub -e -f vars.env < payload.json.tmpl
# or
gsub -e -f vars.env -t payload.json.tmpl

# Note: If USER is declared inside vars.env, the live environment 
# assignment 'export USER=alpine' will safely override it at runtime.

```

---

## Continuous Streaming Telemetry Example

Transform live system metrics streams into structured, real-time JSON payloads using persistent pipelines.

### The Streaming Architecture

By declaring a structural layout with `--template (-t)` and routing variables using `-f -`, `gsub` binds your layout in RAM and maps incoming micro-batches on the fly. Empty lines (`\n`) serve as the frame boundary signal to flush the rendered object directly to `STDOUT`.

```
+------------------+     Raw KEY=VALUE Stream      +---------+     Structured JSON     +------------------+
| Linux Kernel     | ----------------------------> |  gsub   | ----------------------> | Downstream Tools |
| (/proc/meminfo)  |   MemTotal=31921860           |   -e    |   { "host": "root",     | (e.g., jq, curl) |
+------------------+   MemFree=24560652            +---------+     "metrics": {...} }  +------------------+
                                                        ^
                                                        |
                                              [ telemetry.json.tmpl ]

```

### 1. The Template File (`telemetry.json.tmpl`)

```json
{
  "host": "{{$env.USER}}",
  "metrics": {
    "total": "{{MemTotal}}",
    "free": "{{MemFree}}",
    "available": "{{MemAvailable}}",
    "cached": "{{Cached}}"
  }
}

```

### 2. The Persistent Shell Loop Pipeline

This continuous background tracking script reads metrics every 5 seconds, transforms raw fields into uniform key-value pairs, inserts the mandatory blank line delimiter, and pumps them directly through `gsub` downstream to `jq`:

```bash
#!/bin/sh
set -o pipefail

TEMPLATE="telemetry.json.tmpl"
export USER=$(whoami)

while true; do
    # 1. Parse structural kernel metrics and clean string suffixes
    awk '
        $1 ~ /^(MemTotal|MemFree|MemAvailable|Cached):/ {
            sub(/:/, "", $1)
            print $1 "=" $2
        }
    ' /proc/meminfo
    
    # 2. Print an explicit empty newline to trigger the frame boundary flush
    echo ""
    
    sleep 5
done | gsub -e -t "$TEMPLATE" -f - -p "[mem-worker]" | jq --unbuffered -c .

```

### 3. Expected Stream Output (NDJSON)

```json
{"host":"root","metrics":{"total":"31921860","free":"24560652","available":"27369956","cached":"4128456"}}
{"host":"root","metrics":{"total":"31921860","free":"24558124","available":"27367320","cached":"4128512"}}

```

If a core data metric goes missing or becomes unavailable without an explicitly assigned template fallback default value, `gsub` aborts execution instantly to prevent downstream monitoring pipeline corruption.

---

## Configuration Reference

| Long Flag | Short Flag | Description |
| --- | --- | --- |
| `--env` | `-e` | Cascade evaluation into systemic environment variables. Takes operational precedence over file matches. |
| `--file` | `-f` | Path targeting a standard `.env` configuration file. Use `-` to read live stream variable pairs directly from `STDIN`. |
| `--template` | `-t` | Path to the template layout file. Using this releases `STDIN` to act exclusively as an infinite variable feed. |
| `--prefix` | `-p` | Toggles or sets custom runtime tracking prefix tags on logs written to `STDERR` (default: `[gsub] `). |
| `--version` | `-v` | Prints compiled binary version payload. |
| `--help` | `-h` | Prints available command routing matrices. |

---

## Error Diagnostics Formatting

`gsub` logs strictly to `STDERR` using structured `logfmt` lines, keeping your standard output streams clean and easy to parse using centralized logging platforms.

```text
[mem-worker] state=failed reason=missing_vars targets=MemTotal,MemFree

```

---

## License

Distributed under the terms of the **MIT License**.
