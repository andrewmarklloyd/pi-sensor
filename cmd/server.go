package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/aws"
	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/clients"
	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/config"
	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/crypto"
	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/datadog"
	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/mqtt"
	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/postgres"
	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/redis"
	mqttC "github.com/eclipse/paho.mqtt.golang"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var (
	logger           *zap.SugaredLogger
	forwarderLogger  *zap.SugaredLogger
	version          = "unknown"
	sensorConfigChan = make(chan config.SensorConfig, 1)
	sensorConfigMap  = make(map[string]int32)
)

const (
	dataRetentionCronFrequency = 12 * time.Hour
	fullBackupCronFrequency    = 6 * time.Hour
	tokenExpMetricFreq         = 10 * time.Minute
)

func runServer() {
	l, _ := zap.NewProduction()
	logger = l.Sugar().Named("pi_sensor_server")
	defer logger.Sync()
	logger.Infof("Running server version: %s", version)

	forwarderLogger = l.Sugar().Named("pi_sensor_agent")
	defer forwarderLogger.Sync()

	serverConfig := config.ServerConfig{
		AppName:                 viper.GetString("APP_NAME"),
		MosquittoServerDomain:   viper.GetString("MOSQUITTO_DOMAIN"),
		MosquittoServerUser:     viper.GetString("MOSQUITTO_SERVER_USER"),
		MosquittoServerPassword: viper.GetString("MOSQUITTO_SERVER_PASSWORD"),
		RedisURL:                viper.GetString("REDIS_URL"),
		RedisTLSURL:             viper.GetString("REDIS_TLS_URL"),
		PostgresURL:             viper.GetString("DATABASE_URL"),
		Port:                    viper.GetString("PORT"),
		MockMode:                viper.GetBool("MOCK_MODE"),
		AllowedAPIKeys:          viper.GetStringSlice("ALLOWED_API_KEYS"),
		GoogleConfig: config.GoogleConfig{
			AuthorizedUsers: viper.GetString("AUTHORIZED_USERS"),
			ClientId:        viper.GetString("GOOGLE_CLIENT_ID"),
			ClientSecret:    viper.GetString("GOOGLE_CLIENT_SECRET"),
			RedirectURL:     viper.GetString("REDIRECT_URL"),
			SessionSecret:   viper.GetString("SESSION_SECRET"),
		},
		DatadogConfig: config.DatadogConfig{
			APIKey:         viper.GetString("DD_API_KEY"),
			APPKey:         viper.GetString("DD_APP_KEY"),
			TokensMetadata: buildTokenMetadata(),
		},
		S3Config: config.S3Config{
			AccessKeyID:       viper.GetString("SPACES_AWS_ACCESS_KEY_ID"),
			SecretAccessKey:   viper.GetString("SPACES_AWS_SECRET_ACCESS_KEY"),
			Region:            viper.GetString("SPACES_AWS_REGION"),
			URL:               viper.GetString("SPACES_URL"),
			Bucket:            viper.GetString("SPACES_BUCKET_NAME"),
			RetentionEnabled:  viper.GetBool("DB_RETENTION_ENABLED"),
			MaxRetentionRows:  parseRetentionRowsConfig(viper.GetString("DB_MAX_RETENTION_ROWS")),
			FullBackupEnabled: viper.GetBool("DB_FULL_BACKUP_ENABLED"),
		},
		EncryptionKey: viper.GetString("ENCRYPTION_KEY"),
		NTFYConfig: config.NTFYConfig{
			Topic: viper.GetString("NTFY_TOPIC"),
		},
	}

	serverClients, err := createClients(serverConfig)
	if err != nil {
		logger.Fatalf("Error creating clients: %s", err)
	}

	if err := serverClients.Mosquitto.Connect(); err != nil {
		logger.Fatalf("error connecting to mosquitto server: %s", err)
	}

	webServer := newWebServer(serverConfig, serverClients)

	allCfg, err := serverClients.Postgres.GetSensorConfig()
	if err != nil {
		logger.Fatalf("error getting sensor config from postgres: %s", err)
	}

	for _, c := range allCfg {
		sensorConfigMap[c.Source] = c.OpenTimeoutMinutes
	}

	// receive updates to sensorconfig from postgres
	go func(sensorConfigMap map[string]int32) {
		cfg := <-sensorConfigChan
		sensorConfigMap[cfg.Source] = cfg.OpenTimeoutMinutes
	}(sensorConfigMap)

	var delayTimerMap map[string]*time.Timer = make(map[string]*time.Timer)
	serverClients.Mosquitto.Subscribe(config.SensorStatusTopic, func(message string) {
		err := handleSensorStatusSubscribe(serverClients, webServer, serverConfig, message, delayTimerMap)
		if err != nil {
			logger.Errorf("handling sensor status message: %s", err)
		}
	})

	var heartbeatTimerMap map[string]*time.Timer = make(map[string]*time.Timer)
	serverClients.Mosquitto.Subscribe(config.SensorHeartbeatTopic, func(messageString string) {
		var h config.Heartbeat
		err := json.Unmarshal([]byte(messageString), &h)
		if err != nil {
			logger.Errorf("error unmarshalling message from heartbeat channel: %s. Message received was: %s", err, messageString)
			return
		}

		err = serverClients.DDClient.PublishHeartbeat(context.Background(), h.Name)
		if err != nil {
			logger.Errorf("publishing heartbeat for %s: %s", h.Name, err.Error())
		}

		currentTimer := heartbeatTimerMap[h.Name]
		if currentTimer != nil {
			currentTimer.Stop()
		}

		timer := time.AfterFunc(config.HeartbeatTimeout, func() { handleHeartbeatTimeout(h, serverClients, serverConfig, webServer) })
		heartbeatTimerMap[h.Name] = timer
	})

	if serverConfig.S3Config.FullBackupEnabled {
		runFullBackup(serverClients, serverConfig)
	}

	if serverConfig.S3Config.RetentionEnabled {
		runDataRetention(serverClients, serverConfig)
	}

	configureCronJobs(serverClients, serverConfig)

	err = webServer.httpServer.ListenAndServe()
	if err != nil {
		logger.Fatalf("Error starting web server: %s", err)
	}
}

