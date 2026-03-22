package kafka

import (
	"fmt"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"

	"github.com/youruser/kafka-event-sourcing/internal/events"
)

type Producer struct {
	producer *kafka.Producer
}

func NewProducer(broker string) (*Producer, error) {
	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": broker,
		"acks":              "all",
		"retries":           3,
	})
	if err != nil {
		return nil, fmt.Errorf("creating producer: %w", err)
	}

	// Drain delivery reports in background
	go func() {
		for range p.Events() {
		}
	}()

	return &Producer{producer: p}, nil
}

// Publish sends an event envelope to the given topic, keyed by aggregate ID.
func (p *Producer) Publish(topic string, env *events.Envelope) error {
	value, err := env.Serialize()
	if err != nil {
		return fmt.Errorf("serializing envelope: %w", err)
	}

	return p.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &topic,
			Partition: kafka.PartitionAny,
		},
		Key:   []byte(env.AggregateID),
		Value: value,
	}, nil)
}

// Flush waits for all messages to be delivered.
func (p *Producer) Flush(timeoutMs int) {
	p.producer.Flush(timeoutMs)
}

// Close shuts down the producer.
func (p *Producer) Close() {
	p.producer.Flush(5000)
	p.producer.Close()
}
