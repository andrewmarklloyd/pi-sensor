package op

import (
	"bufio"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func GetRateLimit() (bool, time.Duration, error) {
	out, err := getRateLimitOutput()
	if err != nil {
		return true, time.Hour, err
	}
	return parseRateLimitOutput(out)
}

func parseRateLimitOutput(out string) (bool, time.Duration, error) {
	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 0 || fields[0] == "TYPE" {
			continue
		}
		if len(fields) < 6 {
			return true, time.Hour, fmt.Errorf("less than 6 fields were found when running ratelimit command")
		}

		limit, _ := strconv.Atoi(fields[2])
		used, _ := strconv.Atoi(fields[3])
		remaining, _ := strconv.Atoi(fields[4])
		reset := strings.Join(fields[5:], " ")

		slog.Info("rate limit status",
			"type", fields[0],
			"action", fields[1],
			"limit", limit,
			"used", used,
			"remaining", remaining,
			"reset", reset,
		)

		if remaining == 0 {
			dur, err := parseOPResetDuration(reset)
			if err != nil {
				return true, time.Hour, err
			}
			return true, dur, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return true, time.Hour, err
	}

	return false, time.Hour, nil
}

func getRateLimitOutput() (string, error) {
	cmd := exec.Command("op", "service-account", "ratelimit")

	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	return string(out), nil
}

// parseOPResetDuration parses the "RESET" field from
// `op service-account ratelimit` output into a time.Duration.
//
// Supports examples like:
//
//	"N/A"
//	"57 minutes from now"
//	"23 hours from now"
//	"1 hour 12 minutes from now"
//	"2 days 3 hours from now"
//	"1 minute 20 seconds from now"
func parseOPResetDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(strings.ToLower(s))

	if s == "n/a" {
		return 0, nil
	}

	// Must end with "from now"
	if !strings.HasSuffix(s, "from now") {
		return 0, fmt.Errorf("invalid RESET format: %q", s)
	}

	// Remove trailing "from now"
	s = strings.TrimSuffix(s, "from now")
	s = strings.TrimSpace(s)

	// Tokenize: e.g.
	// "1 hour 12 minutes" → ["1","hour","12","minutes"]
	parts := strings.Fields(s)
	if len(parts) == 0 {
		return 0, errors.New("empty duration")
	}
	if len(parts)%2 != 0 {
		return 0, fmt.Errorf("invalid duration expression: %q", s)
	}

	var total time.Duration

	for i := 0; i < len(parts); i += 2 {
		numStr := parts[i]
		unitStr := parts[i+1]

		n, err := strconv.Atoi(numStr)
		if err != nil {
			return 0, fmt.Errorf("invalid number %q: %w", numStr, err)
		}
		if n < 0 {
			return 0, fmt.Errorf("negative duration not allowed: %d", n)
		}

		var dur time.Duration
		switch {
		case strings.HasPrefix(unitStr, "sec"):
			dur = time.Duration(n) * time.Second
		case strings.HasPrefix(unitStr, "min"):
			dur = time.Duration(n) * time.Minute
		case strings.HasPrefix(unitStr, "hour"):
			dur = time.Duration(n) * time.Hour
		case strings.HasPrefix(unitStr, "day"):
			dur = time.Duration(n) * 24 * time.Hour
		default:
			return 0, fmt.Errorf("unknown unit %q", unitStr)
		}

		total += dur
	}

	return total, nil
}
