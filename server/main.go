package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

func init() {

}

var (
	brokerurl = flag.String("brokerurl", os.Getenv("CLOUDMQTT_URL"), "The broker to connect to")
	topic     = flag.String("topic", os.Getenv("TOPIC"), "The topic to subscribe")
	redisurl  = flag.String("redisurl", os.Getenv("REDIS_URL"), "The redis cluster to connect to")
	mockMode  = flag.String("mockMode", os.Getenv("MOCK_MODE"), "Mock mode for local development")
	logger    = log.New(os.Stdout, "[Pi-Sensor Server] ", log.LstdFlags)
)

var _webServer webServer
var _redisClient redisClient

func newClientHandler() {
	state, _ := _redisClient.ReadAllState()
	for k, v := range state {
		_webServer.sendMessage(fmt.Sprintf("%s|%s", k, v))
	}
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
	if *redisurl == "" {
		log.Fatalln("redisurl is required")
	}
	port := os.Getenv("PORT")
	if port == "" {
		log.Fatalln("PORT must be set")
	}

	_webServer = newWebServer(port, newClientHandler)

	mqttClient := newMQTTClient(*brokerurl, *topic)
	mqttClient.Subscribe(func(message string) {
		messageSplit := strings.Split(message, "|")
		_redisClient.WriteState(messageSplit[0], messageSplit[1])
		_webServer.sendMessage(message)
	})
	// var err error
	_redisClient, _ = newRedisClient(*redisurl)

	_webServer.startServer()
}
