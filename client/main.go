package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"strconv"
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
	topic = "sensor/status"
)

func main() {
	logger.Print("Initializing app")
	flag.Parse()
	if *brokerurl == "" {
		log.Fatalln("one broker is required")
	}
	if *sensorSource == "" {
		log.Fatalln("sensorSource is required")
	}
	mockMode, _ := strconv.ParseBool(*mockFlag)

	defaultPin := 15
	pinNum, err := strconv.Atoi(os.Getenv("GPIO_PIN"))
	if err != nil {
		logger.Printf("Failed to parse GPIO_PIN env var, using default %d", defaultPin)
		pinNum = defaultPin
	}

	mqttClient := newMQTTClient(*brokerurl, topic, *sensorSource, mockMode)
	pinClient := newPinClient(pinNum, mockMode)

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		logger.Println("Cleaning up")
		mqttClient.Cleanup()
		pinClient.Cleanup()
		os.Exit(0)
	}()

	lastStatus := "CLOSED"
	currentStatus := pinClient.CurrentStatus()
	for true {
		currentStatus = pinClient.CurrentStatus()
		if currentStatus != lastStatus {
			lastStatus = currentStatus
			mqttClient.publish(*sensorSource, currentStatus, time.Now().UTC().Unix())
		}
		time.Sleep(5 * time.Second)
	}
}
