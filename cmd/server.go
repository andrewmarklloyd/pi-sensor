package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/clients"
	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/config"
	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/mqtt"
	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/notification"
	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/postgres"
	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/redis"

	"github.com/spf13/viper"
)

var (
	logger          = log.New(os.Stdout, "[Pi-Sensor Server] ", log.LstdFlags)
	forwarderLogger = log.New(os.Stdout, "[Log Forwarder] ", log.LstdFlags)
	version         = "unknown"
)

func runServer() {
	logger.Println("Running server version:", version)
	serverConfig := config.ServerConfig{
		MqttBrokerURL:      viper.GetString("CLOUDMQTT_URL"),
		MqttServerUser:     viper.GetString("CLOUDMQTT_SERVER_USER"),
		MqttServerPassword: viper.GetString("CLOUDMQTT_SERVER_PASSWORD"),
		RedisURL:           viper.GetString("REDIS_URL"),
		PostgresURL:        viper.GetString("DATABASE_URL"),
		Port:               viper.GetString("PORT"),
		MockMode:           viper.GetBool("MOCK_MODE"),
		GoogleConfig: config.GoogleConfig{
			AuthorizedUsers: viper.GetString("AUTHORIZED_USERS"),
			ClientId:        viper.GetString("GOOGLE_CLIENT_ID"),
			ClientSecret:    viper.GetString("GOOGLE_CLIENT_SECRET"),
			RedirectURL:     viper.GetString("REDIRECT_URL"),
			SessionSecret:   viper.GetString("SESSION_SECRET"),
		},
		TwilioConfig: config.TwilioConfig{
			AccountSID: viper.GetString("TWILIO_ACCOUNT_SID"),
			AuthToken:  viper.GetString("TWILIO_AUTH_TOKEN"),
			To:         viper.GetString("TWILIO_TO"),
			From:       viper.GetString("TWILIO_FROM"),
		},
	}

	serverClients, err := createClients(serverConfig)
	if err != nil {
		logger.Fatalln("Error creating clients:", err)
	}

	err = serverClients.Mqtt.Connect()
	if err != nil {
		logger.Fatalln("error connecting to mqtt:", err)
	}

	webServer := newWebServer(serverConfig, serverClients)

	var delayTimerMap map[string]*time.Timer = make(map[string]*time.Timer)
	serverClients.Mqtt.Subscribe(config.SensorStatusTopic, func(message string) {
		err := handleSensorStatusSubscribe(serverClients, webServer, serverConfig, message, delayTimerMap)
		if err != nil {
			logger.Println(err)
		}
	})

	var heartbeatTimerMap map[string]*time.Timer = make(map[string]*time.Timer)
	serverClients.Mqtt.Subscribe(config.SensorHeartbeatTopic, func(messageString string) {
		var h config.Heartbeat
		err := json.Unmarshal([]byte(messageString), &h)
		if err != nil {
			logger.Println(fmt.Errorf("error unmarshalling message from heartbeat channel: %s. Message received was: %s", err, messageString))
			return
		}
		currentTimer := heartbeatTimerMap[h.Name]
		if currentTimer != nil {
			currentTimer.Stop()
		}

		timer := time.AfterFunc(config.HeartbeatTimeout, func() { handleAppHeartbeatTimeout(h, serverClients.Messenger) })
		heartbeatTimerMap[h.Name] = timer
	})

	ticker := time.NewTicker(6 * time.Hour)
	go func() {
		for range ticker.C {
			if err := serverClients.Messenger.CheckBalance(); err != nil {
				logger.Println(err)
			}
		}
	}()

	err = webServer.httpServer.ListenAndServe()
	if err != nil {
		logger.Fatalln("Error starting web server:", err)
	}
}

