package main

import (
	"fmt"
	"time"

	corev2 "github.com/sensu/core/v2"
	"github.com/sensu/sensu-plugin-sdk/sensu"
	"github.com/shirou/gopsutil/v3/disk"
)

// Config represents the metrics plugin config
type Config struct {
	sensu.PluginConfig
	Scheme       string
	IgnorePaths  []string
	IncludePaths []string
	IgnoreTypes  []string
	IncludeTypes []string
}

var (
	plugin = Config{
		PluginConfig: sensu.PluginConfig{
			Name:     "metrics-disk-capacity",
			Short:    "Output disk capacity metrics",
			Keyspace: "",
		},
	}

	options = []sensu.ConfigOption{
		&sensu.PluginConfigOption[string]{
			Path:      "Scheme",
			Argument:  "scheme",
			Shorthand: "s",
			Default:   "disk_capacity",
			Usage:     "Metric naming scheme prefix",
			Value:     &plugin.Scheme,
		},
		&sensu.SlicePluginConfigOption[string]{
			Path:      "IgnorePaths",
			Argument:  "ignore-paths",
			Shorthand: "i",
			Usage:     "Comma-separated list of mount paths to ignore",
			Value:     &plugin.IgnorePaths,
		},
		&sensu.SlicePluginConfigOption[string]{
			Path:      "IncludePaths",
			Argument:  "include-paths",
			Shorthand: "I",
			Usage:     "Comma-separated list of mount paths to include (if set, only these are checked)",
			Value:     &plugin.IncludePaths,
		},
		&sensu.SlicePluginConfigOption[string]{
			Path:      "IgnoreTypes",
			Argument:  "ignore-types",
			Shorthand: "x",
			Usage:     "Comma-separated list of filesystem types to ignore",
			Value:     &plugin.IgnoreTypes,
		},
		&sensu.SlicePluginConfigOption[string]{
			Path:      "IncludeTypes",
			Argument:  "include-types",
			Shorthand: "t",
			Usage:     "Comma-separated list of filesystem types to include (if set, only these are checked)",
			Value:     &plugin.IncludeTypes,
		},
	}
)

func main() {
	metric := sensu.NewGoHandler(&plugin.PluginConfig, options, checkArgs, executeMetric)
	metric.Execute()
}

func checkArgs(event *corev2.Event) error {
	return nil
}

func executeMetric(event *corev2.Event) error {
	partitions, err := disk.Partitions(false)
	if err != nil {
		return fmt.Errorf("failed to get disk partitions: %v", err)
	}

	timestamp := time.Now().Unix()

	for _, partition := range partitions {
		// Skip if filesystem type should be ignored
		if shouldIgnoreType(partition.Fstype) {
			continue
		}

		// Skip if not in include types (when include types is specified)
		if len(plugin.IncludeTypes) > 0 && !contains(plugin.IncludeTypes, partition.Fstype) {
			continue
		}

		// Skip if mount point should be ignored
		if contains(plugin.IgnorePaths, partition.Mountpoint) {
			continue
		}

		// Skip if not in include paths (when include paths is specified)
		if len(plugin.IncludePaths) > 0 && !contains(plugin.IncludePaths, partition.Mountpoint) {
			continue
		}

		usage, err := disk.Usage(partition.Mountpoint)
		if err != nil {
			// Skip partitions we can't read (e.g., permission issues)
			continue
		}

		// Sanitize mount point for metric name
		sanitizedMount := sanitizePath(partition.Mountpoint)

		// Output capacity metrics in Graphite plaintext format
		// Convert bytes to megabytes for capacity metrics
		usedMB := usage.Used / (1024 * 1024)
		totalMB := usage.Total / (1024 * 1024)
		freeMB := usage.Free / (1024 * 1024)

		fmt.Printf("%s.%s.used_mb %d %d\n", plugin.Scheme, sanitizedMount, usedMB, timestamp)
		fmt.Printf("%s.%s.total_mb %d %d\n", plugin.Scheme, sanitizedMount, totalMB, timestamp)
		fmt.Printf("%s.%s.free_mb %d %d\n", plugin.Scheme, sanitizedMount, freeMB, timestamp)
		fmt.Printf("%s.%s.used_percent %.2f %d\n", plugin.Scheme, sanitizedMount, usage.UsedPercent, timestamp)
	}

	return nil
}

func shouldIgnoreType(fstype string) bool {
	return contains(plugin.IgnoreTypes, fstype)
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func sanitizePath(path string) string {
	if path == "/" {
		return "root"
	}
	// Remove leading slash and replace remaining slashes with underscores
	sanitized := ""
	for i, c := range path {
		if i == 0 && c == '/' {
			continue
		}
		if c == '/' {
			sanitized += "_"
		} else {
			sanitized += string(c)
		}
	}
	return sanitized
}
