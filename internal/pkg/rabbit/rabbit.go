package rabbit

import (
	"context"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitProducer struct {
	channel   *amqp.Channel
	queueName string
}

type ConsumerCallback func(d amqp.Delivery)

func Consumer(cloudAMQPURL, queueName string, callback ConsumerCallback) error {
	config := amqp.Config{
		Properties: amqp.NewConnectionProperties(),
	}
	config.Properties.SetClientConnectionName("pi-sensor-consumer")

	conn, err := amqp.DialConfig(cloudAMQPURL, config)
	if err != nil {
		return fmt.Errorf("dialing AMQP: %w", err)
	}

	defer conn.Close()

	channelRabbitMQ, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("opening a channel: %w", err)
	}

	defer channelRabbitMQ.Close()

	messages, err := channelRabbitMQ.Consume(
		queueName, // queue name
		"",        // consumer
		true,      // auto-ack
		false,     // exclusive
		false,     // no local
		false,     // no wait
		nil,       // arguments
	)
	if err != nil {
		return fmt.Errorf("opening consume of queue: %w", err)
	}

	forever := make(chan bool)

	go func() {
		for message := range messages {
			callback(message)
		}
	}()

	<-forever

	return nil
}

func NewProducer(amqpServerURL, queueName string) (RabbitProducer, error) {

	connectRabbitMQ, err := amqp.Dial(amqpServerURL)
	if err != nil {
		return RabbitProducer{}, err
	}
	// defer connectRabbitMQ.Close()

	channelRabbitMQ, err := connectRabbitMQ.Channel()
	if err != nil {
		return RabbitProducer{}, err
	}
	// defer channelRabbitMQ.Close()

	_, err = channelRabbitMQ.QueueDeclare(
		queueName, // queue name
		true,      // durable
		false,     // auto delete
		false,     // exclusive
		false,     // no wait
		nil,       // arguments
	)

	if err != nil {
		return RabbitProducer{}, err
	}

	return RabbitProducer{
		channel:   channelRabbitMQ,
		queueName: queueName,
	}, err

}

func (rp *RabbitProducer) Publish(queueName, body string) error {

	message := amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte(body),
	}

	if err := rp.channel.PublishWithContext(
		context.Background(),
		"",        // exchange
		queueName, // queue name
		false,     // mandatory
		false,     // immediate
		message,   // message to publish
	); err != nil {
		return err
	}

	return nil
}
