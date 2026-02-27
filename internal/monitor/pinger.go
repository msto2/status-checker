package monitor

import (
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// PingResult represents the result of a ping operation
type PingResult struct {
	Success      bool
	ResponseTime int // milliseconds
	Error        string
}

// Ping executes a system ping command and returns the result
func Ping(ipAddress string, timeout int, count int) PingResult {
	var cmd *exec.Cmd

	// Platform-specific ping command
	if runtime.GOOS == "windows" {
		// Windows: ping -n <count> -w <timeout_ms> <ip>
		cmd = exec.Command("ping", "-n", fmt.Sprintf("%d", count),
			"-w", fmt.Sprintf("%d", timeout*1000), ipAddress)
	} else {
		// Linux/Unix: ping -c <count> -W <timeout_sec> <ip>
		cmd = exec.Command("ping", "-c", fmt.Sprintf("%d", count),
			"-W", fmt.Sprintf("%d", timeout), ipAddress)
	}

	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil {
		// Ping command failed (host unreachable, timeout, etc.)
		return PingResult{
			Success: false,
			Error:   fmt.Sprintf("ping failed: %v", err),
		}
	}

	// Parse response time from output
	responseTime := extractResponseTime(outputStr)

	// Check if ping was successful
	if responseTime > 0 || containsSuccessIndicator(outputStr) {
		return PingResult{
			Success:      true,
			ResponseTime: responseTime,
		}
	}

	return PingResult{
		Success: false,
		Error:   "no response received",
	}
}

// extractResponseTime parses the ping output to extract average response time
func extractResponseTime(output string) int {
	// Try to find time= pattern (works for both Windows and Linux)
	// Examples:
	// Windows: "Reply from 192.168.1.1: bytes=32 time=1ms TTL=64"
	// Linux: "64 bytes from 192.168.1.1: icmp_seq=1 ttl=64 time=0.5 ms"

	// Pattern for "time=XXms" or "time=XX.XXms" or "time<1ms"
	timeRegex := regexp.MustCompile(`time[=<]\s*(\d+(?:\.\d+)?)\s*ms`)
	matches := timeRegex.FindAllStringSubmatch(output, -1)

	if len(matches) == 0 {
		return 0
	}

	// Calculate average if multiple times found
	var total float64
	count := 0
	for _, match := range matches {
		if len(match) > 1 {
			if val, err := strconv.ParseFloat(match[1], 64); err == nil {
				total += val
				count++
			}
		}
	}

	if count > 0 {
		return int(total / float64(count))
	}

	return 0
}

// containsSuccessIndicator checks if the ping output indicates success
func containsSuccessIndicator(output string) bool {
	output = strings.ToLower(output)

	// Windows success indicators
	if strings.Contains(output, "reply from") {
		return true
	}

	// Linux success indicators
	if strings.Contains(output, "bytes from") {
		return true
	}

	// Check for packet loss statistics
	// Example: "4 packets transmitted, 4 received, 0% packet loss"
	packetLossRegex := regexp.MustCompile(`(\d+)\s+received`)
	matches := packetLossRegex.FindStringSubmatch(output)
	if len(matches) > 1 {
		if received, err := strconv.Atoi(matches[1]); err == nil && received > 0 {
			return true
		}
	}

	return false
}
