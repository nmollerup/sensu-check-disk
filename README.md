[![Go Test](https://github.com/nmollerup/sensu-check-disk/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/nmollerup/sensu-check-disk/actions/workflows/test.yml)
[![goreleaser](https://github.com/nmollerup/sensu-check-disk/actions/workflows/release.yml/badge.svg?branch=main&event=release)](https://github.com/nmollerup/sensu-check-disk/actions/workflows/release.yml)

# Sensu Check Disk

## Table of Contents

- [Overview](#overview)
- [Usage](#usage)
  - [check-disk-usage](#check-disk-usage)
  - [metrics-disk-usage](#metrics-disk-usage)
- [Configuration](#configuration)
  - [Asset Registration](#asset-registration)
  - [Check Definition](#check-definition)
- [Installation from Source](#installation-from-source)
- [Contributing](#contributing)

## Overview

This plugin provides native disk instrumentation for monitoring and metrics collection in Sensu Go. It is a Golang replacement for [sensu-plugins/sensu-plugins-disk-checks](https://github.com/sensu-plugins/sensu-plugins-disk-checks).

Features:
- Check disk usage with configurable warning and critical thresholds
- Collect detailed disk usage metrics (bytes, inodes, percentages)
- Filter by filesystem type and mount paths
- Cross-platform support (Linux, macOS, Windows)
- Native Go binary with no runtime dependencies

## Usage

### check-disk-usage

Check disk usage against warning and critical thresholds.

```bash
check-disk-usage --warning 80 --critical 90
```

**Options:**

```
  -w, --warning float           Warning threshold percentage for disk usage
  -c, --critical float          Critical threshold percentage for disk usage
  -i, --ignore-paths strings    Comma-separated list of mount paths to ignore
  -I, --include-paths strings   Comma-separated list of mount paths to include (if set, only these are checked)
  -x, --ignore-types strings    Comma-separated list of filesystem types to ignore
  -t, --include-types strings   Comma-separated list of filesystem types to include (if set, only these are checked)
```

**Examples:**

Check all filesystems with 80% warning and 90% critical thresholds:
```bash
check-disk-usage --warning 80 --critical 90
```

Check only ext4 filesystems:
```bash
check-disk-usage --warning 80 --critical 90 --include-types ext4
```

Ignore tmpfs and devtmpfs filesystems:
```bash
check-disk-usage --warning 80 --critical 90 --ignore-types tmpfs,devtmpfs
```

Check specific mount points:
```bash
check-disk-usage --warning 80 --critical 90 --include-paths /,/home
```

### metrics-disk-usage

Output disk usage metrics in Graphite plaintext format.

```bash
metrics-disk-usage
```

**Options:**

```
  -s, --scheme string           Metric naming scheme prefix (default "disk_usage")
  -i, --ignore-paths strings    Comma-separated list of mount paths to ignore
  -I, --include-paths strings   Comma-separated list of mount paths to include (if set, only these are checked)
  -x, --ignore-types strings    Comma-separated list of filesystem types to ignore
  -t, --include-types strings   Comma-separated list of filesystem types to include (if set, only these are checked)
```

**Examples:**

Output metrics for all filesystems:
```bash
metrics-disk-usage
```

Output metrics with custom scheme:
```bash
metrics-disk-usage --scheme servers.disk
```

Output metrics only for ext4 filesystems:
```bash
metrics-disk-usage --include-types ext4
```

**Metrics Output:**

The plugin outputs the following metrics for each filesystem:
- `used_bytes` - Bytes used
- `total_bytes` - Total bytes
- `free_bytes` - Bytes free
- `used_percent` - Percentage used
- `inodes_used` - Inodes used
- `inodes_total` - Total inodes
- `inodes_free` - Inodes free
- `inodes_used_percent` - Percentage of inodes used

## Configuration

### Asset Registration

Assets are the best way to make use of this plugin. If you're not using an asset, please consider doing so!

```yaml
---
type: Asset
api_version: core/v2
metadata:
  name: sensu-check-disk
spec:
  builds:
    - url: https://github.com/nmollerup/sensu-check-disk/releases/download/{{ version }}/sensu-check-disk_{{ version }}_linux_amd64.tar.gz
      sha512: REPLACE_WITH_SHA512
      filters:
        - entity.system.os == 'linux'
        - entity.system.arch == 'amd64'
```

### Check Definition

**Check disk usage:**

```yaml
---
type: CheckConfig
api_version: core/v2
metadata:
  name: check-disk-usage
spec:
  command: check-disk-usage --warning 80 --critical 90 --ignore-types tmpfs,devtmpfs
  runtime_assets:
    - sensu-check-disk
  subscriptions:
    - system
  interval: 60
  publish: true
```

**Metrics collection:**

```yaml
---
type: CheckConfig
api_version: core/v2
metadata:
  name: metrics-disk-usage
spec:
  command: metrics-disk-usage --scheme disk_usage
  runtime_assets:
    - sensu-check-disk
  subscriptions:
    - system
  interval: 60
  output_metric_format: graphite_plaintext
  output_metric_handlers:
    - influxdb
  publish: true
```

## Installation from Source

Download the latest version or create an executable from this source.

**From source:**

```bash
go install github.com/nmollerup/sensu-check-disk/cmd/check-disk-usage@latest
go install github.com/nmollerup/sensu-check-disk/cmd/metrics-disk-usage@latest
```

**From release:**

Download the latest release from the [releases page](https://github.com/nmollerup/sensu-check-disk/releases).

**Build from source:**

```bash
git clone https://github.com/nmollerup/sensu-check-disk.git
cd sensu-check-disk
go build -o bin/check-disk-usage ./cmd/check-disk-usage
go build -o bin/metrics-disk-usage ./cmd/metrics-disk-usage
```

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.
