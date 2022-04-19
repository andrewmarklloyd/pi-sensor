package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/robfig/cron"
)

var (
	brokerurl    = flag.String("brokerurl", os.Getenv("CLOUDMQTT_URL"), "The MQTT broker to connect")
	sensorSource = flag.String("sensorSource", os.Getenv("SENSOR_SOURCE"), "The sensor location or name")
	mockFlag     = flag.String("mockMode", os.Getenv("MOCK_MODE"), "Mock mode for local development")
	logger       *log.Logger
)

const (
	topic                    = "sensor/status"
	heartbeatIntervalSeconds = 60
	statusFile               = "/home/pi/.pi-sensor-status"
)

func main() {
	flag.Parse()
	if *brokerurl == "" {
		log.Fatalln("one broker is required")
	}
	if *sensorSource == "" {
		log.Fatalln("sensorSource is required")
	}

	logger = log.New(os.Stdout, fmt.Sprintf("[Pi-Sensor Agent-%s] ", *sensorSource), log.LstdFlags)
	version := os.Getenv("APP_VERSION")
	logger.Print("Initializing app, version:", version)

	mockMode, _ := strconv.ParseBool(*mockFlag)

	defaultPin := 18
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

	configureHeartbeat(mqttClient, *sensorSource)

	mqttClient.subscribeRestart(func(messageString string) {
		if *sensorSource == messageString {
			logger.Println("Received restart message, restarting app now")
			os.Exit(0)
		}
	})

	lastStatus, err := getLastStatus(statusFile)
	if err != nil {
		logger.Println(fmt.Errorf("error reading status file: %s. Setting status to %s", err, UNKNOWN))
		lastStatus = UNKNOWN
	}

	var currentStatus string
	for true {
		currentStatus = pinClient.CurrentStatus()
		err = writeStatus(statusFile, currentStatus)
		if err != nil {
			logger.Println("error writing status file:", err)
		}
		if currentStatus != lastStatus {
			logger.Println(fmt.Sprintf("%s is %s", *sensorSource, currentStatus))
			lastStatus = currentStatus
			mqttClient.publish(*sensorSource, currentStatus, time.Now().UTC().Unix())
		}
		time.Sleep(5 * time.Second)
	}
}

func configureHeartbeat(mqttClient mqttClient, sensorSource string) {
	cronLib := cron.New()
	cronLib.AddFunc(fmt.Sprintf("@every %ds", heartbeatIntervalSeconds), func() {
		mqttClient.publishHeartbeat(sensorSource, time.Now().UTC().Unix())
	})
	cronLib.Start()
}

func getLastStatus(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return strings.Trim(strings.TrimSpace(string(b)), "\n"), nil
}

func writeStatus(path, status string) error {
	return os.WriteFile(path, []byte(status), 0644)
}
