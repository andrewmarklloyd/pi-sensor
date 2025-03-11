package op

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
)

func GetRateLimit() (bool, string, error) {
	cmd := exec.Command("op", "service-account", "ratelimit")

	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(out))
		return true, "", err
	}

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 6 {
			return true, "", fmt.Errorf("less than 6 fields were found when running ratelimit command")
		}
		if fields[4] == "0" {
			return true, strings.Join(fields[5:], " "), nil
		}
	}

	if err := scanner.Err(); err != nil {
		return true, "", err
	}

	return false, "", nil
}
