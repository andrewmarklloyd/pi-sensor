package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gofrs/uuid"
)

var (
	logger = log.New(os.Stdout, "[Pi-Sensor Log Forwarder] ", log.LstdFlags)
)

const (
	logForwarderChannel      = "logs/submit"
	sensorHeartbeatChannel   = "sensor/heartbeat"
	heartbeatIntervalSeconds = 60
	appSource                = "app_log-forwarder"
)

func main() {
	brokerurl := flag.String("brokerurl", os.Getenv("CLOUDMQTT_URL"), "The broker to connect to")
	flag.Parse()
	if *brokerurl == "" {
		logger.Fatalln("at least one broker is required")
	}

	_mqttClient := newMQTTClient(*brokerurl)

	ch := make(chan string)
	// TODO: get from config
	go tailSystemdLogs("door-light", ch)

	for logs := range ch {
		logLines := strings.Split(strings.Replace(logs, "\n", `\n`, -1), `\n`)
		for _, line := range logLines {
			if line != "" {
				_mqttClient.sendLogs(line)
			}
		}
	}
}

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

func (c mqttClient) publishHeartbeat(sensorSource string, timestamp int64) {
	ts := strconv.FormatInt(timestamp, 10)
	text := fmt.Sprintf("%s|%s", sensorSource, ts)
	token := c.client.Publish(sensorHeartbeatChannel, 0, false, text)
	token.Wait()
}

func (c mqttClient) sendLogs(logs string) {
	text := fmt.Sprintf("{\"message\":\"%s\"}", strings.Replace(logs, "\n", `\n`, -1))
	token := c.client.Publish(logForwarderChannel, 0, false, text)
	token.Wait()
}

func tailSystemdLogs(systemdUnit string, ch chan string) {
	cmd := exec.Command("tail", "-f", "-n0", "test.txt")
	// cmd := exec.Command("journalctl", "-u", systemdUnit, "-f", "-n 0")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	buf := make([]byte, 1024)
	for {
		n, err := stdout.Read(buf)
		if err != nil {
			break
		}

		ch <- string(buf[0:n])
	}
	close(ch)
	if err := cmd.Wait(); err != nil {
		log.Fatal(err)
	}
}
