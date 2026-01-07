package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/op"
)

func main() {
	limited, resetDuration, err := op.GetRateLimit()
	if err != nil {
		panic(err)
	}

	if !limited {
		os.Exit(0)
	}

	fmt.Printf("being rate limited by 1password until %s from now, starting maintenance web server\n", resetDuration)

	ticker := time.NewTicker(resetDuration)
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
