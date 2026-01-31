package main

import (
	"fmt"

	corev2 "github.com/sensu/core/v2"
	"github.com/sensu/sensu-plugin-sdk/sensu"
	"github.com/shirou/gopsutil/v3/disk"
)

// Config represents the check plugin config
type Config struct {
	sensu.PluginConfig
	Warning     float64
	Critical    float64
	IgnorePaths []string
	IncludePaths []string
	IgnoreTypes []string
	IncludeTypes []string
}

var (
	plugin = Config{
		PluginConfig: sensu.PluginConfig{
			Name:     "check-disk-usage",
			Short:    "Check disk usage and inodes",
			Keyspace: "",
		},
	}

	options = []sensu.ConfigOption{
		&sensu.PluginConfigOption[float64]{
			Path:      "Warning",
			Argument:  "warning",
			Shorthand: "w",
			Usage:     "Warning threshold percentage for disk usage",
			Value:     &plugin.Warning,
		},
		&sensu.PluginConfigOption[float64]{
			Path:      "Critical",
			Argument:  "critical",
			Shorthand: "c",
			Usage:     "Critical threshold percentage for disk usage",
			Value:     &plugin.Critical,
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
	check := sensu.NewCheck(&plugin.PluginConfig, options, checkArgs, executeCheck, false)
	check.Execute()
}

func checkArgs(event *corev2.Event) (int, error) {
	if plugin.Critical <= 0 {
		return sensu.CheckStateWarning, fmt.Errorf("--critical is required and must be greater than 0")
	}
	if plugin.Warning <= 0 {
		return sensu.CheckStateWarning, fmt.Errorf("--warning is required and must be greater than 0")
	}
	if plugin.Warning >= plugin.Critical {
		return sensu.CheckStateWarning, fmt.Errorf("--warning must be less than --critical")
	}
	return sensu.CheckStateOK, nil
}

func executeCheck(event *corev2.Event) (int, error) {
	partitions, err := disk.Partitions(false)
	if err != nil {
		return sensu.CheckStateCritical, fmt.Errorf("failed to get disk partitions: %v", err)
	}

	var warnings []string
	var criticals []string

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

		usedPercent := usage.UsedPercent

		if usedPercent >= plugin.Critical {
			criticals = append(criticals, fmt.Sprintf("%s at %.2f%% usage", partition.Mountpoint, usedPercent))
		} else if usedPercent >= plugin.Warning {
			warnings = append(warnings, fmt.Sprintf("%s at %.2f%% usage", partition.Mountpoint, usedPercent))
		}
	}

	// Return critical if any critical thresholds are exceeded
	if len(criticals) > 0 {
		fmt.Printf("CRITICAL - Disk usage exceeded critical threshold on: %v\n", criticals)
		return sensu.CheckStateCritical, nil
	}

	// Return warning if any warning thresholds are exceeded
	if len(warnings) > 0 {
		fmt.Printf("WARNING - Disk usage exceeded warning threshold on: %v\n", warnings)
		return sensu.CheckStateWarning, nil
	}

	fmt.Println("OK - All disk usage within thresholds")
	return sensu.CheckStateOK, nil
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
