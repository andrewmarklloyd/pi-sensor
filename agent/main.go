package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	mqttC "github.com/eclipse/paho.mqtt.golang"
	"github.com/jaedle/golang-tplink-hs100/pkg/configuration"
	"github.com/jaedle/golang-tplink-hs100/pkg/hs100"
	"go.uber.org/zap"

	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/config"
	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/gpio"
	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/mqtt"
	"github.com/andrewmarklloyd/pi-sensor/internal/pkg/tailscale"
)

var (
	mosquittoDomain               = flag.String("mosquittodomain", os.Getenv("MOSQUITTO_DOMAIN"), "The mosquitto domain to connect")
	mosquittoAgentUser            = flag.String("mosquittoagentuser", os.Getenv("MOSQUITTO_AGENT_USER"), "The mosquitto agent user to connect")
	mosquittoAgentPassword        = flag.String("mosquittoagentpassword", os.Getenv("MOSQUITTO_AGENT_PASSWORD"), "The mosquitto agent password to connect")
	mosquittoProtocol             = flag.String("mosquittoprotocol", os.Getenv("MOSQUITTO_PROTOCOL"), "The mosquitto protocol to connect")
	sensorSource                  = flag.String("sensorSource", os.Getenv("SENSOR_SOURCE"), "The sensor location or name")
	mockFlag                      = flag.String("mockMode", os.Getenv("MOCK_MODE"), "Mock mode for local development")
	sensorReadIntervalSecondsFlag = flag.String("sensorReadIntervalSeconds", os.Getenv("SENSOR_READ_INTERVAL_SECONDS"), "How often in seconds to read the sensor status")
	version                       = "unknown"
)

const (
	heartbeatIntervalSeconds = 60
)

func main() {
	l, err := zap.NewProduction()
	if err != nil {
		log.Fatalln("Error creating logger:", err)
	}
	// need a temporary init structured logger before reading sensorSource
	initLogger := l.Sugar().Named("pi-sensor-agent-init")
	defer initLogger.Sync()

	flag.Parse()
	if *sensorSource == "" {
		initLogger.Fatal("SENSOR_SOURCE env var is required")
	}

	logger := l.Sugar().Named(fmt.Sprintf("pi_sensor_agent-%s", *sensorSource))
	defer logger.Sync()

	// todo: uncomment and improve after able to use command on rpi
	// limited, resetDuration, err := op.GetRateLimit()
	// if err != nil {
	// 	logger.Errorf("error getting 1password rate limiting: %s", err.Error())
	// }
	// if limited {
	// 	logger.Errorf("being rate limited by 1password until %s, starting maintenance web server", resetDuration)
	// }

	logger.Infof("Initializing app, version: %s", version)

	mockMode, _ := strconv.ParseBool(*mockFlag)

	defaultPin := 18
	pinNum, err := strconv.Atoi(os.Getenv("GPIO_PIN"))
	if err != nil {
		logger.Infof("Failed to parse GPIO_PIN env var, using default %d", defaultPin)
		pinNum = defaultPin
	} else {
		logger.Infof("Using GPIO_PIN %d", pinNum)
	}

	sensorReadIntervalSeconds, err := strconv.Atoi(*sensorReadIntervalSecondsFlag)
	if err != nil {
		logger.Fatalf("converting SENSOR_READ_INTERVAL_SECONDS to int: %s", err)
	}

	mosquittoClient := configureMosquittoClient(*mosquittoDomain, *mosquittoAgentUser, *mosquittoAgentPassword, *mosquittoProtocol, *logger)
	if err := mosquittoClient.Connect(); err != nil {
		logger.Fatalf("error connecting to mosquitto server: %s", err)
	}

	pinClient := gpio.NewPinClient(pinNum, mockMode)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		logger.Info("SIGTERM received, cleaning up")
		mosquittoClient.Cleanup()
		pinClient.Cleanup()
		os.Exit(0)
	}()

	h := config.Heartbeat{
		Name:    *sensorSource,
		Type:    config.HeartbeatTypeSensor,
		Version: version,
	}

	ticker := time.NewTicker(heartbeatIntervalSeconds * time.Second)
	go func() {
		for range ticker.C {
			if err := mosquittoClient.PublishHeartbeat(h); err != nil {
				logger.Errorf("error publishing mosquitto heartbeat: %s", err)
			}

		}
	}()

	tailscaleStatusTicker := time.NewTicker(time.Hour)
	go func() {
		for range tailscaleStatusTicker.C {
			status, err := tailscale.CheckStatus()
			if err != nil {
				logger.Errorf("error checking tailscale status: %s", err)
			} else {
				if status.BackendState != "Running" {
					logger.Errorf("Tailscale BackendState should be 'Running' but value is: '%s'", status.BackendState)
				}
			}
		}
	}()

	mosquittoClient.Subscribe(config.SensorRestartTopic, func(messageString string) {
		if *sensorSource == messageString {
			logger.Info("Received restart message, restarting app now")
			os.Exit(0)
		}
	})

	statusFile := getStatusFileName(*sensorSource)

	lastStatus, err := getLastStatus(statusFile)
	if err != nil {
		logger.Warnf("error reading status file: %s. Setting status to %s", err, config.UNKNOWN)
		lastStatus = config.UNKNOWN
	}

	outletEnabled := os.Getenv("OUTLET_ENABLED") == "true" || os.Getenv("OUTLET_ENABLED") == "True"
	var device *hs100.Hs100
	if outletEnabled {
		logger.Info("Outlet enabled, setting up device")
		device, err = getHS100Device()
		if err != nil {
			logger.Errorf("error getting hs100 device: %w", err)
		}
	}

	var currentStatus string
	for {
		currentStatus = pinClient.CurrentStatus()

		err = writeStatus(statusFile, currentStatus)
		if err != nil {
			logger.Errorf("error writing status file: %s", err)
		}
		if currentStatus != lastStatus {
			logger.Infof(fmt.Sprintf("%s is %s", *sensorSource, currentStatus))
			lastStatus = currentStatus

			if err := mosquittoClient.PublishSensorStatus(config.SensorStatus{
				Source:  *sensorSource,
				Status:  currentStatus,
				Version: version,
			}); err != nil {
				logger.Errorf("Error publishing message to sensor status channel: %s", err)
			}

			if outletEnabled && device != nil {
				if currentStatus == gpio.OPEN {
					if err := device.TurnOn(); err != nil {
						logger.Errorf("turning device on: %s", err.Error())
					}
				} else if currentStatus == gpio.CLOSED {
					if err := device.TurnOff(); err != nil {
						logger.Errorf("turning device off: %s", err.Error())
					}
				}
			}
		}
		time.Sleep(time.Duration(sensorReadIntervalSeconds) * time.Second)
	}
}

