package main

import (
	"flag"
	"log"
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
	mqttClient.Subscribe()
}
