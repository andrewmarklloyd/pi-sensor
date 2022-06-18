package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	mqttC "github.com/eclipse/paho.mqtt.golang"
	"go.uber.org/zap"

	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/config"
	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/mqtt"
	"github.com/jaedle/golang-tplink-hs100/pkg/configuration"
	"github.com/jaedle/golang-tplink-hs100/pkg/hs100"
)

var (
	logger *zap.SugaredLogger
)

const (
	heartbeatIntervalSeconds = 60
	appSource                = "door-light"
)

func main() {
	l, _ := zap.NewProduction()

	logger = l.Sugar().Named("pi-sensor-door-light")
	defer logger.Sync()

	appVersion := os.Getenv("APP_VERSION")
	if appVersion == "" {
		appVersion = "Unknown"
	}

	logger.Infof("Running app version: %s", appVersion)

	brokerurl := flag.String("brokerurl", os.Getenv("CLOUDMQTT_URL"), "The broker to connect to")
	appUser := flag.String("appUser", os.Getenv("CLOUDMQTT_APP_USER"), "The MQTT app user to connect")
	appUserPassword := flag.String("appUserPassword", os.Getenv("CLOUDMQTT_APP_PASSWORD"), "The MQTT app user password to connect")
	deviceNamesArg := flag.String("devicenames", os.Getenv("DOOR_LIGHT_DEVICE_NAMES"), "The devices to control as a comma separated list")
	door := flag.String("door", os.Getenv("DOOR_LIGHT_DOOR"), "The door to monitor")

	if *brokerurl == "" {
		logger.Fatal("at least one broker is required")
	}

	if *appUser == "" {
		logger.Fatal("appUser is required")
	}

	if *appUserPassword == "" {
		logger.Fatal("appUserPassword is required")
	}

	urlSplit := strings.Split(*brokerurl, "@")
	if len(urlSplit) != 2 {
		logger.Fatal("unexpected CLOUDMQTT_URL parsing error")
	}
	domain := urlSplit[1]

	mqttAddr := fmt.Sprintf("mqtt://%s:%s@%s", *appUser, *appUserPassword, domain)

	deviceNames := strings.Split(*deviceNamesArg, ",")
	if len(deviceNames) == 0 {
		logger.Fatal("devicenames required")
	}

	if *door == "" {
		logger.Fatal("door required")
	}

	allDevices, err := hs100.Discover("192.168.1.1/24", configuration.Default().WithTimeout(time.Second))
	if err != nil {
		logger.Fatalf("Error getting devices: %s", err)
	}

	var targetDevices []hs100.Hs100
	for _, d := range allDevices {
		name, err := d.GetName()
		if err != nil {
			logger.Fatalf("Error getting device name: %s", err)
		}
		for _, n := range deviceNames {
			if name == n {
				targetDevices = append(targetDevices, *d)
			}
		}
	}

	if len(targetDevices) == 0 {
		logger.Fatalf("None of discovered devices matches expected device names: %s", deviceNames)
	}

	h := config.Heartbeat{
		Name:    appSource,
		Type:    config.HeartbeatTypeApp,
		Version: appVersion,
	}

	mqttClient := mqtt.NewMQTTClient(mqttAddr, func(client mqttC.Client) {
		logger.Info("Connected to MQTT server")
	}, func(client mqttC.Client, err error) {
		logger.Fatalf("Connection to MQTT server lost: %v", err)
	})
	err = mqttClient.Connect()
	if err != nil {
		logger.Fatalf("error connecting to mqtt: %s", err)
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

	mqttClient.Subscribe(config.SensorStatusTopic, func(messageString string) {
		for _, d := range targetDevices {
			err := triggerOutlet(d, messageString, *door)
			if err != nil {
				logger.Errorf("Error triggering outlet: %s", err)
			}
		}
	})

	go forever()
	select {} // block forever
}

func forever() {
	for {
		time.Sleep(5 * time.Minute)
	}
}

func triggerOutlet(outlet hs100.Hs100, messageString string, door string) error {
	if !strings.Contains(messageString, door) {
		return nil
	}
	name, err := outlet.GetName()
	if err != nil {
		return fmt.Errorf("getting outlet name: %s", err)
	}
	if strings.Contains(messageString, config.OPEN) {
		logger.Infof("Turning %s outlet ON", name)
		return outlet.TurnOn()
	} else if strings.Contains(messageString, config.CLOSED) {
		logger.Infof("Turning %s outlet OFF", name)
		return outlet.TurnOff()
	}
	return fmt.Errorf(fmt.Sprintf("Message did not contain %s or %s", config.OPEN, config.CLOSED))
}
