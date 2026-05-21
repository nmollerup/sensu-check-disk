# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Build a specific binary
go build -o bin/check-disk-usage ./cmd/check-disk-usage
go build -o bin/metrics-disk-usage ./cmd/metrics-disk-usage

# Build all binaries
go build ./...

# Run all tests
go test -v ./...

# Run tests for a specific package
go test -v ./cmd/check-fstab-mounts/...

# Lint / vet
go vet ./...

# Check formatting (CI enforces this)
gofmt -s -l .

# Format in place
gofmt -s -w .
```

## Architecture

Each binary lives in its own `cmd/<name>/` directory as a standalone `package main` with no shared library code — logic is duplicated across commands by design. The pattern is consistent across all commands:

1. A `Config` struct embeds `sensu.PluginConfig` and holds CLI flag values.
2. A global `plugin` instance and `options` slice declare flag bindings via `sensu.PluginConfigOption` / `sensu.SlicePluginConfigOption`.
3. `main()` wires everything together via `sensu.NewCheck` (for checks) or `sensu.NewGoHandler` (for metrics) and calls `.Execute()`.
4. Two callbacks are passed: `checkArgs` for validation and `executeCheck`/`executeMetric` for the actual logic.

**Check commands** (`check-*`) return Sensu status codes (`CheckStateOK`, `CheckStateWarning`, `CheckStateCritical`) and print human-readable status lines to stdout.

**Metric commands** (`metrics-*`) output Graphite plaintext format: `<scheme>.<sanitized_mount>.<metric_name> <value> <unix_timestamp>`.

**Disk enumeration** uses `github.com/shirou/gopsutil/v3/disk` for cross-platform partition and usage data. **SMART checks** shell out to `smartctl` via `os/exec` (with `sudo`).

**Filtering** (paths and filesystem types) follows the same include/ignore pattern in every disk command: ignore-list takes precedence; include-list acts as an allowlist when non-empty.

## Releases

Releases are built with GoReleaser (`CGO_ENABLED=0`) targeting `linux/{386,amd64,arm,arm64}`. Trigger a release by pushing a git tag. The `.goreleaser.yml` defines the per-binary build matrix; the test workflow (`test.yml`) runs `go test`, `go vet`, and `gofmt` on every push to `main` and on PRs.
