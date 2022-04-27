package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var version = ""

var (
	brokerurl        = flag.String("brokerurl", os.Getenv("CLOUDMQTT_URL"), "The broker to connect to")
	redisurl         = flag.String("redisurl", os.Getenv("REDIS_URL"), "The redis cluster to connect to")
	postgresurl      = flag.String("postgresurl", os.Getenv("DATABASE_URL"), "The postresql cluster to connect to")
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

	logger          = log.New(os.Stdout, "[Pi-Sensor Server] ", log.LstdFlags)
	forwarderLogger = log.New(os.Stdout, "[Log Forwarder] ", log.LstdFlags)

	mockMode bool
)

const (
	sensorStatusChannel    = "sensor/status"
	sensorHeartbeatChannel = "sensor/heartbeat"
	logForwarderChannel    = "logs/submit"
	openTimeout            = 5 * time.Minute
	heartbeatTimeout       = 5 * time.Minute
)

type LogMessage struct {
	Message string `json:"message"`
}

var _webServer webServer
var _redisClient redisClient
var _mqttClient mqttClient
var _postgresClient postgresClient

func newClientHandler() {
	state, stateErr := _redisClient.ReadAllState()
	armingState, armingStateErr := _redisClient.ReadAllArming()
	if stateErr != nil && armingStateErr != nil {
		logger.Println(fmt.Sprintf("Error getting state or arming state from redis: %s, %s", stateErr, armingStateErr))
	} else {
		_webServer.sendSensorState(state, armingState)
	}
}

type Sensor struct {
	Source string
}

func sensorRestartHandler(w http.ResponseWriter, req *http.Request) {
	var sensor Sensor
	err := json.NewDecoder(req.Body).Decode(&sensor)
	if err != nil {
		http.Error(w, "Error parsing request", http.StatusBadRequest)
		return
	}
	_mqttClient.publishSensorRestart(sensor.Source)
	logger.Println(fmt.Sprintf("Publishing sensor restart message for %s", sensor.Source))
	fmt.Fprintf(w, "{\"status\":\"success\"}")
}

func sensorArmingHandler(w http.ResponseWriter, req *http.Request) {
	var sensor Sensor
	err := json.NewDecoder(req.Body).Decode(&sensor)
	if err != nil {
		http.Error(w, "Error parsing request", http.StatusBadRequest)
		return
	}
	armedString, _ := _redisClient.ReadArming(sensor.Source)
	armed := "false"
	if armedString == "" || armedString == "false" {
		armed = "true"
	}
	_redisClient.WriteArming(sensor.Source, armed)
	fmt.Fprintf(w, fmt.Sprintf(`{"status":"success", "armed":"%s"}`, armed))
}

func reportHandler(w http.ResponseWriter, req *http.Request) {
	query := req.URL.Query()
	sensor := query.Get("sensor")
	pageString := query.Get("page")
	page, err := strconv.Atoi(pageString)
	if err != nil {
		http.Error(w, "Page not found", http.StatusBadRequest)
		return
	}
	if sensor == "" {
		http.Error(w, "Pass sensor in request", http.StatusBadRequest)
		return
	}
	messages, numPages, err := _postgresClient.getSensorStatus(sensor, page)
	if err != nil {
		logger.Fatalln("Error getting messages", err)
		http.Error(w, "Error getting report", http.StatusBadRequest)
		return
	}
	json, _ := json.Marshal(messages)
	fmt.Fprintf(w, fmt.Sprintf(`{"messages":%s,"numPages":%d}`, string(json), numPages))
}

func allSensorsHandler(w http.ResponseWriter, req *http.Request) {
	sensors, err := _redisClient.GetAllSensors()
	if err != nil {
		logger.Fatalln("Error getting all keys", err)
		http.Error(w, "Error getting sensors", http.StatusBadRequest)
		return
	}
	json, _ := json.Marshal(sensors)
	fmt.Fprintf(w, fmt.Sprintf(`{"sensors":%s}`, string(json)))
}

