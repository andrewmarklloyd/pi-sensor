package main

import (
	"flag"
	"fmt"
	"html"
	"log"
	"net/http"
	"os"

	"github.com/andrewmarklloyd/pi-sensor/server/internal/pkg/state"
)

func init() {

}

var (
	brokerurl = flag.String("brokerurl", os.Getenv("CLOUDMQTT_URL"), "The broker to connect to")
	topic     = flag.String("topic", os.Getenv("TOPIC"), "The topic to subscribe")
	testMode  = flag.String("mockMode", os.Getenv("MOCK_MODE"), "Mock mode for local development")
	logger    = log.New(os.Stdout, "[Pi-Sensor Server] ", log.LstdFlags)

	stateClient state.Client
)

func main() {
	flag.Parse()

	if *brokerurl == "" {
		log.Fatalln("at least one broker is required")
	}
	if *topic == "" {
		log.Fatalln("topic to publish to is required")
	}

	mqttClient := newMQTTClient(*brokerurl, *topic)
	mqttClient.Subscribe(func(message string) {
		logger.Println(message)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
	})

	http.HandleFunc("/hi", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hi")
	})

	log.Fatal(http.ListenAndServe(":8081", nil))
}
