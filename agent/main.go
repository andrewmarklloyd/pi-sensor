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

	mqttC "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"

	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/config"
	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/gpio"
	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/mqtt"
	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/tailscale"
)

var (
	brokerurl     = flag.String("brokerurl", os.Getenv("CLOUDMQTT_URL"), "The MQTT broker to connect")
	agentUser     = flag.String("agentuser", os.Getenv("CLOUDMQTT_AGENT_USER"), "The MQTT agent user to connect")
	agentPassword = flag.String("agentpassword", os.Getenv("CLOUDMQTT_AGENT_PASSWORD"), "The MQTT agent password to connect")
	sensorSource  = flag.String("sensorSource", os.Getenv("SENSOR_SOURCE"), "The sensor location or name")
	mockFlag      = flag.String("mockMode", os.Getenv("MOCK_MODE"), "Mock mode for local development")
	version       = "unknown"
)

const (
	heartbeatIntervalSeconds = 60
)

func main() {
	l, err := zap.NewProduction()
	if err != nil {
		log.Fatalln("Error creating logger:", err)
	}
	// need a temporary init structured logger before reading sensorSource
	initLogger := l.Sugar().Named("pi-sensor-agent-init")
	defer initLogger.Sync()

	flag.Parse()
	if *sensorSource == "" {
		initLogger.Fatal("SENSOR_SOURCE env var is required")
	}

	logger := l.Sugar().Named(fmt.Sprintf("pi_sensor_agent-%s", *sensorSource))
	defer logger.Sync()

	logger.Infof("Initializing app, version: %s", version)

	if *brokerurl == "" {
		logger.Fatal("CLOUDMQTT_URL env var is required")
	}

	if *agentUser == "" {
		logger.Fatal("CLOUDMQTT_AGENT_USER environment variable not found")
	}

	if *agentPassword == "" {
		logger.Fatal("CLOUDMQTT_AGENT_PASSWORD environment variable not found")
	}

	urlSplit := strings.Split(*brokerurl, "@")
	if len(urlSplit) != 2 {
		logger.Fatal("unexpected CLOUDMQTT_URL parsing error")
	}
	domain := urlSplit[1]

	mqttAddr := fmt.Sprintf("mqtt://%s:%s@%s", *agentUser, *agentPassword, domain)

	mockMode, _ := strconv.ParseBool(*mockFlag)

	defaultPin := 18
	pinNum, err := strconv.Atoi(os.Getenv("GPIO_PIN"))
	if err != nil {
		logger.Infof("Failed to parse GPIO_PIN env var, using default %d", defaultPin)
		pinNum = defaultPin
	} else {
		logger.Infof("Using GPIO_PIN %d", pinNum)
	}

	mqttClient := mqtt.NewMQTTClient(mqttAddr, func(client mqttC.Client) {
		logger.Info("Connected to MQTT server")
	}, func(client mqttC.Client, err error) {
		logger.Errorf("Connection to MQTT server lost: %v", err)
	})
	err = mqttClient.Connect()
	if err != nil {
		logger.Fatalf("error connecting to mqtt: %s", err)
	}
	pinClient := gpio.NewPinClient(pinNum, mockMode)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		logger.Info("SIGTERM received, cleaning up")
		mqttClient.Cleanup()
		pinClient.Cleanup()
		os.Exit(0)
	}()

	h := config.Heartbeat{
		Name:    *sensorSource,
		Type:    config.HeartbeatTypeSensor,
		Version: version,
	}

	ticker := time.NewTicker(heartbeatIntervalSeconds * time.Second)
	go func() {
		for range ticker.C {
			err := mqttClient.PublishHeartbeat(h)
			if err != nil {
				logger.Errorf("error publishing heartbeat: %s", err)
			}
		}
	}()

	tailscaleStatusTicker := time.NewTicker(time.Hour)
	go func() {
		for range tailscaleStatusTicker.C {
			status, err := tailscale.CheckStatus()
			if err != nil {
				logger.Errorf("error checking tailscale status: %s", err)
			} else {
				if status.BackendState != "Running" || !status.Self.Online {
					logger.Errorf("Tailscale BackendState should be 'Running' and Self.Online should be true but values are: BackendState:%s, Self.Online:%t", status.BackendState)
				} else {
					logger.Infof("Tailscale status check results - BackendState:%s, Self.Online:%t", status.BackendState, status.Self.Online)
				}
			}
		}
	}()

	mqttClient.Subscribe(config.SensorRestartTopic, func(messageString string) {
		if *sensorSource == messageString {
			logger.Info("Received restart message, restarting app now")
			os.Exit(0)
		}
	})

	statusFile := getStatusFileName(*sensorSource)

	lastStatus, err := getLastStatus(statusFile)
	if err != nil {
		logger.Warnf("error reading status file: %s. Setting status to %s", err, config.UNKNOWN)
		lastStatus = config.UNKNOWN
	}

	var currentStatus string
	for {
		currentStatus = pinClient.CurrentStatus()
		err = writeStatus(statusFile, currentStatus)
		if err != nil {
			logger.Errorf("error writing status file: %s", err)
		}
		if currentStatus != lastStatus {
			logger.Infof(fmt.Sprintf("%s is %s", *sensorSource, currentStatus))
			lastStatus = currentStatus

			err := mqttClient.PublishSensorStatus(config.SensorStatus{
				Source:  *sensorSource,
				Status:  currentStatus,
				Version: version,
			})
			if err != nil {
				logger.Errorf("Error publishing message to sensor status channel: %s", err)
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

func getStatusFileName(sensorSource string) string {
	return fmt.Sprintf("/home/pi/.pi-sensor-status-%s", sensorSource)
}
