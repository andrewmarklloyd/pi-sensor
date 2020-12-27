package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/Shopify/sarama"
	"github.com/ivanbeldad/kasa-go"
	"github.com/robfig/cron/v3"
)

func init() {
	sarama.Logger = log.New(os.Stdout, "[Consumer] ", log.LstdFlags)
}

var (
	brokers   = flag.String("brokers", os.Getenv("CLOUDKARAFKA_BROKERS"), "The Kafka brokers to connect to, as a comma separated list")
	userName  = flag.String("username", os.Getenv("CLOUDKARAFKA_USERNAME"), "The SASL username")
	passwd    = flag.String("passwd", os.Getenv("CLOUDKARAFKA_PASSWORD"), "The SASL password")
	topic     = flag.String("topic", os.Getenv("CLOUDKARAFKA_TOPIC"), "The topic to consume")
	group     = flag.String("group", os.Getenv("CLOUDKARAFKA_CONSUMER_GROUP"), "The consumer group id")
	algorithm = flag.String("algorithm", "sha512", "The SASL SCRAM SHA algorithm sha256 or sha512 as mechanism")
	certFile  = flag.String("certificate", "", "The optional certificate file for client authentication")
	keyFile   = flag.String("key", "", "The optional key file for client authentication")
	caFile    = flag.String("ca", "", "The optional certificate authority file for TLS client authentication")
	verifySSL = flag.Bool("verify", false, "Optional verify ssl certificates chain")
	useTLS    = flag.Bool("tls", true, "Use TLS to communicate with the cluster")
	logMsg    = flag.Bool("logmsg", false, "True to log consumed messages to console")

	kasaUsername = flag.String("kasaUsername", os.Getenv("KASA_USERNAME"), "The Kasa/TP Link cloud username")
	kasaPassword = flag.String("kasaPassword", os.Getenv("KASA_PASSWORD"), "The Kasa/TP Link cloud password")
	kasaAPI      kasa.API
	hs100        kasa.HS100

	testMode = flag.String("testMode", os.Getenv("TEST_MODE"), "Test mode for local development")

	logger = log.New(os.Stdout, "[Pi-Sensor Consumer] ", log.LstdFlags)

	mockData []string
)

func createTLSConfiguration() (t *tls.Config) {
	t = &tls.Config{
		InsecureSkipVerify: true,
	}
	return t
}

// Consumer represents a Sarama consumer group consumer
type Consumer struct {
	ready chan bool
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (consumer *Consumer) Setup(sarama.ConsumerGroupSession) error {
	// Mark the consumer as ready
	close(consumer.ready)
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (consumer *Consumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
func (consumer *Consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {

	// NOTE:
	// Do not move the code below to a goroutine.
	// The `ConsumeClaim` itself is called within a goroutine, see:
	// https://github.com/Shopify/sarama/blob/master/consumer_group.go#L27-L29
	for message := range claim.Messages() {
		log.Printf("Message claimed: value = %s, timestamp = %v, topic = %s", string(message.Value), message.Timestamp, message.Topic)
		session.MarkMessage(message, "")
		send(string(message.Value))
	}

	return nil
}

func configConsumer() {
	conf := sarama.NewConfig()
	conf.Producer.Retry.Max = 1
	conf.Producer.RequiredAcks = sarama.WaitForAll
	conf.Producer.Return.Successes = true
	conf.Metadata.Full = true
	conf.Version = sarama.V0_10_2_0
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

	consumer := Consumer{
		ready: make(chan bool),
	}

	ctx, cancel := context.WithCancel(context.Background())
	splitBrokers := strings.Split(*brokers, ",")
	client, err := sarama.NewConsumerGroup(splitBrokers, *group, conf)
	if err != nil {
		log.Panicf("Error creating consumer group client: %v", err)
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			// `Consume` should be called inside an infinite loop, when a
			// server-side rebalance happens, the consumer session will need to be
			// recreated to get the new claims

			if err := client.Consume(ctx, []string{*topic}, &consumer); err != nil {
				log.Panicf("Error from consumer: %v", err)
			}
			// check if context was cancelled, signaling that the consumer should stop
			if ctx.Err() != nil {
				return
			}
			consumer.ready = make(chan bool)
		}
	}()

	<-consumer.ready // Await till the consumer has been set up
	log.Println("Sarama consumer up and running!...")

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-ctx.Done():
		log.Println("terminating: context cancelled")
	case <-sigterm:
		log.Println("terminating: via signal")
	}
	cancel()
	wg.Wait()
	if err = client.Close(); err != nil {
		log.Panicf("Error closing client: %v", err)
	}
}

func configOutlet() {
	// kasaAPI, err := kasa.Connect(*kasaUsername, *kasaPassword)
	// if err != nil {
	// 	log.Fatalln(err)
	// }

	// devices, _ := kasaAPI.GetDevicesInfo()
	// for _, device := range devices {
	// 	fmt.Println(device)
	// }
}

func getMockData() string {
	rand.Seed(time.Now().Unix()) // initialize global pseudo random generator
	return mockData[rand.Intn(len(mockData))]
}

func main() {
	flag.Parse()

	if *brokers == "" {
		log.Fatalln("at least one broker is required")
	}

	if *userName == "" {
		log.Fatalln("SASL username is required")
	}

	if *passwd == "" {
		log.Fatalln("SASL password is required")
	}

	if *testMode == "true" {
		mockData = make([]string, 0)
		mockData = append(mockData,
			"{\"state\":\"OPEN\",\"source\":\"office-door\"}",
			"{\"state\":\"CLOSED\",\"source\":\"office-door\"}")
		logger.Println("Running in test mode")
		cronLib := cron.New()
		cronLib.AddFunc(fmt.Sprintf("@every %ds", 5), func() {
			send(getMockData())
		})
		cronLib.Start()
		NewServer()
	} else {
		go func() {
			NewServer()
		}()
		configConsumer()
	}
}
