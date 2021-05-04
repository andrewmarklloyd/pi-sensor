package main

import (
	"fmt"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

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
	opts.SetClientID("go_mqtt_client")
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

func (c mqttClient) publish(sensorSource string, currentStatus string) {
	text := fmt.Sprintf("%s:%s", sensorSource, currentStatus)
	token := c.client.Publish(c.topic, 0, false, text)
	token.Wait()
}
