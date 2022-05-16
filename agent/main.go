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

	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/config"
	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/gpio"
	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/mqtt"
)

var (
	brokerurl     = flag.String("brokerurl", os.Getenv("CLOUDMQTT_URL"), "The MQTT broker to connect")
	agentUser     = flag.String("agentuser", os.Getenv("CLOUDMQTT_AGENT_USER"), "The MQTT agent user to connect")
	agentPassword = flag.String("agentpassword", os.Getenv("CLOUDMQTT_AGENT_PASSWORD"), "The MQTT agent password to connect")
	sensorSource  = flag.String("sensorSource", os.Getenv("SENSOR_SOURCE"), "The sensor location or name")
	mockFlag      = flag.String("mockMode", os.Getenv("MOCK_MODE"), "Mock mode for local development")
	logger        *log.Logger
)

const (
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

	if *agentUser == "" {
		logger.Fatalln("CLOUDMQTT_AGENT_USER environment variable not found")
	}

	if *agentPassword == "" {
		logger.Fatalln("CLOUDMQTT_AGENT_PASSWORD environment variable not found")
	}

	urlSplit := strings.Split(*brokerurl, "@")
	if len(urlSplit) != 2 {
		logger.Fatalln("unexpected CLOUDMQTT_URL parsing error")
	}
	domain := urlSplit[1]

	mqttAddr := fmt.Sprintf("mqtt://%s:%s@%s", *agentUser, *agentPassword, domain)

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

	mqttClient := mqtt.NewMQTTClient(mqttAddr, logger)
	err = mqttClient.Connect()
	if err != nil {
		logger.Fatalln("error connecting to mqtt:", err)
	}
	pinClient := gpio.NewPinClient(pinNum, mockMode)

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		logger.Println("Cleaning up")
		mqttClient.Cleanup()
		pinClient.Cleanup()
		os.Exit(0)
	}()

	h := config.Heartbeat{
		Name: *sensorSource,
		Type: config.HeartbeatTypeSensor,
	}

	ticker := time.NewTicker(heartbeatIntervalSeconds * time.Second)
	go func() {
		for range ticker.C {
			mqttClient.PublishHeartbeat(h)
		}
	}()

	mqttClient.Subscribe(config.SensorRestartTopic, func(messageString string) {
		if *sensorSource == messageString {
			logger.Println("Received restart message, restarting app now")
			os.Exit(0)
		}
	})

	lastStatus, err := getLastStatus(statusFile)
	if err != nil {
		logger.Println(fmt.Errorf("error reading status file: %s. Setting status to %s", err, config.UNKNOWN))
		lastStatus = config.UNKNOWN
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

			err := mqttClient.PublishSensorStatus(config.SensorStatus{
				Source: *sensorSource,
				Status: currentStatus,
			})
			if err != nil {
				logger.Println("Error publishing message to sensor status channel:", err)
			}
		}
		time.Sleep(5 * time.Second)
	}
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
