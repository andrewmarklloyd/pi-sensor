package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	brokerurl    = flag.String("brokerurl", os.Getenv("CLOUDMQTT_URL"), "The MQTT broker to connect")
	sensorSource = flag.String("sensorSource", os.Getenv("SENSOR_SOURCE"), "The sensor location or name")
	mockFlag     = flag.String("mockMode", os.Getenv("MOCK_MODE"), "Mock mode for local development")
	logger       = log.New(os.Stdout, "[Pi-Senser Client] ", log.LstdFlags)
)

const (
	topic   = "sensor/status"
	CLOSED  = "CLOSED"
	OPEN    = "OPEN"
	UNKNOWN = "UNKNOWN"
)

func main() {
	logger.Print("Initializing app")
	flag.Parse()
	if *brokerurl == "" {
		log.Fatalln("one broker is required")
	}

	mqttClient := newMQTTClient(*brokerurl, topic, *sensorSource)

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		logger.Println("Cleaning up")
		mqttClient.Cleanup()
		os.Exit(0)
	}()

	lastStatus := UNKNOWN
	var currentStatus string
	for true {
		currentStatus = randStatus()
		if currentStatus != lastStatus {
			logger.Println(fmt.Sprintf("%s is %s", *sensorSource, currentStatus))
			lastStatus = currentStatus
			mqttClient.publish(*sensorSource, currentStatus, time.Now().UTC().Unix())
		}
		// time.Sleep(50 * time.Millisecond)
	}
}

func randStatus() string {
	rand.Seed(time.Now().Unix())
	randStatus := []string{
		CLOSED,
		OPEN,
	}
	n := rand.Int() % len(randStatus)
	return randStatus[n]
}
