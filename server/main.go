package main

import (
	"flag"
	"fmt"
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

func getSensors(w http.ResponseWriter, req *http.Request) {
	if *testMode == "true" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
	}
	fmt.Fprintf(w, "[{\"garage\":\"OPEN\"}]")
	// fmt.Fprintf(w, fmt.Sprintf("{\"message\":\"%s\"}", "garage|OPEN"))
}

func main() {
	logger.Println("Initializing server")
	flag.Parse()

	if *brokerurl == "" {
		log.Fatalln("at least one broker is required")
	}
	if *topic == "" {
		log.Fatalln("topic to publish to is required")
	}
	port := os.Getenv("PORT")
	if port == "" {
		log.Fatalln("PORT must be set")
	}

	var webServer webServer
	webServer = newWebServer(port, getSensors)

	mqttClient := newMQTTClient(*brokerurl, *topic)
	mqttClient.Subscribe(func(message string) {
		webServer.sendMessage(message)
	})
	webServer.startServer()

	// on new web connections, send the last message for each topic to websocket clients
}
