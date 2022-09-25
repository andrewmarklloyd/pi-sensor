package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/aws"
	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/clients"
	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/config"
	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/mqtt"
	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/postgres"
	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/redis"
	mqttC "github.com/eclipse/paho.mqtt.golang"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var (
	logger          *zap.SugaredLogger
	forwarderLogger *zap.SugaredLogger
	version         = "unknown"
)

const (
	dataRetentionCronFrequency = 12 * time.Hour
	fullBackupCronFrequency    = 6 * time.Hour
)

func runServer() {
	l, _ := zap.NewProduction()
	logger = l.Sugar().Named("pi-sensor-server")
	defer logger.Sync()
	logger.Infof("Running server version: %s", version)

	forwarderLogger = l.Sugar().Named("pi-sensor-agent")
	defer forwarderLogger.Sync()

	serverConfig := config.ServerConfig{
		AppName:            viper.GetString("APP_NAME"),
		MqttBrokerURL:      viper.GetString("CLOUDMQTT_URL"),
		MqttServerUser:     viper.GetString("CLOUDMQTT_SERVER_USER"),
		MqttServerPassword: viper.GetString("CLOUDMQTT_SERVER_PASSWORD"),
		RedisURL:           viper.GetString("REDIS_URL"),
		RedisTLSURL:        viper.GetString("REDIS_TLS_URL"),
		PostgresURL:        viper.GetString("DATABASE_URL"),
		Port:               viper.GetString("PORT"),
		MockMode:           viper.GetBool("MOCK_MODE"),
		AllowedAPIKeys:     viper.GetStringSlice("ALLOWED_API_KEYS"),
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
	}

	serverClients, err := createClients(serverConfig)
	if err != nil {
		logger.Fatalf("Error creating clients: %s", err)
	}

	err = serverClients.Mqtt.Connect()
	if err != nil {
		logger.Fatalf("error connecting to mqtt: %s", err)
	}

	logger.Infof("Using bucket:", serverConfig.S3Config.Bucket)
	info, err := serverClients.AWS.GetBucketInfo(context.Background())
	if err != nil {
		logger.Fatalf("error getting bucket info: %s", err)
	}

	logger.Infof("AWS Bucket Info - Size: %d bytes, Versions: %d, DeleteMarkers: %d", info.Size, info.NumVersions, info.NumDeleteMarkers)

	webServer := newWebServer(serverConfig, serverClients)

	var delayTimerMap map[string]*time.Timer = make(map[string]*time.Timer)
	serverClients.Mqtt.Subscribe(config.SensorStatusTopic, func(message string) {
		err := handleSensorStatusSubscribe(serverClients, webServer, serverConfig, message, delayTimerMap)
		if err != nil {
			logger.Errorf("handling sensor status message: %s", err)
		}
	})

	var heartbeatTimerMap map[string]*time.Timer = make(map[string]*time.Timer)
	serverClients.Mqtt.Subscribe(config.SensorHeartbeatTopic, func(messageString string) {
		var h config.Heartbeat
		err := json.Unmarshal([]byte(messageString), &h)
		if err != nil {
			logger.Errorf("error unmarshalling message from heartbeat channel: %s. Message received was: %s", err, messageString)
			return
		}
		currentTimer := heartbeatTimerMap[h.Name]
		if currentTimer != nil {
			currentTimer.Stop()
		}

		timer := time.AfterFunc(config.HeartbeatTimeout, func() { handleHeartbeatTimeout(h, serverClients, serverConfig, webServer) })
		heartbeatTimerMap[h.Name] = timer
	})

	if serverConfig.S3Config.RetentionEnabled {
		runDataRetention(serverClients, serverConfig)
	}

	if serverConfig.S3Config.FullBackupEnabled {
		runFullBackup(serverClients, serverConfig)
	}

	// configureCronJobs(serverClients, serverConfig)

	err = webServer.httpServer.ListenAndServe()
	if err != nil {
		logger.Fatalf("Error starting web server: %s", err)
	}
}

func configureCronJobs(serverClients clients.ServerClients, serverConfig config.ServerConfig) {
	ticker := time.NewTicker(6 * time.Hour)
	go func() {
		for range ticker.C {
			if err := serverClients.Messenger.CheckBalance(); err != nil {
				logger.Errorf("checking twilio balance: %s", err)
			}
		}
	}()

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
		logger.Infof("Row count is less than or equal to max %d, no action required", serverConfig.S3Config.MaxRetentionRows)
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
	redisClient, err := redis.NewRedisClient(serverConfig.RedisTLSURL)
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

	mqttClient := mqtt.NewMQTTClient(mqttAddr, func(client mqttC.Client) {
		logger.Info("Connected to MQTT server")
	}, func(client mqttC.Client, err error) {
		logger.Fatalf("Connection to MQTT server lost: %v", err)
	})

	awsClient, err := aws.NewClient(serverConfig)
	if err != nil {
		return clients.ServerClients{}, fmt.Errorf("error creating AWS client: %s", err)
	}

	return clients.ServerClients{
		Redis:    redisClient,
		Postgres: postgresClient,
		Mqtt:     mqttClient,
		AWS:      awsClient,
	}, nil
}

func handleHeartbeatTimeout(h config.Heartbeat, serverClients clients.ServerClients, serverConfig config.ServerConfig, webServer WebServer) {
	if h.Type == config.HeartbeatTypeApp {
		logger.Warnf("Heartbeat timeout occurred for %s", h.Name)
		if !serverConfig.MockMode {
			err := serverClients.Mqtt.PublishHASensorLostConnection(h.Name)
			if err != nil {
				logger.Errorf("publishing lost connection message: %w", err)
			}
		}
	} else if h.Type == config.HeartbeatTypeSensor {
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

		logger.Warnf("Heartbeat timeout occurred for %s", lastStatus.Source)

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
			err := serverClients.Mqtt.PublishHASensorLostConnection(lastStatus.Source)
			if err != nil {
				logger.Errorf("publishing lost connection message: %w", err)
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
	armed := true
	if armedString == "" || armedString == "false" {
		armed = false
	}

	err = serverClients.Mqtt.PublishHASensorStatus(currentStatus)
	if err != nil {
		return fmt.Errorf("publishing ha sensor status: %w", err)
	}

	if (lastStatus.Status == config.CLOSED && currentStatus.Status == config.OPEN) || (lastStatus.Status == config.UNKNOWN && currentStatus.Status == config.OPEN) {
		logger.Infof("%s was just opened", currentStatus.Source)
		if !serverConfig.MockMode && armed {
			err = serverClients.Mqtt.PublishHASensorNotify(currentStatus)
			if err != nil {
				return fmt.Errorf("publishing ha sensor notify: %w", err)
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
		logger.Errorf("Message status '%s' not recognized", currentStatus.Status)
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
	logger.Warn(message)
	if !mockMode && armed {
		err := serverClients.Mqtt.PublishHAOpenWarn(s)
		if err != nil {
			logger.Errorf("publishing HA open warning message: %w", err)
		}
	}
}
