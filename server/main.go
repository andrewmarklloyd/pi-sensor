package main

import (
	"flag"
	"log"
	"os"
	"strconv"
)

var (
	brokerurl       = flag.String("brokerurl", os.Getenv("CLOUDMQTT_URL"), "The broker to connect to")
	topic           = flag.String("topic", os.Getenv("TOPIC"), "The topic to subscribe")
	redisurl        = flag.String("redisurl", os.Getenv("REDIS_URL"), "The redis cluster to connect to")
	mockFlag        = flag.String("mockMode", os.Getenv("MOCK_MODE"), "Mock mode for local development")
	port            = flag.String("port", os.Getenv("PORT"), "Port for the web server")
	authorizedusers = flag.String("authorizedusers", os.Getenv("AUTHORIZED_USERS"), "")
	clientid        = flag.String("clientid", os.Getenv("GOOGLE_CLIENT_ID"), "")
	clientsecret    = flag.String("clientsecret", os.Getenv("GOOGLE_CLIENT_SECRET"), "")
	redirecturl     = flag.String("redirecturl", os.Getenv("REDIRECT_URL"), "")
	sessionsecret   = flag.String("sessionsecret", os.Getenv("SESSION_SECRET"), "")

	logger = log.New(os.Stdout, "[Pi-Sensor Server] ", log.LstdFlags)
)

const (
	sensorStatusChannel = "sensor/status"
)

var _webServer webServer
var _redisClient redisClient

func newClientHandler() {
	state, err := _redisClient.ReadAllState()
	if err != nil {
		logger.Println("Error getting state from redis:", err)
	} else {
		_webServer.sendSensorList(state)
	}
}

func main() {
	logger.Println("Initializing server")
	flag.Parse()

	if *brokerurl == "" {
		logger.Fatalln("at least one broker is required")
	}
	if *topic == "" {
		logger.Fatalln("topic to publish to is required")
	}
	if *redisurl == "" {
		logger.Fatalln("redisurl is required")
	}
	if *port == "" {
		logger.Fatalln("PORT must be set")
	}
	if *authorizedusers == "" {
		logger.Fatalln("authorizedusers must be set")
	}
	if *clientid == "" {
		logger.Fatalln("clientid must be set")
	}
	if *clientsecret == "" {
		logger.Fatalln("clientsecret must be set")
	}
	if *redirecturl == "" {
		logger.Fatalln("redirecturl must be set")
	}
	if *sessionsecret == "" {
		logger.Fatalln("sessionsecret must be set")
	}
	mockMode, _ := strconv.ParseBool(*mockFlag)

	serverConfig := ServerConfig{
		brokerurl: *brokerurl,
		topic:     *topic,
		redisurl:  *redisurl,
		port:      *port,
		mockMode:  mockMode,
		googleConfig: GoogleConfig{
			authorizedUsers: *authorizedusers,
			clientId:        *clientid,
			clientSecret:    *clientsecret,
			redirectUrl:     *redirecturl,
			sessionSecret:   *sessionsecret,
		},
	}

	_webServer = newWebServer(serverConfig, newClientHandler)

	mqttClient := newMQTTClient(serverConfig)
	mqttClient.Subscribe(func(message string) {
		messageStruct := toStruct(message)
		_redisClient.WriteState(messageStruct.Source, messageStruct.Status)
		_webServer.sendMessage(sensorStatusChannel, message)
	})

	var err error
	_redisClient, err = newRedisClient(serverConfig.redisurl)
	if err != nil {
		logger.Fatalln("Error creating redis client:", err)
	}
	_webServer.startServer()
}
