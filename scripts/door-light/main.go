package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gofrs/uuid"
	"github.com/jaedle/golang-tplink-hs100/pkg/configuration"
	"github.com/jaedle/golang-tplink-hs100/pkg/hs100"
)

var (
	logger = log.New(os.Stdout, "[Pi-Sensor Outlet] ", log.LstdFlags)
)

const (
	sensorStatusChannel = "sensor/status"
	OPEN                = "OPEN"
	CLOSED              = "CLOSED"
)

// POC for turning on smart outlet when door is open
func main() {
	brokerurl := flag.String("brokerurl", os.Getenv("CLOUDMQTT_URL"), "The broker to connect to")
	deviceName := flag.String("devicename", os.Getenv("DEVICE_NAME"), "The outlet to control")
	door := flag.String("door", os.Getenv("DOOR"), "The door to monitor")
	if *brokerurl == "" {
		logger.Fatalln("at least one broker is required")
	}
	if *deviceName == "" {
		logger.Fatalln("outlet address is required")
	}
	if *door == "" {
		logger.Fatalln("door required")
	}

	devices, err := hs100.Discover("192.168.1.1/24", configuration.Default().WithTimeout(time.Second))
	if err != nil {
		logger.Println(fmt.Errorf("Error getting devices: %s", err))
	}

	var outlet *hs100.Hs100
	for _, d := range devices {
		name, err := d.GetName()
		if err != nil {
			logger.Println(fmt.Errorf("Error getting device name: %s", err))
		}
		if name == *deviceName {
			outlet = d
			break
		}
	}

	if outlet == nil {
		logger.Println(fmt.Sprintf("None of discovered devices matches expected device name %s: ", *deviceName), err)
	}

	_mqttClient := newMQTTClient(*brokerurl)
	_mqttClient.Subscribe(sensorStatusChannel, func(messageString string) {
		err := triggerOutlet(outlet, messageString, *door)
		if err != nil {
			logger.Println(fmt.Errorf("Error triggering outlet: %s", err))
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

func triggerOutlet(outlet *hs100.Hs100, messageString string, door string) error {
	if !strings.Contains(messageString, door) {
		return nil
	}
	if strings.Contains(messageString, OPEN) {
		fmt.Println("Turning outlet ON")
		return outlet.TurnOn()
	} else if strings.Contains(messageString, CLOSED) {
		fmt.Println("Turning outlet OFF")
		return outlet.TurnOff()
	}
	return fmt.Errorf(fmt.Sprintf("Message did not contain %s or %s", OPEN, CLOSED))
}
