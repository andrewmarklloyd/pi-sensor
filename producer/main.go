package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	guuid "github.com/google/uuid"

	"github.com/Shopify/sarama"
	"github.com/stianeikeland/go-rpio"
)

const (
	OPEN    = "OPEN"
	CLOSED  = "CLOSED"
	UNKNOWN = "UNKNOWN"
)

var (
	brokers      = flag.String("brokers", os.Getenv("KAFKA_BROKERS"), "The Kafka brokers to connect to, as a comma separated list")
	userName     = flag.String("username", os.Getenv("KAFKA_USERNAME"), "The SASL username")
	passwd       = flag.String("passwd", os.Getenv("KAFKA_PASSWORD"), "The SASL password")
	topic        = flag.String("topic", os.Getenv("KAFKA_TOPIC"), "The topic to consume")
	sensorSource = flag.String("sensorSource", os.Getenv("SENSOR_SOURCE"), "The sensor location or name")
	certFile     = flag.String("certificate", "", "The optional certificate file for client authentication")
	keyFile      = flag.String("key", "", "The optional key file for client authentication")
	caFile       = flag.String("ca", "", "The optional certificate authority file for TLS client authentication")
	verifySSL    = flag.Bool("verify", false, "Optional verify ssl certificates chain")
	useTLS       = flag.Bool("tls", true, "Use TLS to communicate with the cluster")
	syncProducer sarama.SyncProducer
	pin          rpio.Pin
	logger       = log.New(os.Stdout, "[Pi-Senser Producer] ", log.LstdFlags)
)

type GPIO struct {
}

func (g GPIO) SetupGPIO(pinNumber int) error {
	pin = rpio.Pin(pinNumber)

	err := rpio.Open()
	if err != nil {
		log.Println(fmt.Sprintf("Unable to open gpio: %s, continuing but running in test mode.", err.Error()))
	}

	return nil
}

func (g GPIO) Cleanup() {
	rpio.Close()
	_ = syncProducer.Close()
}

func (g GPIO) CurrentStatus() string {
	var pinState int
	pinState = int(pin.Read())

	if pinState == 0 {
		return CLOSED
	} else if pinState == 1 {
		return OPEN
	}
	return UNKNOWN
}

func initKafka() {
	flag.Parse()

	if *brokers == "" {
		log.Fatalln("at least one broker is required")
	}
	splitBrokers := strings.Split(*brokers, ",")

	if *userName == "" {
		log.Fatalln("SASL username is required")
	}

	if *passwd == "" {
		log.Fatalln("SASL password is required")
	}

	conf := sarama.NewConfig()
	conf.Producer.Retry.Max = 1
	conf.Producer.RequiredAcks = sarama.WaitForAll
	conf.Producer.Return.Successes = true
	conf.Metadata.Full = true
	conf.Version = sarama.V0_10_0_0
	conf.ClientID = "sasl_scram_client"
	conf.Metadata.Full = true
	conf.Net.SASL.Enable = true
	conf.Net.SASL.User = *userName
	conf.Net.SASL.Password = *passwd
	conf.Net.SASL.Handshake = true
	conf.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient { return &XDGSCRAMClient{HashGeneratorFcn: SHA512} }
	conf.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA512

	if *useTLS {
		conf.Net.TLS.Enable = true
		conf.Net.TLS.Config = createTLSConfiguration()
	}
	var err error
	syncProducer, err = sarama.NewSyncProducer(splitBrokers, conf)
	if err != nil {
		logger.Fatalln("failed to create producer: ", err)
	}
}

func createTLSConfiguration() (t *tls.Config) {
	t = &tls.Config{
		InsecureSkipVerify: true,
	}
	return t
}

func initPin() {
	defaultPin := 15
	pinNum, err := strconv.Atoi(os.Getenv("GPIO_PIN"))
	if err != nil {
		log.Printf("Failed to parse GPIO_PIN env var, using default %d", defaultPin)
		pinNum = defaultPin
	}

	var gpio GPIO
	gpio = GPIO{}
	err = gpio.SetupGPIO(pinNum)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		gpio.Cleanup()
		fmt.Println("Cleaning up")
		os.Exit(1)
	}()

	lastStatus := "CLOSED"
	currentStatus := gpio.CurrentStatus()
	for true {
		currentStatus = gpio.CurrentStatus()
		if currentStatus != lastStatus {
			lastStatus = currentStatus
			logger.Printf("Current status: %s", currentStatus)
			produce(fmt.Sprintf("{\"state\":\"%s\",\"source\":\"%s\"}", currentStatus, *sensorSource))
		}
		time.Sleep(time.Second)
	}
}

func produce(message string) {
	_, _, err := syncProducer.SendMessage(&sarama.ProducerMessage{
		Topic: *topic,
		Value: sarama.StringEncoder(message),
	})
	if err != nil {
		logger.Fatalln("failed to send message to ", *topic, err)
	}
}

func main() {
	logger.Print("Initializing app")
	initKafka()

	testMode := os.Getenv("TEST_MODE") == "true"
	if !testMode {
		initPin()
	} else {
		id := guuid.New()
		produce(id.String())
	}
}