func createClients(serverConfig config.ServerConfig) (clients.ServerClients, error) {
	redisClient, err := redis.NewRedisClient(serverConfig.RedisURL)
	if err != nil {
		return clients.ServerClients{}, fmt.Errorf("Error creating redis client: %s", err)
	}

	postgresClient, err := postgres.NewPostgresClient(serverConfig.PostgresURL)
	if err != nil {
		return clients.ServerClients{}, fmt.Errorf("Error creating postgres client: %s", err)
	}

	urlSplit := strings.Split(serverConfig.MqttBrokerURL, "@")
	if len(urlSplit) != 2 {
		return clients.ServerClients{}, fmt.Errorf("unexpected CLOUDMQTT_URL parsing error")
	}
	domain := urlSplit[1]
	mqttAddr := fmt.Sprintf("mqtt://%s:%s@%s", serverConfig.MqttServerUser, serverConfig.MqttServerPassword, domain)

	mqttClient := mqtt.NewMQTTClient(mqttAddr, logger)

	messenger := notification.NewMessenger(serverConfig.TwilioConfig)

	return clients.ServerClients{
		Redis:     redisClient,
		Postgres:  postgresClient,
		Mqtt:      mqttClient,
		Messenger: messenger,
	}, nil
}

func handleAppHeartbeatTimeout(h config.Heartbeat, msgr notification.Messenger) {

}

func handleSensorStatusSubscribe(serverClients clients.ServerClients, webServer WebServer, serverConfig config.ServerConfig, message string, delayTimerMap map[string]*time.Timer) error {
	currentStatus := config.SensorStatus{}
	err := json.Unmarshal([]byte(message), &currentStatus)
	if err != nil {
		return fmt.Errorf("unmarshalling message: %s", err)
	}

	currentStatus.Timestamp = strconv.FormatInt(time.Now().UTC().Unix(), 10)
	currentStatusJson, err := json.Marshal(currentStatus)
	if err != nil {
		return err
	}

	lastMessageString, err := serverClients.Redis.ReadState(currentStatus.Source, context.Background())
	if err != nil && err.Error() != "redis: nil" {
		return fmt.Errorf("reading state from redis: %s", err)
	}

	if lastMessageString == "" {
		// first time sensor sent status
		err = serverClients.Redis.WriteState(currentStatus.Source, string(currentStatusJson), context.Background())
		if err != nil {
			return fmt.Errorf("writing state to redis: %s", err)
		}
		return nil
	}

	lastStatus := config.SensorStatus{}
	err = json.Unmarshal([]byte(lastMessageString), &lastStatus)
	if err != nil {
		return fmt.Errorf("unmarshalling last status: %s", err)
	}

	armedString, err := serverClients.Redis.ReadArming(currentStatus.Source, context.Background())
	armed := true
	if armedString == "" || armedString == "false" {
		armed = false
	}

	if (lastStatus.Status == config.CLOSED && currentStatus.Status == config.OPEN) || (lastStatus.Status == config.UNKNOWN && currentStatus.Status == config.OPEN) {
		logger.Println(fmt.Sprintf("%s was just opened", currentStatus.Source))
		if !serverConfig.MockMode && armed {
			_, err := serverClients.Messenger.SendMessage(fmt.Sprintf("🚪 %s was just opened", currentStatus.Source))
			if err != nil {
				return fmt.Errorf("Error sending open message: %s", err)
			}
		}
	}

	if currentStatus.Status == config.OPEN {
		timer := time.AfterFunc(config.OpenTimeout, func() {
			handleOpenTimeout(serverClients, currentStatus, armed, serverConfig.MockMode)
		})
		delayTimerMap[currentStatus.Source] = timer
	} else if currentStatus.Status == config.CLOSED {
		currentTimer := delayTimerMap[currentStatus.Source]
		if currentTimer != nil {
			currentTimer.Stop()
		}
	} else {
		logger.Println(fmt.Sprintf("Message status '%s' not recognized", currentStatus.Status))
	}

	err = serverClients.Redis.WriteState(currentStatus.Source, string(currentStatusJson), context.Background())
	if err != nil {
		return fmt.Errorf("Error writing state to Redis: %s", err)
	}

	webServer.SendMessage(config.SensorStatusTopic, currentStatus)
	writeErr := serverClients.Postgres.WriteSensorStatus(currentStatus)
	if writeErr != nil {
		return fmt.Errorf("writing sensor status to postgres: %s", writeErr)
	}
	return nil
}

func handleOpenTimeout(serverClients clients.ServerClients, s config.SensorStatus, armed, mockMode bool) {
	message := fmt.Sprintf("🚨 %s opened longer than %s", s.Source, config.OpenTimeout)
	logger.Println(message)
	if !mockMode && armed {
		serverClients.Messenger.SendMessage(message)
	}
}