func main() {
	logger.Println("Initializing server, version", version)
	flag.Parse()

	if *brokerurl == "" {
		logger.Fatalln("at least one broker is required")
	}
	if *redisurl == "" {
		logger.Fatalln("redisurl is required")
	}
	if *postgresurl == "" {
		logger.Fatalln("postgresurl is required")
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
		brokerurl:   *brokerurl,
		redisurl:    *redisurl,
		postgresurl: *postgresurl,
		port:        *port,
		mockMode:    mockMode,
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

	_postgresClient, err = newPostgresClient(serverConfig.postgresurl)
	if err != nil {
		logger.Fatalln("Error creating postgres client:", err)
	}

	messenger := newMessenger(serverConfig.twilioConfig)
	var delayTimerMap map[string]*time.Timer = make(map[string]*time.Timer)
	_webServer = newWebServer(serverConfig, newClientHandler, sensorRestartHandler, sensorArmingHandler, reportHandler, allSensorsHandler)
	_mqttClient = newMQTTClient(serverConfig)
	_mqttClient.Subscribe(sensorStatusChannel, func(messageString string) {
		message := toStruct(messageString)
		lastMessageString, _ := _redisClient.ReadState(message.Source)
		lastMessage := toStruct(lastMessageString)
		armedString, _ := _redisClient.ReadArming(message.Source)
		armed := true
		if armedString == "" || armedString == "false" {
			armed = false
		}
		alertIfOpen(lastMessage, message, messenger, armed)
		if message.Status == "OPEN" {
			timer := time.AfterFunc(openTimeout, newOpenTimeoutFunc(message, messenger, armed))
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
			writeErr := _postgresClient.writeSensorStatus(message)
			if writeErr != nil {
				logger.Println(writeErr)
			}
		} else {
			logger.Println(fmt.Errorf("Error writing state to Redis: %s", err))
		}
	})

	_mqttClient.Subscribe(logForwarderChannel, func(messageString string) {
		var logMessage LogMessage
		err := json.Unmarshal([]byte(messageString), &logMessage)
		if err != nil {
			logger.Println(err)
		} else {
			forwarderLogger.Println(logMessage.Message)
		}
	})

	var heartbeatTimerMap map[string]*time.Timer = make(map[string]*time.Timer)
	_mqttClient.Subscribe(sensorHeartbeatChannel, func(messageString string) {
		heartbeat := toHeartbeat(messageString)
		currentTimer := heartbeatTimerMap[heartbeat.Source]
		if currentTimer != nil {
			// TODO: investigate Reset instead of creating new timers
			currentTimer.Stop()
		}

		var timer *time.Timer
		if isAppHeartbeat(heartbeat) {
			timer = time.AfterFunc(heartbeatTimeout, newAppHeartbeatTimeoutFunc(heartbeat, messenger))
		} else {
			timer = time.AfterFunc(heartbeatTimeout, newHeartbeatTimeoutFunc(heartbeat, messenger))
		}

		heartbeatTimerMap[heartbeat.Source] = timer
	})

	ticker := time.NewTicker(6 * time.Hour)
	go func() {
		for range ticker.C {
			err = messenger.CheckBalance()
			if err != nil {
				logger.Println(err)
			}
		}
	}()

	_webServer.startServer()
}

func newAppHeartbeatTimeoutFunc(h Heartbeat, msgr Messenger) func() {
	return func() {
		handleAppHeartbeatTimeout(h, msgr)
	}
}

func handleAppHeartbeatTimeout(h Heartbeat, msgr Messenger) {
	logger.Println(fmt.Sprintf("Heartbeat timeout occurred for %s", h.Source))
	if !mockMode {
		_, err := msgr.SendMessage(fmt.Sprintf("%s has lost connection", h.Source))
		if err != nil {
			fmt.Println("Error sending app heartbeat timeout message:", err)
		}
	}
}

func newHeartbeatTimeoutFunc(h Heartbeat, msgr Messenger) func() {
	return func() {
		handleHeartbeatTimeout(h, msgr)
	}
}

func handleHeartbeatTimeout(h Heartbeat, msgr Messenger) {
	messageString, err := _redisClient.ReadState(h.Source)
	if err == nil {
		message := toStruct(messageString)
		message.Status = UNKNOWN
		err := _redisClient.WriteState(message.Source, toString(message))
		if err != nil {
			logger.Println(fmt.Sprintf("Error writing message state after heartbeat timeout. Message: %s", messageString))
		} else {
			logger.Println(fmt.Sprintf("Heartbeat timeout occurred for %s", h.Source))
			if !mockMode {
				msgr.SendMessage(fmt.Sprintf("%s sensor has lost connection", h.Source))
			}
			_webServer.sendMessage(sensorStatusChannel, message)

			m := Message{
				Source:    message.Source,
				Timestamp: strconv.FormatInt(time.Now().UTC().Unix(), 10),
				Status:    UNKNOWN,
			}
			writeErr := _postgresClient.writeSensorStatus(m)
			if writeErr != nil {
				logger.Println(writeErr)
			}
		}
	} else {
		logger.Println(err)
	}
}

func isAppHeartbeat(h Heartbeat) bool {
	return strings.Contains(h.Source, "app_")
}

func newOpenTimeoutFunc(m Message, msgr Messenger, armed bool) func() {
	return func() {
		handleOpenTimeout(m, msgr, armed)
	}
}

func handleOpenTimeout(m Message, msgr Messenger, armed bool) {
	message := fmt.Sprintf("🚨 %s opened longer than %s", m.Source, openTimeout)
	logger.Println(message)
	if !mockMode && armed {
		msgr.SendMessage(message)
	}
}

func alertIfOpen(lastMessage Message, currentMessage Message, messenger Messenger, armed bool) {
	if (lastMessage.Status == CLOSED && currentMessage.Status == OPEN) || (lastMessage.Status == UNKNOWN && currentMessage.Status == OPEN) {
		if !mockMode && armed {
			_, err := messenger.SendMessage(fmt.Sprintf("🚪 %s was just opened", currentMessage.Source))
			if err != nil {
				logger.Println("Error sending open message", err)
			}
		} else {
			logger.Println(fmt.Sprintf("%s was just opened", currentMessage.Source))
		}
	} else if lastMessage.Status == "OPEN" && currentMessage.Status == "CLOSED" {
		// intentionally do nothing
	} else {
		logger.Println(fmt.Sprintf("Door status change was not recognized. Last status: %s, current status: %s", lastMessage.Status, currentMessage.Status))
	}
}