func configureCronJobs(serverClients clients.ServerClients, serverConfig config.ServerConfig) {
	if serverConfig.S3Config.RetentionEnabled {
		dataTicker := time.NewTicker(dataRetentionCronFrequency)
		go func() {
			for range dataTicker.C {
				runDataRetention(serverClients, serverConfig)
			}
		}()
	}

	if serverConfig.S3Config.FullBackupEnabled {
		dataTicker := time.NewTicker(fullBackupCronFrequency)
		go func() {
			for range dataTicker.C {
				runFullBackup(serverClients, serverConfig)
			}
		}()
	}

	if !serverConfig.MockMode {
		t := time.NewTicker(tokenExpMetricFreq)
		go func() {
			for range t.C {
				for _, token := range serverConfig.DatadogConfig.TokensMetadata {
					err := serverClients.DDClient.PublishTokenDaysLeft(context.Background(), token)
					if err != nil {
						logger.Errorf("error publishing token '%s' days left: %s", token.Name, err)
					}
				}
			}
		}()
	}
}

// using viper.Getint is unsafe because if the env
// var is unset, viper will return 0 resulting in
// all rows being deleted from the database
func parseRetentionRowsConfig(rows string) int {
	if rows == "" {
		return 10000
	}
	rowsInt, err := strconv.Atoi(rows)
	if err != nil {
		logger.Fatalf("failed to parse retention rows from string: %s", err)
	}
	return rowsInt
}

func runDataRetention(serverClients clients.ServerClients, serverConfig config.ServerConfig) {
	logger.Info("Running data retention")
	rowsAboveMax, err := serverClients.Postgres.GetRowsAboveMax(serverConfig.S3Config.MaxRetentionRows)
	if err != nil {
		logger.Errorf("error getting rows above max: %s", err)
		return
	}
	numberRowsAboveMax := len(rowsAboveMax)

	if numberRowsAboveMax == 0 {
		logger.Info("Row count is less than or equal to max, no action required")
		return
	}

	ctx := context.Background()

	err = serverClients.AWS.DownloadOrCreateBackupFile(ctx, serverClients.AWS.RetentionTmpWritePath, serverClients.AWS.RetentionBackupFileKey)
	if err != nil {
		logger.Errorf("downloading or creating backup file: %s", err)
		return
	}

	append := true
	err = serverClients.AWS.WriteBackupFile(rowsAboveMax, append, serverClients.AWS.RetentionTmpWritePath)
	if err != nil {
		logger.Errorf("writing local tmp backup file: %s", err)
		return
	}

	err = serverClients.AWS.UploadBackupFile(ctx, serverClients.AWS.RetentionTmpWritePath, serverClients.AWS.RetentionBackupFileKey)
	if err != nil {
		logger.Errorf("uploading backup file to S3: %s", err)
		return
	}

	rowsAffected, err := serverClients.Postgres.DeleteRows(rowsAboveMax)
	if err != nil {
		logger.Errorf("deleting rows from postgres: %s", err)
		return
	}

	numRowsAffected := int(rowsAffected)
	if numRowsAffected != numberRowsAboveMax {
		logger.Warnf("Number of rows deleted '%d' did not match expected number '%d'. This could indicate a data loss situation", rowsAffected, numberRowsAboveMax)
		return
	}

	if numRowsAffected > 0 {
		logger.Infof("Number of rows deleted and stored in S3 backup: %d", numRowsAffected)
	}
}

