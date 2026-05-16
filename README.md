
# gsub

**gsub** is an ultra-fast, stream-oriented template engine written in Go. It sits natively inside Unix pipelines to identify `{{PLACEHOLDERS}}` and inject values from systemic environment variables, `.env` files, or continuous standard input streams before flushing the parsed result directly to `STDOUT`.

Unlike traditional templating engines that parse an entire file into memory before executing, `gsub` features an **instant-flush streaming engine**. It evaluates data line-by-line, resulting in an exceptionally lean memory footprint capable of handling infinite telemetry streams, massive configuration files, and heavy automation pipelines without bottlenecking or breaking template formatting.

---

##  Key Features

* **Zero Dependencies**: Compiled into a single, isolated binary (~2MB) with no runtime dependencies.
* **Format Agnostic**: Purely unopinionated layout preservation. It perfectly maintains the original formatting, indentation, and structure of **JSON, YAML, HTML, Markdown, or plain-text** templates.
* **Polymorphic STDIN**: Operates as a traditional template compiler or flips into an open-ended, data-hungry streaming engine based on your arguments.
* **Granular Scope Isolation**: Explicitly force namespaces using `{{$env.VAR}}` or `{{$file.VAR}}` to permanently prevent variable collisions between environmental contexts and standard input data streams.
* **Built-in Fallbacks**: Native short-circuit syntax (`{{VAR || 'default'}}`) enables graceful operational degradation without causing pipeline termination.
* **Logfmt Native**: Emits production-grade, single-line cloud-native error metrics to `STDERR` for seamless parsing by log aggregators (e.g., Datadog, Splunk, Loki).
* **Safety First**: Strictly aborts and fails loud if a variable is missing by default—preventing corrupted deployments, broken network configs, or half-baked payloads.
* **Layered Overrides**: Implements explicit variable precedence out of the box (System Env > `.env` file) when running un-namespaced placeholders.
* **Transparent Escaping**: Simply prefix with a backslash `\{{LITERAL}}` to ignore substitution (ideal for technical documentation or frontend frameworks).

---

##  Installation

### Build from Source

Ensure you have a modern Go toolchain installed on your environment, then compile:

```bash
go build -o gsub


```

To make `gsub` accessible globally across your environment, move the compiled binary to your local execution path:

```bash
sudo mv gsub /usr/local/bin/


```

---

## Isolated Sandbox Testing (Podman / Docker)

If you want to test or verify `gsub` inside a completely clean, isolated Linux container environment without installing Go on your host machine or dealing with mount permissions, you can use **Podman** (native on Fedora/RHEL) or **Docker**.

This deployment workflow syncs your code state via `rsync` over the container's secure native transport layer—perfect for simulating real-world remote CI/CD pipelines.

### 1. Launch a Fresh, Unmounted Container

Spin up a completely clean Alpine Linux container running detached in the background:

```bash
# Using Podman
podman run -it -d --name gsub-build-env alpine:latest /bin/sh

# Using Docker (Alternative)
docker run -it -d --name gsub-build-env alpine:latest /bin/sh

```

### 2. Provision the Container Toolchain

Install the lightweight Alpine utilities, compiler toolchain, and `rsync` target engine inside the container:

```bash
# Using Podman
podman exec -it gsub-build-env apk add --no-cache go unzip jq gawk rsync

# Using Docker
docker exec -it gsub-build-env apk add --no-cache go unzip jq gawk rsync

```

### 3. Package and Sync the Source Tree (From Host)

Navigate to your local repository directory, package the tracked Git snapshot, and transfer it over the secure container engine transport:

```bash
# Generate a clean deployment zip of your tracked code state
git archive --format=zip HEAD -o ./gsub-source.zip

# Push the zip file directly into the container using rsync transport wrappers
# If using Podman:
rsync -avz -e "podman exec -i" ./gsub-source.zip gsub-build-env:/tmp/

# If using Docker:
rsync -avz -e "docker exec -i" ./gsub-source.zip gsub-build-env:/tmp/

```

### 4. Compile and Run Inside the Container Environment

Attach to the container shell to unzip your archive, build the native binary, and execute the continuous telemetry examples:

```bash
# Drop into the container's interactive shell
podman exec -it gsub-build-env /bin/sh

# (Inside Container Prompt) Unpack the source archive
mkdir -p /app
unzip /tmp/gsub-source.zip -d /app
cd /app

# Compile and symlink globally inside the container system
go build -o gsub main.go
ln -s $(pwd)/gsub /usr/local/bin/gsub

# Export your pipeline environment tracking variable and run the examples!
export USER=alpine
./examples/amd_zen_thermals/json-example.sh

```

