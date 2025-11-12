// Copyright 2025 Scott Friedman
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ami

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ProgressInfo contains parsed progress information from build logs.
type ProgressInfo struct {
	Message        string
	CurrentPackage string
	PackageIndex   int
	TotalPackages  int
	Percent        int
	Timestamp      time.Time
}

// parseProgressMarker parses a PCTL_PROGRESS log line.
// Expected format: "PCTL_PROGRESS: Installing gcc@11.3.0 (1/5 packages, 25%)"
func parseProgressMarker(line string) *ProgressInfo {
	if !strings.Contains(line, "PCTL_PROGRESS:") {
		return nil
	}

	info := &ProgressInfo{
		Message:   strings.TrimSpace(strings.SplitN(line, "PCTL_PROGRESS:", 2)[1]),
		Timestamp: time.Now(),
	}

	// Extract package name (e.g., "Installing gcc@11.3.0")
	packageRe := regexp.MustCompile(`Installing ([^\s]+)`)
	if matches := packageRe.FindStringSubmatch(info.Message); len(matches) > 1 {
		info.CurrentPackage = matches[1]
	}

	// Extract package progress (e.g., "(1/5 packages")
	countRe := regexp.MustCompile(`\((\d+)/(\d+) packages`)
	if matches := countRe.FindStringSubmatch(info.Message); len(matches) > 2 {
		info.PackageIndex, _ = strconv.Atoi(matches[1])
		info.TotalPackages, _ = strconv.Atoi(matches[2])
	}

	// Extract percentage (e.g., "25%")
	percentRe := regexp.MustCompile(`(\d+)%`)
	if matches := percentRe.FindStringSubmatch(info.Message); len(matches) > 1 {
		info.Percent, _ = strconv.Atoi(matches[1])
	}

	return info
}

// formatProgressBar creates a visual progress bar.
// Example: [==================>                    ] 45%
func formatProgressBar(percent int, width int) string {
	if width < 10 {
		width = 40
	}

	filled := (percent * width) / 100
	if filled > width {
		filled = width
	}

	var bar strings.Builder
	bar.WriteString("[")

	for i := 0; i < filled; i++ {
		bar.WriteString("=")
	}

	if filled < width {
		bar.WriteString(">")
		for i := filled + 1; i < width; i++ {
			bar.WriteString(" ")
		}
	}

	bar.WriteString("]")
	return bar.String()
}

// estimateTimeRemaining calculates estimated time remaining based on progress.
func estimateTimeRemaining(percent int, elapsed time.Duration) time.Duration {
	if percent <= 0 {
		return 0
	}
	if percent >= 100 {
		return 0
	}

	// Calculate total time based on current progress
	totalTime := time.Duration(float64(elapsed) / float64(percent) * 100)
	remaining := totalTime - elapsed

	if remaining < 0 {
		return 0
	}

	return remaining
}

// formatDuration formats a duration in a human-readable way.
// Examples: "2m 15s", "45m", "1h 30m"
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}

	if d < time.Hour {
		minutes := int(d.Minutes())
		seconds := int(d.Seconds()) % 60
		if seconds > 0 {
			return fmt.Sprintf("%dm %ds", minutes, seconds)
		}
		return fmt.Sprintf("%dm", minutes)
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	if minutes > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dh", hours)
}

// displayProgress shows formatted progress output.
func displayProgress(info *ProgressInfo, startTime time.Time) {
	elapsed := time.Since(startTime)

	// Clear current line and move cursor to beginning
	fmt.Print("\r\033[K")

	// Progress bar
	bar := formatProgressBar(info.Percent, 40)

	// Build output
	if info.CurrentPackage != "" && info.TotalPackages > 0 {
		// Package-level progress
		fmt.Printf("📦 Installing %s (%d/%d packages) %s %d%%",
			info.CurrentPackage,
			info.PackageIndex,
			info.TotalPackages,
			bar,
			info.Percent)
	} else {
		// General progress
		fmt.Printf("📦 %s %s %d%%",
			info.Message,
			bar,
			info.Percent)
	}

	// Time estimate
	if info.Percent > 0 && info.Percent < 100 {
		remaining := estimateTimeRemaining(info.Percent, elapsed)
		if remaining > 0 {
			fmt.Printf(" (~%s remaining)", formatDuration(remaining))
		}
	}
}

// displayNoProgressWarning shows a warning when no progress has been seen for a while.
func displayNoProgressWarning(lastUpdate time.Time) {
	fmt.Print("\r\033[K") // Clear line

	elapsed := time.Since(lastUpdate)

	fmt.Printf("\n⚠️  Warning: No progress updates for %s\n", formatDuration(elapsed))
	fmt.Println("   This can happen if:")
	fmt.Println("   - Package is compiling from source (normal for large packages)")
	fmt.Println("   - Network latency to CloudWatch Logs API")
	fmt.Println("   - Instance is experiencing issues")
	fmt.Println()
	fmt.Println("   Build is still running. Waiting for updates...")
	fmt.Print("\n")
}

// displayCompletionMessage shows a completion message with total time.
func displayCompletionMessage(startTime time.Time) {
	fmt.Print("\r\033[K") // Clear line

	elapsed := time.Since(startTime)

	fmt.Printf("\n✅ Package installation complete! (total time: %s)\n", formatDuration(elapsed))
}
