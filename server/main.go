package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
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

	_webServer = newWebServer(serverConfig, newClientHandler)

	mqttClient := newMQTTClient(serverConfig)
	mqttClient.Subscribe(func(messageString string) {
		message := toStruct(messageString)
		lastMessageString, _ := _redisClient.ReadState(message.Source)
		lastMessage := toStruct(lastMessageString)
		// TODO: add feature flag
		alertIfOpen(lastMessage, message)
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

func alertIfOpen(lastMessage Message, currentMessage Message) {
	logger.Println(lastMessage, currentMessage)
	if lastMessage.Status == "CLOSED" && currentMessage.Status == "OPEN" {
		logger.Println("Door was just opened")
	} else if lastMessage.Status == "OPEN" && currentMessage.Status == "CLOSED" {
		logger.Println("Door was just closed")
	} else {
		// intentionally do nothing
	}
}

// func configureOpenAlert(statusInterval int) {
// 	cronLib.AddFunc(fmt.Sprintf("@every %ds", statusInterval), func() {
// 		state, err := util.ReadState()
// 		if err != nil {
// 			log.Println(fmt.Sprintf("Error getting armed status: %s", err))
// 			return
// 		}
// 		if state.FirstReportedOpenTime != "" {
// 			firstReportedOpenTime, _ := time.Parse(time.RFC3339, state.FirstReportedOpenTime)
// 			now := time.Now()
// 			maxTimeSinceDoorOpened := now.Add(-maxDoorOpenedTime)
// 			if firstReportedOpenTime.Before(maxTimeSinceDoorOpened) && !state.AlertNotified {
// 				message := fmt.Sprintf("Door opened for longer than %s", maxDoorOpenedTime)
// 				if testMessageMode {
// 					log.Println(message)
// 				} else {
// 					messenger.SendMessage(message)
// 				}
// 				state.AlertNotified = true
// 				util.WriteState(state)
// 			}
// 		}
// 	})
// }