Once executed, you will see real-time hardware telemetry JSON frames streaming cleanly to your console window every 5 seconds. Press `Ctrl+C` to halt execution.

---

## Operational Modes & Core Mechanics

`gsub` dynamically adapts its behavior based on the presence of the `--template (-t)` flag. This shifts `gsub` between two highly distinct operational architectures:

### Mode 1: Static Configuration Mode (Without `-t`)

`STDIN` is treated entirely as the structural template text. `gsub` reads the template from the pipe, pulls variables from static sources (system environment or a file on disk), evaluates the tokens, and exits immediately.

### Mode 2: Continuous Telemetry Mode (With `-t`)

By declaring a template file using `-t`, `STDIN` is completely freed to act as an open data stream. When you couple this with `-f -`, `gsub` holds the structural template in memory and processes incoming `KEY=VALUE` strings line-by-line indefinitely, instantly flashing parsed blocks down the pipeline.

---

## Interpolation Syntax Guide

`gsub` supports multiple advanced token paradigms inside standard double curly braces:

| Pattern Syntax | Evaluation Logic |
| --- | --- |
| `{{NODE_NAME}}` | **Standard Variable:** Waterfalls down your context. Evaluates against System Env first (if `-e` is passed), then falls back to your input file/stream. |
| `{{$env.USER}}` | **Strict Environment Scope:** Bypasses data streams completely to fetch from system variables. |
| `{{$file.MemFree}}` | **Strict Stream Scope:** Bypasses system environment data to query fields parsed directly out of your incoming text files or `STDIN`. |
| `{{RACK_ID \|\| 'za-midrand-01'}}` | **Default Fallback:** If the variable cannot be resolved by standard lookup, the tool falls back to the literal string between single quotes without throwing a pipeline failure. Supports namespaced keys like `{{$file.STATUS \|\| 'operational'}}`. |

---

## Practical Examples & Use Cases

### 1. The Classic One-Shot Pipe (Mode 1)

Inject system environment variables natively into a basic string statement.

```bash
export NODE_NAME="Core-Switch-01"
echo "Connecting to {{NODE_NAME}}..." | gsub -e
# Output: Connecting to Core-Switch-01...


```

### 2. Configuration Management with Layered Precedence (Mode 1)

Manage system defaults safely via a physical `.env` file while preserving the ability to dynamically override specific settings via your shell environment.

> **Precedence Rule:** System environment variables (`-e`) always take priority over target file variables (`-f`) on un-namespaced keys. Missing variables fall back to the file.

```text
# defaults.env
DB_USER=app_user
DB_PORT=5432


```

```bash
# Override the database user on the fly without touching the configuration file
export DB_USER=root

gsub -e -f defaults.env < config.tmpl
# {{DB_USER}} resolves cleanly to 'root'
# {{DB_PORT}} falls back to '5432'


```

### 3. Continuous Telemetry & Ingestion Pipelines (Mode 2)

Transform raw, live system metrics into structured, real-time JSON payloads and ship them directly to an aggregation server or an API webhook.

#### The Blueprint

In this architecture, your data collector script formats metrics into standard `KEY=VALUE` pairs and flushes them. `gsub` intercepts these blocks, binds them alongside systemic environment profiles (like `{{USER}}`), and maps them directly to your template.

```
+------------------+     Raw KEY=VALUE Stream      +---------+     Structured JSON     +------------------+
| Linux Kernel     | ----------------------------> |  gsub   | ----------------------> | Remote Analytics |
| (/proc/meminfo)  |   MemTotal=31921860           |   -e    |   { "host": "admin",    | Server / Webhook |
+------------------+   MemFree=24560652            +---------+     "metrics": {...} }  +------------------+
                                                        ^
                                                        |
                                              [ payload.json.tmpl ]


```

#### A. The Template (`payload.json.tmpl`):

```json
{
  "host": "{{USER}}",
  "metrics": {
    "total": "{{MemTotal}}",
    "free": "{{MemFree}}",
    "available": "{{MemAvailable}}"
  }
}


```

#### B. The Pipeline Loop:

Run this continuous tracking loop to feed standard input frames into `gsub` at a 5-second interval. It extracts the values from the kernel virtual file, strips the colons using `awk`, and passes the data cleanly down the line:

