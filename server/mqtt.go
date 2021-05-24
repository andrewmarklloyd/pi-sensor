package main

import (
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gofrs/uuid"
)

type fn func(string)

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	logger.Println("Connected")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	logger.Printf("Connect lost: %v", err)
}

type mqttClient struct {
	client   mqtt.Client
	mockMode bool
}

func newMQTTClient(serverConfig ServerConfig) mqttClient {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(serverConfig.brokerurl)
	var clientID string
	if serverConfig.mockMode {
		u, _ := uuid.NewV4()
		clientID = u.String()
	} else {
		clientID = "server"
	}
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
		serverConfig.mockMode,
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
