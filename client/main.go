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
	topic        = flag.String("topic", os.Getenv("TOPIC"), "The topic to consume")
	sensorSource = flag.String("sensorSource", os.Getenv("SENSOR_SOURCE"), "The sensor location or name")
	logger       = log.New(os.Stdout, "[Pi-Senser Client] ", log.LstdFlags)
)

func main() {
	logger.Print("Initializing app")
	flag.Parse()
	if *brokerurl == "" {
		log.Fatalln("one broker is required")
	}
	if *topic == "" {
		log.Fatalln("topic to publish to is required")
	}
	if *sensorSource == "" {
		log.Fatalln("sensorSource is required")
	}

	defaultPin := 15
	pinNum, err := strconv.Atoi(os.Getenv("GPIO_PIN"))
	if err != nil {
		logger.Printf("Failed to parse GPIO_PIN env var, using default %d", defaultPin)
		pinNum = defaultPin
	}

	mqttClient := newMQTTClient(*brokerurl, *topic)

	mockMode := os.Getenv("MOCK_MODE") == "true"
	pinClient := newPinClient(pinNum, mockMode)
	logger.Println(pinClient)

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
			logger.Printf("Current status: %s", currentStatus)
			mqttClient.publish(*sensorSource, currentStatus)
		}
		time.Sleep(10 * time.Second)
	}
}
