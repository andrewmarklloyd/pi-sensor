package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gofrs/uuid"
	"github.com/jaedle/golang-tplink-hs100/pkg/configuration"
	"github.com/jaedle/golang-tplink-hs100/pkg/hs100"
	"github.com/robfig/cron"
)

var (
	logger = log.New(os.Stdout, "[Pi-Sensor Outlet] ", log.LstdFlags)
)

const (
	sensorStatusChannel      = "sensor/status"
	sensorHeartbeatChannel   = "sensor/heartbeat"
	heartbeatIntervalSeconds = 60
	appSource                = "app_door-light"
	OPEN                     = "OPEN"
	CLOSED                   = "CLOSED"
)

func main() {
	appVersion := os.Getenv("APP_VERSION")
	if appVersion == "" {
		appVersion = "Unknown"
	}
	logger.Println(fmt.Sprintf("Running app version: %s", appVersion))
	brokerurl := flag.String("brokerurl", os.Getenv("CLOUDMQTT_URL"), "The broker to connect to")
	deviceNamesArg := flag.String("devicenames", os.Getenv("DOOR_LIGHT_DEVICE_NAMES"), "The devices to control as a comma separated list")
	door := flag.String("door", os.Getenv("DOOR_LIGHT_DOOR"), "The door to monitor")
	if *brokerurl == "" {
		logger.Fatalln("at least one broker is required")
	}

	deviceNames := strings.Split(*deviceNamesArg, ",")
	if len(deviceNames) == 0 {
		logger.Fatalln("devicenames required")
	}

	if *door == "" {
		logger.Fatalln("door required")
	}

	allDevices, err := hs100.Discover("192.168.1.1/24", configuration.Default().WithTimeout(time.Second))
	if err != nil {
		logger.Fatalln(fmt.Errorf("Error getting devices: %s", err))
	}

	var targetDevices []hs100.Hs100
	for _, d := range allDevices {
		name, err := d.GetName()
		if err != nil {
			logger.Fatalln(fmt.Errorf("Error getting device name: %s", err))
		}
		for _, n := range deviceNames {
			if name == n {
				targetDevices = append(targetDevices, *d)
			}
		}
	}

	if len(targetDevices) == 0 {
		logger.Fatalln(fmt.Sprintf("None of discovered devices matches expected device names: %s", deviceNames))
	}

	_mqttClient := newMQTTClient(*brokerurl)
	cronLib := cron.New()
	cronLib.AddFunc(fmt.Sprintf("@every %ds", heartbeatIntervalSeconds), func() {
		err := _mqttClient.publishHeartbeat(appSource, time.Now().UTC().Unix())
		if err != nil {
			logger.Println(err)
		}
	})
	cronLib.Start()

	_mqttClient.Subscribe(sensorStatusChannel, func(messageString string) {
		for _, d := range targetDevices {
			err := triggerOutlet(d, messageString, *door)
			if err != nil {
				logger.Println(fmt.Errorf("Error triggering outlet: %s", err))
			}
		}
	})

	go forever()
	select {} // block forever
}

type fn func(string)

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	logger.Println("Connected to MQTT server")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	logger.Fatalf("Connection to MQTT server lost: %v", err)
}

type mqttClient struct {
	client mqtt.Client
}

func newMQTTClient(brokerurl string) mqttClient {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(brokerurl)
	var clientID string
	u, _ := uuid.NewV4()
	clientID = u.String()
	logger.Println("Starting MQTT client with id", clientID)
	opts.SetClientID(clientID)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	return mqttClient{
		client,
	}
}

func (c mqttClient) Cleanup() {
	c.client.Disconnect(250)
}

func (c mqttClient) Subscribe(topic string, subscribeHandler fn) {
	var wg sync.WaitGroup
	wg.Add(1)

	if token := c.client.Subscribe(topic, 0, func(client mqtt.Client, msg mqtt.Message) {
		subscribeHandler(string(msg.Payload()))
	}); token.Wait() && token.Error() != nil {
		logger.Fatal(token.Error())
	}
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
	if strings.Contains(messageString, OPEN) {
		logger.Println(fmt.Sprintf("Turning %s outlet ON", name))
		return outlet.TurnOn()
	} else if strings.Contains(messageString, CLOSED) {
		logger.Println(fmt.Sprintf("Turning %s outlet OFF", name))
		return outlet.TurnOff()
	}
	return fmt.Errorf(fmt.Sprintf("Message did not contain %s or %s", OPEN, CLOSED))
}

func (c mqttClient) publishHeartbeat(sensorSource string, timestamp int64) error {
	ts := strconv.FormatInt(timestamp, 10)
	text := fmt.Sprintf("%s|%s", sensorSource, ts)
	token := c.client.Publish(sensorHeartbeatChannel, 0, false, text)
	token.Wait()
	return token.Error()
}