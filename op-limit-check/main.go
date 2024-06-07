package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

func main() {
	limited, resetDuration, err := GetRateLimit()
	if err != nil {
		panic(err)
	}

	if !limited {
		os.Exit(0)
	}

	fmt.Printf("being rate limited by 1password until %s, starting maintenance web server\n", resetDuration)

	// todo: capture sleep time from output
	// for example 11 hours etc
	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		for range ticker.C {
			os.Exit(0)
		}
	}()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "This site is in maintenance mode")
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func GetRateLimit() (bool, string, error) {
	cmd := exec.Command("/app/op", "service-account", "ratelimit")

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
