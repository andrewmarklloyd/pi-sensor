package main

import (
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type fn func(string)

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	logger.Println("Connected")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	logger.Printf("Connect lost: %v", err)
}

type mqttClient struct {
	client mqtt.Client
	topic  string
}

func newMQTTClient(brokerurl string, topic string) mqttClient {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(brokerurl)
	opts.SetClientID("go_mqtt_sever")
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
	return mqttClient{
		client,
		topic,
	}
}

func (c mqttClient) Cleanup() {
	c.client.Disconnect(250)
}

func (c mqttClient) Subscribe(subscribeHandler fn) {
	var wg sync.WaitGroup
	wg.Add(1)

	if token := c.client.Subscribe(c.topic, 0, func(client mqtt.Client, msg mqtt.Message) {
		subscribeHandler(string(msg.Payload()))
	}); token.Wait() && token.Error() != nil {
		logger.Fatal(token.Error())
	}
}
