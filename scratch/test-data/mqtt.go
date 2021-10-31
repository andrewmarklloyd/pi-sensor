package main

import (
	"fmt"
	"strconv"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gofrs/uuid"
)

type fn func(string)

const (
	sensorHeartbeatChannel = "sensor/heartbeat"
	restartTopic           = "sensor/restart"
)

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	logger.Println("Connected to MQTT server")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	logger.Printf("Connection to MQTT server lost: %v", err)
}

type mqttClient struct {
	client mqtt.Client
	topic  string
}

func newMQTTClient(brokerurl string, topic string, sensorSource string) mqttClient {
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
		topic,
	}
}

func (c mqttClient) Cleanup() {
	c.client.Disconnect(250)
}

func (c mqttClient) publish(sensorSource string, currentStatus string, timestamp int64) {
	ts := strconv.FormatInt(timestamp, 10)
	text := fmt.Sprintf("%s|%s|%s", sensorSource, currentStatus, ts)
	token := c.client.Publish(c.topic, 0, false, text)
	token.Wait()
}
