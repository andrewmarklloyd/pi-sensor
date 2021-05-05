package main

import (
	"flag"
	"log"
	"os"
)

func init() {

}

var (
	brokerurl = flag.String("brokerurl", os.Getenv("CLOUDMQTT_URL"), "The broker to connect to")
	topic     = flag.String("topic", os.Getenv("TOPIC"), "The topic to subscribe")
	mockMode  = flag.String("mockMode", os.Getenv("MOCK_MODE"), "Mock mode for local development")
	logger    = log.New(os.Stdout, "[Pi-Sensor Server] ", log.LstdFlags)
)

var _webServer webServer

func newClientHandler() {
	logger.Println("New client handler, sending status for all")
	// get latest status for all sensors and send them to the client
	_webServer.sendMessage("garage|OPEN")
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

	_webServer = newWebServer(port, newClientHandler)

	mqttClient := newMQTTClient(*brokerurl, *topic)
	mqttClient.Subscribe(func(message string) {
		_webServer.sendMessage(message)
	})
	_webServer.startServer()
}