func runFullBackup(serverClients clients.ServerClients, serverConfig config.ServerConfig) {
	logger.Info("Running full database backup")
	rows, err := serverClients.Postgres.GetAllRows()
	if err != nil {
		logger.Errorf("getting all rows from db: %s", err)
		return
	}

	append := false
	err = serverClients.AWS.WriteBackupFile(rows, append, serverClients.AWS.FullBackupTmpWritePath)
	if err != nil {
		logger.Errorf("writing backup tmp file: %s", err)
		return
	}

	ctx := context.Background()
	err = serverClients.AWS.UploadBackupFile(ctx, serverClients.AWS.FullBackupTmpWritePath, serverClients.AWS.FullBackupFileKey)
	if err != nil {
		logger.Errorf("uploading backup file to S3: %s", err)
		return
	}

	logger.Infof("Full backup to S3 success, number of rows backed up: %d", len(rows))
}

func createClients(serverConfig config.ServerConfig) (clients.ServerClients, error) {
	var redisClient redis.Client
	var err error

	if serverConfig.RedisTLSURL != "" {
		redisClient, err = redis.NewRedisClient(serverConfig.RedisTLSURL, true)
	} else {
		redisClient, err = redis.NewRedisClient(serverConfig.RedisURL, false)
	}

	if err != nil {
		return clients.ServerClients{}, fmt.Errorf("creating redis client: %s", err)
	}

	postgresClient, err := postgres.NewPostgresClient(serverConfig.PostgresURL, sensorConfigChan)
	if err != nil {
		return clients.ServerClients{}, fmt.Errorf("creating postgres client: %s", err)
	}

	mosquittoAddr := fmt.Sprintf("mqtts://%s:%s@%s:1883", serverConfig.MosquittoServerUser, serverConfig.MosquittoServerPassword, serverConfig.MosquittoServerDomain)

	insecureSkipVerifyMosquitto := false
	mosquittoClient := mqtt.NewMQTTClient(mosquittoAddr, insecureSkipVerifyMosquitto, func(client mqttC.Client) {
		logger.Info("Connected to mosquitto server")
	}, func(client mqttC.Client, err error) {
		logger.Warnf("Connection to mosquitto server lost: %v", err)
	}, func(mqttC.Client, *mqttC.ClientOptions) {
		logger.Info("Server client is reconnecting")
	})

	awsClient, err := aws.NewClient(serverConfig)
	if err != nil {
		return clients.ServerClients{}, fmt.Errorf("error creating AWS client: %s", err)
	}
	ddClient := datadog.NewDatadogClient(serverConfig.DatadogConfig.APIKey, serverConfig.DatadogConfig.APPKey)

	cryptoUtil, err := crypto.NewUtil(serverConfig.EncryptionKey)
	if err != nil {
		return clients.ServerClients{}, fmt.Errorf("error creating crypto client: %s", err)
	}

	return clients.ServerClients{
		Redis:      redisClient,
		Postgres:   postgresClient,
		Mosquitto:  mosquittoClient,
		AWS:        awsClient,
		DDClient:   ddClient,
		CryptoUtil: cryptoUtil,
	}, nil
}

