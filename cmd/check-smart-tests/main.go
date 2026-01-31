package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	corev2 "github.com/sensu/core/v2"
	"github.com/sensu/sensu-plugin-sdk/sensu"
)

// Config represents the check plugin config
type Config struct {
	sensu.PluginConfig
	Devices           []string
	SmartctlPath      string
	ConfigFile        string
	ShortTestInterval int
	LongTestInterval  int
}

type SmartConfig struct {
	Devices []string `json:"devices"`
}

var (
	plugin = Config{
		PluginConfig: sensu.PluginConfig{
			Name:     "check-smart-tests",
			Short:    "Check SMART self-test status and timing",
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
		&sensu.PluginConfigOption[int]{
			Path:      "ShortTestInterval",
			Argument:  "short-test-interval",
			Shorthand: "l",
			Default:   24,
			Usage:     "Maximum hours since last short test (0 to disable)",
			Value:     &plugin.ShortTestInterval,
		},
		&sensu.PluginConfigOption[int]{
			Path:      "LongTestInterval",
			Argument:  "long-test-interval",
			Shorthand: "t",
			Default:   336,
			Usage:     "Maximum hours since last extended test (0 to disable, default 14 days)",
			Value:     &plugin.LongTestInterval,
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
		// Run smartctl -a to get all SMART information including test log
		cmd := exec.Command("sudo", plugin.SmartctlPath, "-a", device)
		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		if err != nil {
			// Check if it's an actual failure or just unsupported
			if strings.Contains(outputStr, "Unsupported") || strings.Contains(outputStr, "Unknown") {
				continue
			}
		}

		// Parse test log
		shortTestAge, longTestAge, testFailures := parseTestLog(outputStr)

		// Check for test failures
		if len(testFailures) > 0 {
			failures = append(failures, fmt.Sprintf("%s: Tests failed: %s", device, strings.Join(testFailures, ", ")))
			continue
		}

		// Check short test interval
		if plugin.ShortTestInterval > 0 && shortTestAge > plugin.ShortTestInterval {
			warnings = append(warnings, fmt.Sprintf("%s: Short test not run in %d hours (threshold: %d)", 
				device, shortTestAge, plugin.ShortTestInterval))
		}

		// Check long test interval
		if plugin.LongTestInterval > 0 && longTestAge > plugin.LongTestInterval {
			warnings = append(warnings, fmt.Sprintf("%s: Extended test not run in %d hours (threshold: %d)", 
				device, longTestAge, plugin.LongTestInterval))
		}
	}

	if len(failures) > 0 {
		fmt.Printf("CRITICAL - SMART test failures: %v\n", failures)
		return sensu.CheckStateCritical, nil
	}

	if len(warnings) > 0 {
		fmt.Printf("WARNING - SMART test interval warnings: %v\n", warnings)
		return sensu.CheckStateWarning, nil
	}

	fmt.Println("OK - All SMART tests passed and within time intervals")
	return sensu.CheckStateOK, nil
}

func parseTestLog(output string) (shortTestAge int, longTestAge int, failures []string) {
	shortTestAge = 999999
	longTestAge = 999999

	// Look for test log entries
	lines := strings.Split(output, "\n")
	inTestLog := false

	for _, line := range lines {
		if strings.Contains(line, "Self-test Log") {
			inTestLog = true
			continue
		}

		if !inTestLog {
			continue
		}

		// Parse test log line
		// Format: # 1  Short offline    Completed without error       00%     12345         -
		re := regexp.MustCompile(`#\s+\d+\s+(Short|Extended|Long)\s+\w+\s+(.*?)\s+\d+%\s+(\d+)`)
		matches := re.FindStringSubmatch(line)
		
		if len(matches) > 3 {
			testType := matches[1]
			status := strings.TrimSpace(matches[2])
			lifeHours, _ := strconv.Atoi(matches[3])

			// Calculate age in hours (approximate)
			age := lifeHours

			// Check for failures
			if strings.Contains(strings.ToLower(status), "fail") || 
			   strings.Contains(strings.ToLower(status), "error") {
				failures = append(failures, fmt.Sprintf("%s test at %d hours", testType, lifeHours))
				continue
			}

			// Update test ages (we want the most recent test)
			if strings.HasPrefix(testType, "Short") && age < shortTestAge {
				shortTestAge = age
			}
			if strings.HasPrefix(testType, "Extended") || strings.HasPrefix(testType, "Long") {
				if age < longTestAge {
					longTestAge = age
				}
			}
		}
	}

	// Convert to hours since test
	now := time.Now()
	if shortTestAge != 999999 {
		shortTestAge = int(now.Sub(time.Now().Add(-time.Duration(shortTestAge) * time.Hour)).Hours())
	}
	if longTestAge != 999999 {
		longTestAge = int(now.Sub(time.Now().Add(-time.Duration(longTestAge) * time.Hour)).Hours())
	}

	return
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
