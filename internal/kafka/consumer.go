package kafka

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"

	"github.com/youruser/kafka-event-sourcing/internal/events"
)

// MessageHandler processes a deserialized event envelope.
// The partition and offset are passed so the handler can store them atomically.
type MessageHandler func(ctx context.Context, env *events.Envelope, partition int32, offset int64) error

type Consumer struct {
	consumer *kafka.Consumer
}

func NewConsumer(broker, groupID string) (*Consumer, error) {
	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers":  broker,
		"group.id":           groupID,
		"auto.offset.reset":  "earliest",
		"enable.auto.commit": false, // We commit manually after projection
	})
	if err != nil {
		return nil, fmt.Errorf("creating consumer: %w", err)
	}
	return &Consumer{consumer: c}, nil
}

// Subscribe starts consuming from the given topics and calls handler for each message.
func (c *Consumer) Subscribe(ctx context.Context, topics []string, handler MessageHandler) error {
	if err := c.consumer.SubscribeTopics(topics, nil); err != nil {
		return fmt.Errorf("subscribing: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			log.Println("Consumer shutting down...")
			c.consumer.Close()
			return nil
		default:
			msg, err := c.consumer.ReadMessage(1 * time.Second)
			if err != nil {
				// Timeout is normal, just continue
				if kafkaErr, ok := err.(kafka.Error); ok && kafkaErr.Code() == kafka.ErrTimedOut {
					continue
				}
				log.Printf("Consumer read error: %v", err)
				continue
			}

			env, err := events.DeserializeEnvelope(msg.Value)
			if err != nil {
				log.Printf("Failed to deserialize event: %v", err)
				continue
			}

			if err := handler(ctx, env, msg.TopicPartition.Partition, int64(msg.TopicPartition.Offset)); err != nil {
				log.Printf("Handler error for event %s: %v", env.EventType, err)
				continue
			}

			// Commit offset only after successful projection
			if _, err := c.consumer.CommitMessage(msg); err != nil {
				log.Printf("Failed to commit offset: %v", err)
			}
		}
	}
}

func (c *Consumer) Close() {
	c.consumer.Close()
}