func handleHeartbeatTimeout(h config.Heartbeat, serverClients clients.ServerClients, serverConfig config.ServerConfig, webServer WebServer) {
	if h.Type == config.HeartbeatTypeSensor {
		messageString, err := serverClients.Redis.ReadState(h.Name, context.Background())
		if err != nil {
			logger.Errorf("Error handling timeout: reading redis state: %s Message string was: %s", err, messageString)
			return
		}

		lastStatus := config.SensorStatus{}
		err = json.Unmarshal([]byte(messageString), &lastStatus)
		if err != nil {
			logger.Errorf("Error handling timeout, unmarshalling state: %s", err)
			return
		}

		logger.Errorf("Heartbeat timeout occurred for %s", lastStatus.Source)

		lastStatus.Status = config.UNKNOWN
		lastStatusJson, err := json.Marshal(lastStatus)
		if err != nil {
			logger.Errorf("Error handling timeout: marshalling state: %s", err)
			return
		}
		err = serverClients.Redis.WriteState(lastStatus.Source, string(lastStatusJson), context.Background())
		if err != nil {
			logger.Errorf("Error writing message state after heartbeat timeout. Message: %s", messageString)
			return
		}

		if !serverConfig.MockMode {
			msg := config.NTFYMessage{
				Body:     fmt.Sprintf("%s sensor lost connection", h.Name),
				Priority: "urgent",
				Tags:     []string{"rotating_light,skull"},
			}
			err := sendPushNotification(serverClients, serverConfig, msg)
			if err != nil {
				logger.Errorf("sending lost connection push notification: %w", err)
			}
		}

		webServer.SendMessage(config.SensorStatusTopic, lastStatus)

		writeErr := serverClients.Postgres.WriteSensorStatus(lastStatus)
		if writeErr != nil {
			logger.Errorf("Error writing sensor status to postgres: %s", writeErr)
		}
	} else {
		logger.Errorf("Unknown heartbeat type receieved: %s", h)
	}
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
	if err != nil {
		if err.Error() != "redis: nil" {
			return fmt.Errorf("reading state from redis: %w", err)
		}
		armedString = ""
	}

	armed := true
	if armedString == "" || armedString == "false" {
		armed = false
	}

	err = serverClients.Mosquitto.PublishHASensorStatus(currentStatus)
	if err != nil {
		return fmt.Errorf("publishing ha sensor status: %w", err)
	}

	if (lastStatus.Status == config.CLOSED && currentStatus.Status == config.OPEN) || (lastStatus.Status == config.UNKNOWN && currentStatus.Status == config.OPEN) {
		logger.Infof("%s was just opened", currentStatus.Source)
		if armed {
			msg := config.NTFYMessage{
				Body:     fmt.Sprintf("%s was just opened", currentStatus.Source),
				Priority: "default",
				Tags:     []string{"loudspeaker"},
			}

			err = sendPushNotification(serverClients, serverConfig, msg)
			if err != nil {
				return fmt.Errorf("sending push notifications: %w", err)
			}
		}
	}

	if currentStatus.Status == config.OPEN {
		openTimeout, ok := sensorConfigMap[currentStatus.Source]
		if !ok {
			openTimeout = int32(config.DefaultOpenTimeoutMinutes)
		}

		duration := time.Duration(openTimeout) * time.Minute
		timer := time.AfterFunc(duration, func() {
			handleOpenTimeout(serverClients, serverConfig, currentStatus, armed, serverConfig.MockMode, openTimeout)
		})
		delayTimerMap[currentStatus.Source] = timer
	} else if currentStatus.Status == config.CLOSED {
		currentTimer := delayTimerMap[currentStatus.Source]
		if currentTimer != nil {
			currentTimer.Stop()
		}
	} else {
		logger.Errorf("Message status '%s' not recognized", currentStatus.Status)
	}

	err = serverClients.Redis.WriteState(currentStatus.Source, string(currentStatusJson), context.Background())
	if err != nil {
		return fmt.Errorf("writing state to Redis: %s", err)
	}

	webServer.SendMessage(config.SensorStatusTopic, currentStatus)
	writeErr := serverClients.Postgres.WriteSensorStatus(currentStatus)
	if writeErr != nil {
		return fmt.Errorf("writing sensor status to postgres: %s", writeErr)
	}
	return nil
}

func handleOpenTimeout(serverClients clients.ServerClients, serverConfig config.ServerConfig, s config.SensorStatus, armed, mockMode bool, openTimeout int32) {
	body := fmt.Sprintf("%s opened longer than %d min", s.Source, openTimeout)
	logger.Warn(body)
	if !mockMode && armed {
		msg := config.NTFYMessage{
			Body:     body,
			Priority: "default",
			Tags:     []string{"warning"},
		}
		err := sendPushNotification(serverClients, serverConfig, msg)
		if err != nil {
			logger.Errorf("sending push notification: %w", err)
		}
	}
}

func buildTokenMetadata() []config.TokenMetadata {
	return []config.TokenMetadata{
		{
			Name:       "github-ci",
			Owner:      "digitalocean",
			Expiration: viper.GetString("DO_TOKEN_EXP_GITHUB_CI"),
		},
		{
			Name:       "github-ci",
			Owner:      "tailscale",
			Expiration: viper.GetString("TS_TOKEN_EXP_GITHUB_CI"),
		},
	}
}

func sendPushNotification(serverClients clients.ServerClients, serverConfig config.ServerConfig, msg config.NTFYMessage) error {
	req, _ := http.NewRequest("POST", fmt.Sprintf("https://ntfy.sh/%s", serverConfig.NTFYConfig.Topic), strings.NewReader(msg.Body))
	req.Header.Set("Title", "Pi Sensor Update")
	req.Header.Set("Priority", msg.Priority)
	req.Header.Set("Tags", strings.Join(msg.Tags, ","))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("posting to ntfy: %w", err)
	}

	var r config.NTFYResponse
	err = json.NewDecoder(resp.Body).Decode(&r)
	if err != nil {
		return fmt.Errorf("decoding ntfy response: %w", err)
	}

	logger.Infof("response from ntfy: (id: %s) (event: %s)", r.Id, r.Event)

	return nil
}
