package mqtt

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/config"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gofrs/uuid"
)

type fn func(string)

type MqttClient struct {
	client mqtt.Client
}

func NewMQTTClient(addr string, connectHandler func(client mqtt.Client), connectionLostHandler func(client mqtt.Client, err error)) MqttClient {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(addr)
	var clientID string
	u, _ := uuid.NewV4()
	clientID = u.String()
	opts.SetClientID(clientID)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectionLostHandler
	client := mqtt.NewClient(opts)

	return MqttClient{
		client,
	}
}

func (c MqttClient) Connect() error {
	if token := c.client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}

func (c MqttClient) Cleanup() {
	c.client.Disconnect(250)
}

func (c MqttClient) Subscribe(topic string, subscribeHandler fn) error {
	var wg sync.WaitGroup
	wg.Add(1)

	if token := c.client.Subscribe(topic, 0, func(client mqtt.Client, msg mqtt.Message) {
		subscribeHandler(string(msg.Payload()))
	}); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}

func (c MqttClient) publish(topic, message string) error {
	token := c.client.Publish(topic, 0, false, message)
	token.Wait()
	return token.Error()
}

func (c MqttClient) PublishHeartbeat(h config.Heartbeat) error {
	j, err := json.Marshal(h)
	if err != nil {
		return fmt.Errorf("marshalling heartbeat: %s", err)
	}
	return c.publish(config.SensorHeartbeatTopic, string(j))
}

func (c MqttClient) PublishSensorStatus(h config.SensorStatus) error {
	j, err := json.Marshal(h)
	if err != nil {
		return fmt.Errorf("marshalling sensor status: %s", err)
	}
	return c.publish(config.SensorStatusTopic, string(j))
}

func (c MqttClient) PublishSensorRestart(sensorSource string) error {
	return c.publish(config.SensorRestartTopic, sensorSource)
}

func (c MqttClient) PublishHASensorStatus(h config.SensorStatus) error {
	j, err := json.Marshal(h)
	if err != nil {
		return fmt.Errorf("marshalling sensor status: %s", err)
	}
	return c.publish(config.HASensorStatusTopic, string(j))
}

func (c MqttClient) PublishHASensorArming(p config.APIPayload) error {
	j, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("marshalling api payload: %s", err)
	}
	return c.publish(config.HASensorArmingTopic, string(j))
}
