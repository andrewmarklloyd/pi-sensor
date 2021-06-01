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

	mockMode bool
)

const (
	sensorStatusChannel    = "sensor/status"
	sensorHeartbeatChannel = "sensor/heartbeat"
	openTimeout            = 5 * time.Minute
	heartbeatTimeout       = 15 * time.Second
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

	mockMode, _ = strconv.ParseBool(*mockFlag)

	serverConfig := ServerConfig{
		brokerurl: *brokerurl,
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

	var err error
	_redisClient, err = newRedisClient(serverConfig.redisurl)
	if err != nil {
		logger.Fatalln("Error creating redis client:", err)
	}

	messenger := newMessenger(serverConfig.twilioConfig)
	var delayTimerMap map[string]*time.Timer = make(map[string]*time.Timer)
	_webServer = newWebServer(serverConfig, newClientHandler)
	mqttClient := newMQTTClient(serverConfig)
	mqttClient.Subscribe(sensorStatusChannel, func(messageString string) {
		message := toStruct(messageString)
		lastMessageString, _ := _redisClient.ReadState(message.Source)
		lastMessage := toStruct(lastMessageString)
		alertIfOpen(lastMessage, message, messenger)
		if message.Status == "OPEN" {
			// TODO: use a returned parameterized function similar to heartbeat
			timer := time.AfterFunc(openTimeout, func() {
				message := fmt.Sprintf("%s opened longer than %s", message.Source, openTimeout)
				if mockMode {
					logger.Println(message)
				} else {
					messenger.SendMessage(message)
				}
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

		err := _redisClient.WriteState(message.Source, messageString)
		if err == nil {
			_webServer.sendMessage(sensorStatusChannel, message)
		} else {
			logger.Println(fmt.Errorf("Error writing state to Redis: %s", err))
		}
	})

	var heartbeatTimerMap map[string]*time.Timer = make(map[string]*time.Timer)
	mqttClient.Subscribe(sensorHeartbeatChannel, func(messageString string) {
		heartbeat := toHeartbeat(messageString)
		currentTimer := heartbeatTimerMap[heartbeat.Source]
		if currentTimer != nil {
			currentTimer.Stop()
		}
		timer := time.AfterFunc(heartbeatTimeout, newHeartbeatTimeoutFunc(heartbeat))
		heartbeatTimerMap[heartbeat.Source] = timer
	})

	_webServer.startServer()
}

func newHeartbeatTimeoutFunc(h Heartbeat) func() {
	return func() {
		handleHeartbeatTimeout(h)
	}
}

func handleHeartbeatTimeout(h Heartbeat) {
	logger.Println(fmt.Sprintf("Heartbeat timeout occurred for %s", h.Source))
	messageString, err := _redisClient.ReadState(h.Source)
	if err == nil {
		message := toStruct(messageString)
		message.Status = UNKNOWN
		err := _redisClient.WriteState(message.Source, toString(message))
		if err != nil {
			logger.Println(fmt.Sprintf("Error writing message state after heartbeat timeout. Message: %s", messageString))
		} else {
			_webServer.sendMessage(sensorStatusChannel, message)
		}
	} else {
		logger.Println(err)
	}
}

func alertIfOpen(lastMessage Message, currentMessage Message, messenger Messenger) {
	if (lastMessage.Status == CLOSED && currentMessage.Status == OPEN) || (lastMessage.Status == UNKNOWN && currentMessage.Status == OPEN) {
		if mockMode {
			logger.Println(fmt.Sprintf("%s was just opened", currentMessage.Source))
		} else {
			_, err := messenger.SendMessage(fmt.Sprintf("%s was just opened", currentMessage.Source))
			if err != nil {
				logger.Println("Error sending open message", err)
			}
		}
	} else if lastMessage.Status == "OPEN" && currentMessage.Status == "CLOSED" {
		// intentionally do nothing
	} else {
		logger.Println(fmt.Sprintf("Door status change was not recognized. Last status: %s, current status: %s", lastMessage.Status, currentMessage.Status))
	}
}
