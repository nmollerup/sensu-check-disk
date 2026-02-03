package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	corev2 "github.com/sensu/core/v2"
	"github.com/sensu/sensu-plugin-sdk/sensu"
	"github.com/shirou/gopsutil/v3/disk"
)

// Config represents the check plugin config
type Config struct {
	sensu.PluginConfig
	FstabPath string
}

var (
	plugin = Config{
		PluginConfig: sensu.PluginConfig{
			Name:     "check-fstab-mounts",
			Short:    "Check that filesystems in fstab are mounted",
			Keyspace: "",
		},
	}

	options = []sensu.ConfigOption{
		&sensu.PluginConfigOption[string]{
			Path:      "FstabPath",
			Argument:  "fstab-path",
			Shorthand: "f",
			Default:   "/etc/fstab",
			Usage:     "Path to fstab file",
			Value:     &plugin.FstabPath,
		},
	}
)

type FstabEntry struct {
	Device     string
	MountPoint string
	FSType     string
	Options    string
}

func main() {
	check := sensu.NewCheck(&plugin.PluginConfig, options, checkArgs, executeCheck, false)
	check.Execute()
}

func checkArgs(event *corev2.Event) (int, error) {
	return sensu.CheckStateOK, nil
}

func executeCheck(event *corev2.Event) (int, error) {
	// Parse fstab
	entries, err := parseFstab(plugin.FstabPath)
	if err != nil {
		return sensu.CheckStateCritical, fmt.Errorf("failed to parse fstab: %v", err)
	}

	// Get currently mounted filesystems
	partitions, err := disk.Partitions(true)
	if err != nil {
		return sensu.CheckStateCritical, fmt.Errorf("failed to get mounted partitions: %v", err)
	}

	// Create a map of mounted paths
	mounted := make(map[string]bool)
	for _, partition := range partitions {
		mounted[partition.Mountpoint] = true
	}

	// Check which fstab entries are not mounted
	var unmounted []string
	for _, entry := range entries {
		// Skip swap, bind mounts, and special filesystems
		if entry.FSType == "swap" || strings.Contains(entry.Options, "bind") {
			continue
		}

		// Skip comments and special entries
		if strings.HasPrefix(entry.MountPoint, "#") || entry.MountPoint == "" {
			continue
		}

		// Check if mount point exists in mounted filesystems
		if !mounted[entry.MountPoint] {
			unmounted = append(unmounted, fmt.Sprintf("%s (%s)", entry.MountPoint, entry.Device))
		}
	}

	if len(unmounted) > 0 {
		fmt.Printf("CRITICAL - Filesystems not mounted: %v\n", unmounted)
		return sensu.CheckStateCritical, nil
	}

	fmt.Println("OK - All fstab filesystems are mounted")
	return sensu.CheckStateOK, nil
}

func parseFstab(path string) ([]FstabEntry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var entries []FstabEntry
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Split by whitespace
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		entry := FstabEntry{
			Device:     fields[0],
			MountPoint: fields[1],
			FSType:     fields[2],
			Options:    fields[3],
		}

		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}
