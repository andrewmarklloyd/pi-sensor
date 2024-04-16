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
	limited, err := GetRateLimit()
	if err != nil {
		panic(err)
	}

	if !limited {
		os.Exit(0)
	}

	fmt.Println("being rate limited by 1password, starting maintenance web server")

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

func GetRateLimit() (bool, error) {
	cmd := exec.Command("/app/op", "service-account", "ratelimit")

	out, err := cmd.Output()
	if err != nil {
		return true, err
	}

	output := []string{}
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		text := scanner.Text()
		if !strings.Contains(text, "TYPE") && !strings.Contains(text, "ACTION") && !strings.Contains(text, "N/A") {
			output = append(output, text)
		}
	}

	if err := scanner.Err(); err != nil {
		return true, err
	}

	if len(output) > 0 {
		fmt.Println(string(out))
		return true, nil
	}

	return false, nil
}
