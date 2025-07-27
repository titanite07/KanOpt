package messaging

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
)

type RabbitMQ struct {
	conn    *amqp091.Connection
	channel *amqp091.Channel
	logger  *logrus.Logger
}

type Event struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	BoardID   string                 `json:"boardId"`
	UserID    string                 `json:"userId"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

const (
	EventExchange    = "kanopt.events"
	EventQueue       = "kanopt.events.queue"
	EventRoutingKey  = "kanopt.event"
)

func NewRabbitMQ(url string, logger *logrus.Logger) (*RabbitMQ, error) {
	conn, err := amqp091.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	rmq := &RabbitMQ{
		conn:    conn,
		channel: channel,
		logger:  logger,
	}

	// Setup exchanges and queues
	if err := rmq.setup(); err != nil {
		rmq.Close()
		return nil, fmt.Errorf("failed to setup messaging: %w", err)
	}

	return rmq, nil
}

func (rmq *RabbitMQ) setup() error {
	// Declare exchange
	err := rmq.channel.ExchangeDeclare(
		EventExchange, // name
		"topic",       // type
		true,          // durable
		false,         // auto-deleted
		false,         // internal
		false,         // no-wait
		nil,           // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare exchange: %w", err)
	}

	// Declare queue
	_, err = rmq.channel.QueueDeclare(
		EventQueue, // name
		true,       // durable
		false,      // delete when unused
		false,      // exclusive
		false,      // no-wait
		nil,        // arguments
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	// Bind queue to exchange
	err = rmq.channel.QueueBind(
		EventQueue,      // queue name
		EventRoutingKey, // routing key
		EventExchange,   // exchange
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to bind queue: %w", err)
	}

	return nil
}

func (rmq *RabbitMQ) PublishEvent(event Event) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	err = rmq.channel.Publish(
		EventExchange,   // exchange
		EventRoutingKey, // routing key
		false,           // mandatory
		false,           // immediate
		amqp091.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp091.Persistent,
			Timestamp:    time.Now(),
			Body:         body,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	rmq.logger.WithFields(logrus.Fields{
		"event_id":   event.ID,
		"event_type": event.Type,
		"board_id":   event.BoardID,
		"user_id":    event.UserID,
	}).Debug("Event published")

	return nil
}

func (rmq *RabbitMQ) ConsumeEvents(handler func(Event) error) error {
	msgs, err := rmq.channel.Consume(
		EventQueue, // queue
		"",         // consumer
		false,      // auto-ack
		false,      // exclusive
		false,      // no-local
		false,      // no-wait
		nil,        // args
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	go func() {
		for d := range msgs {
			var event Event
			if err := json.Unmarshal(d.Body, &event); err != nil {
				rmq.logger.WithError(err).Error("Failed to unmarshal event")
				d.Nack(false, false)
				continue
			}

			if err := handler(event); err != nil {
				rmq.logger.WithError(err).WithFields(logrus.Fields{
					"event_id":   event.ID,
					"event_type": event.Type,
				}).Error("Failed to handle event")
				d.Nack(false, true)
				continue
			}

			d.Ack(false)
			rmq.logger.WithFields(logrus.Fields{
				"event_id":   event.ID,
				"event_type": event.Type,
			}).Debug("Event processed")
		}
	}()

	rmq.logger.Info("Started consuming events")
	return nil
}

func (rmq *RabbitMQ) Close() error {
	if rmq.channel != nil {
		rmq.channel.Close()
	}
	if rmq.conn != nil {
		return rmq.conn.Close()
	}
	return nil
}
