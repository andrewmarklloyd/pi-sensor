package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
)

var (
	brokerurl        = flag.String("brokerurl", os.Getenv("CLOUDMQTT_URL"), "The broker to connect to")
	topic            = flag.String("topic", os.Getenv("TOPIC"), "The topic to subscribe")
	redisurl         = flag.String("redisurl", os.Getenv("REDIS_URL"), "The redis cluster to connect to")
	mockFlag         = flag.String("mockMode", os.Getenv("MOCK_MODE"), "Mock mode for local development")
	port             = flag.String("port", os.Getenv("PORT"), "Port for the web server")
	authorizedusers  = flag.String("authorizedusers", os.Getenv("AUTHORIZED_USERS"), "")
	clientid         = flag.String("clientid", os.Getenv("GOOGLE_CLIENT_ID"), "")
	clientsecret     = flag.String("clientsecret", os.Getenv("GOOGLE_CLIENT_SECRET"), "")
	redirecturl      = flag.String("redirecturl", os.Getenv("REDIRECT_URL"), "")
	sessionsecret    = flag.String("sessionsecret", os.Getenv("SESSION_SECRET"), "")
	twilioaccountsid = flag.String("twilioaccountsid", os.Getenv("TWILIO_ACCOUNT_SID"), "")
	twilioauthtoken  = flag.String("twilioauthtoken", os.Getenv("TWILIO_AUTH_TOKEN"), "")
	twilioto         = flag.String("twilioto", os.Getenv("TWILIO_TO"), "")
	twiliofrom       = flag.String("twiliofrom", os.Getenv("TWILIO_FROM"), "")

	logger = log.New(os.Stdout, "[Pi-Sensor Server] ", log.LstdFlags)
)

const (
	sensorStatusChannel = "sensor/status"
	openTimeout         = 5 * time.Minute
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
	if *twilioaccountsid == "" {
		logger.Fatalln("twilioaccountsid must be set")
	}
	if *twilioauthtoken == "" {
		logger.Fatalln("twilioauthtoken must be set")
	}
	if *twilioto == "" {
		logger.Fatalln("twilioto must be set")
	}
	if *twiliofrom == "" {
		logger.Fatalln("twiliofrom must be set")
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
		twilioConfig: TwilioConfig{
			accountSID: *twilioaccountsid,
			authToken:  *twilioauthtoken,
			to:         *twilioto,
			from:       *twiliofrom,
		},
	}

	messenger := newMessenger(serverConfig.twilioConfig)
	var delayTimerMap map[string]*time.Timer = make(map[string]*time.Timer)
	_webServer = newWebServer(serverConfig, newClientHandler)
	mqttClient := newMQTTClient(serverConfig)
	mqttClient.Subscribe(func(messageString string) {
		message := toStruct(messageString)
		lastMessageString, _ := _redisClient.ReadState(message.Source)
		lastMessage := toStruct(lastMessageString)
		alertIfOpen(lastMessage, message, messenger)
		if message.Status == "OPEN" {
			timer := time.AfterFunc(openTimeout, func() {
				messenger.SendMessage(fmt.Sprintf("%s opened longer than %s", message.Source, openTimeout))
			})
			delayTimerMap[message.Source] = timer
		} else if message.Status == "CLOSED" {
			currentTimer := delayTimerMap[message.Source]
			if currentTimer != nil {
				currentTimer.Stop()
			}
		} else {
			logger.Println(fmt.Sprintf("Message status '%s' not recognized", message.Status))
		}

		_redisClient.WriteState(message.Source, messageString)
		_webServer.sendMessage(sensorStatusChannel, message)
		err := _redisClient.WriteState(message.Source, messageString)
		if err == nil {
			_webServer.sendMessage(sensorStatusChannel, message)
		} else {
			logger.Println(fmt.Errorf("Error writing state to Redis: %s", err))
		}
	})

	var err error
	_redisClient, err = newRedisClient(serverConfig.redisurl)
	if err != nil {
		logger.Fatalln("Error creating redis client:", err)
	}
	_webServer.startServer()
}

func alertIfOpen(lastMessage Message, currentMessage Message, messenger Messenger) {
	if lastMessage.Status == "CLOSED" && currentMessage.Status == "OPEN" {
		messenger.SendMessage(fmt.Sprintf("%s was just opened", currentMessage.Source))
	} else if lastMessage.Status == "OPEN" && currentMessage.Status == "CLOSED" {
		// intentionally do nothing
	} else {
		logger.Println(fmt.Sprintf("Door status was not changed from open to closed OR from closed to open. Last status: %s, current status: %s", lastMessage.Status, currentMessage.Status))
	}
}
