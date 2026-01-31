package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	corev2 "github.com/sensu/core/v2"
	"github.com/sensu/sensu-plugin-sdk/sensu"
)

// Config represents the check plugin config
type Config struct {
	sensu.PluginConfig
	Devices      []string
	SmartctlPath string
	ConfigFile   string
}

type SmartConfig struct {
	Devices []string `json:"devices"`
}

var (
	plugin = Config{
		PluginConfig: sensu.PluginConfig{
			Name:     "check-smart",
			Short:    "Check SMART disk health status",
			Keyspace: "",
		},
	}

	options = []sensu.ConfigOption{
		&sensu.SlicePluginConfigOption[string]{
			Path:      "Devices",
			Argument:  "devices",
			Shorthand: "d",
			Usage:     "Comma-separated list of devices to check (e.g., /dev/sda,/dev/sdb)",
			Value:     &plugin.Devices,
		},
		&sensu.PluginConfigOption[string]{
			Path:      "SmartctlPath",
			Argument:  "smartctl-path",
			Shorthand: "s",
			Default:   "smartctl",
			Usage:     "Path to smartctl binary",
			Value:     &plugin.SmartctlPath,
		},
		&sensu.PluginConfigOption[string]{
			Path:      "ConfigFile",
			Argument:  "config-file",
			Shorthand: "c",
			Default:   "/etc/sensu/conf.d/smart.json",
			Usage:     "Path to JSON config file with device list",
			Value:     &plugin.ConfigFile,
		},
	}
)

func main() {
	check := sensu.NewCheck(&plugin.PluginConfig, options, checkArgs, executeCheck, false)
	check.Execute()
}

func checkArgs(event *corev2.Event) (int, error) {
	// Load devices from config file if no devices specified
	if len(plugin.Devices) == 0 {
		if _, err := os.Stat(plugin.ConfigFile); err == nil {
			data, err := os.ReadFile(plugin.ConfigFile)
			if err == nil {
				var config SmartConfig
				if err := json.Unmarshal(data, &config); err == nil {
					plugin.Devices = config.Devices
				}
			}
		}
	}

	// If still no devices, check common device patterns
	if len(plugin.Devices) == 0 {
		plugin.Devices = detectDevices()
	}

	if len(plugin.Devices) == 0 {
		return sensu.CheckStateWarning, fmt.Errorf("no devices specified or detected")
	}

	return sensu.CheckStateOK, nil
}

func executeCheck(event *corev2.Event) (int, error) {
	var failures []string
	var warnings []string

	for _, device := range plugin.Devices {
		// Run smartctl -H (health check)
		cmd := exec.Command("sudo", plugin.SmartctlPath, "-H", device)
		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		if err != nil {
			// Check if it's an actual failure or just unsupported
			if strings.Contains(outputStr, "Unsupported") || strings.Contains(outputStr, "Unknown") {
				warnings = append(warnings, fmt.Sprintf("%s: SMART not supported", device))
				continue
			}
			failures = append(failures, fmt.Sprintf("%s: %v", device, err))
			continue
		}

		// Parse output for health status
		if strings.Contains(outputStr, "PASSED") {
			continue
		} else if strings.Contains(outputStr, "FAILING_NOW") || strings.Contains(outputStr, "FAILED") {
			failures = append(failures, fmt.Sprintf("%s: SMART health check FAILED", device))
		} else {
			warnings = append(warnings, fmt.Sprintf("%s: Unknown SMART status", device))
		}
	}

	if len(failures) > 0 {
		fmt.Printf("CRITICAL - SMART health failures: %v\n", failures)
		return sensu.CheckStateCritical, nil
	}

	if len(warnings) > 0 {
		fmt.Printf("WARNING - SMART warnings: %v\n", warnings)
		return sensu.CheckStateWarning, nil
	}

	fmt.Println("OK - All SMART health checks passed")
	return sensu.CheckStateOK, nil
}

func detectDevices() []string {
	var devices []string

	// Check for common device names
	commonDevices := []string{
		"/dev/sda", "/dev/sdb", "/dev/sdc", "/dev/sdd",
		"/dev/nvme0n1", "/dev/nvme1n1",
	}

	for _, device := range commonDevices {
		if _, err := os.Stat(device); err == nil {
			devices = append(devices, device)
		}
	}

	return devices
}