```bash
while true; do
    # 1. Capture the raw metrics frame as standard KEY=VALUE string pairs
    METRICS=$(awk '/^(MemTotal|MemFree|MemAvailable):/ { sub(/:/, "", $1); print $1 "=" $2 }' /proc/meminfo)
    
    # 2. Feed the frame via STDIN into gsub
    echo "$METRICS" | gsub -e -t payload.json.tmpl -f -
    
    sleep 5
done


```

#### C. Expected Stream Output:

```json
{
  "host": "admin",
  "metrics": {
    "total": "31921860",
    "free": "24560652",
    "available": "27369956"
  }
}


```

---

### 4. Advanced Hardware Thermals via Downstream Pipelines (UNIX Philosophy)

This example streams physical CPU silicon temperatures out of the kernel hardware monitoring tree (`hwmon`). Following the Unix philosophy of chaining specialized tools, we pass `gsub`'s cleanly formatted template output downstream into `jq` to output single-line **Newline Delimited JSON (NDJSON)** strings ready for `curl` or logging collectors.

#### A. The Template (`thermal_dashboard.json.tmpl`):

```json
{
  "host": "{{$env.USER}}",
  "resource": "amd_zen_thermals",
  "status": "{{$file.SYS_STATUS || 'operational'}}",
  "metrics": {
    "control_peak_celsius": {{$file.Tctl}},
    "core_die_celsius": {{$file.Tccd}}
  }
}


```

#### B. The Compacting Pipeline Loop:

By utilizing `jq -c .` downstream, `gsub` remains unopinionated while the system gains a bulletproof JSON minifier out of the box:

```bash
while true; do
    # Read temperature millidegrees across separate files and export structural metrics pairs
    THERMAL_METRICS=$(awk '
        FILENAME ~ /temp1_input/ { tctl = $1/1000 }
        FILENAME ~ /temp3_input/ { tccd = $1/1000 }
        END {
            print "Tctl=" tctl
            print "Tccd=" tccd
        }
    ' /sys/class/hwmon/hwmon3/temp1_input /sys/class/hwmon/hwmon3/temp3_input)

    # Stream into gsub and pipe downstream to jq for compacting NDJSON rendering
    echo -e "$THERMAL_METRICS" | gsub -e -t thermal_dashboard.json.tmpl -f - | jq -c .

    sleep 5
done


```

#### C. Expected Stream Output (Single Line Lines / NDJSON):

```json
{"host":"admin","resource":"amd_zen_thermals","status":"operational","metrics":{"control_peak_celsius":37.875,"core_die_celsius":26.000}}
{"host":"admin","resource":"amd_zen_thermals","status":"operational","metrics":{"control_peak_celsius":36.625,"core_die_celsius":24.125}}


```

---

## Configuration Reference

| Long Flag | Short Flag | Description |
| --- | --- | --- |
| `--env` | `-e` | Pull values from systemic environment variables. Takes precedence on global keys if combined with `-f`. |
| `--file` | `-f` | Target path to a `.env` file. Pass `-` to instruct `gsub` to stream `KEY=VALUE` variables directly from `STDIN`. |
| `--template` | `-t` | Path to a target template file. Frees `STDIN` to process variable streams. |
| `--allow-missing` | `-a` | Bypasses safety checks. Missing placeholders remain untouched in the output stream instead of throwing an error. Ignored if a token has an inline fallback default value. |
| `--prefix` | `-p` | Customizes the logging signature prefix for `STDERR` outputs (default: `[gsub]`). |
| `--version` | `-v` | Print compilation binary version. |
| `--help` | `-h` | Display help matrix and CLI routing. |

---

## Error Handling & Microservice Logging Safety

`gsub` is deliberately built to **fail loud and fast** to block downstream infrastructure corruption.

To remain native to centralized cloud log aggregators, `gsub` dumps errors into a strict, single-line `logfmt` string written entirely to `STDERR` before dropping execution with exit status code `1`. Every missing variable is sorted alphabetically and joined with a clean comma delimiter without spaces.

Because error logs stream to `STDERR`, they can be easily captured or isolated inside monitoring pipelines without contaminating your valid data flowing down `STDOUT`.

### Default Error Format:

```text
state=failed reason=missing_vars targets=API_KEY,CLOUD_ID fix=--allow-missing


```

### Error Format with Operational Prefix (`-p "[worker-01]"`):

```text
[worker-01] state=failed reason=missing_vars targets=API_KEY,CLOUD_ID fix=--allow-missing


```

---

## License

Distributed under the terms of the **MIT License**.