func getLastStatus(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return strings.Trim(strings.TrimSpace(string(b)), "\n"), nil
}

func writeStatus(path, status string) error {
	return os.WriteFile(path, []byte(status), 0644)
}

func getStatusFileName(sensorSource string) string {
	return fmt.Sprintf("%s/.pi-sensor-status-%s", os.Getenv("HOME"), sensorSource)
}

func configureMosquittoClient(domain, user, password, protocol string, logger zap.SugaredLogger) mqtt.MqttClient {
	mosquittoProtocol := "mqtts"
	if protocol != "" {
		mosquittoProtocol = protocol
	}
	mosquittoAddr := fmt.Sprintf("%s://%s:%s@%s:1883", mosquittoProtocol, user, password, domain)

	// todo: remove this after using prod certbot cert
	insecureSkipVerify := false
	mosquittoClient := mqtt.NewMQTTClient(mosquittoAddr, insecureSkipVerify, func(client mqttC.Client) {
		logger.Info("Connected to mosquitto server")
	}, func(client mqttC.Client, err error) {
		logger.Warnf("Connection to mosquitto server lost: %v", err)
	}, func(mqttC.Client, *mqttC.ClientOptions) {
		logger.Info("Agent client is reconnecting")
	})

	return mosquittoClient
}

func getHS100Device() (*hs100.Hs100, error) {
	allDevices, err := hs100.Discover("192.168.1.1/24", configuration.Default().WithTimeout(time.Second*5))
	if err != nil {
		return nil, err
	}

	var device *hs100.Hs100
	deviceName := "Growler"

	for _, d := range allDevices {
		name, err := d.GetName()
		if err != nil {
			return nil, err
		}

		if name == deviceName {
			device = d
		}
	}

	if device == nil {
		return nil, fmt.Errorf("device name %s could not be found", deviceName)
	}
	return device, nil
}
