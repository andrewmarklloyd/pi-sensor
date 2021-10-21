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
	outletaddress := flag.String("outletaddress", os.Getenv("OUTLET_ADDRESS"), "The outlet to control")
	door := flag.String("door", os.Getenv("DOOR"), "The door to monitor")
	if *brokerurl == "" {
		logger.Fatalln("at least one broker is required")
	}
	if *outletaddress == "" {
		logger.Fatalln("outlet address is required")
	}
	if *door == "" {
		logger.Fatalln("door required")
	}

	_mqttClient := newMQTTClient(*brokerurl)
	outlet := hs100.NewHs100(*outletaddress, configuration.Default())

	_mqttClient.Subscribe(sensorStatusChannel, func(messageString string) {
		err := triggerOutlet(outlet, messageString, *door)
		if err != nil {
			logger.Println("Error triggering outlet:", err)
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
		return outlet.TurnOn()
	} else if strings.Contains(messageString, CLOSED) {
		return outlet.TurnOff()
	}
	return fmt.Errorf(fmt.Sprintf("Message did not contain %s or %s", OPEN, CLOSED))
}
