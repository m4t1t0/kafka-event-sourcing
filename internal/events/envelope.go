package events

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Envelope wraps every domain event with metadata.
// This is what gets serialized into Kafka.
type Envelope struct {
	EventID     string          `json:"event_id"`
	EventType   string          `json:"event_type"`
	AggregateID string          `json:"aggregate_id"`
	OccurredAt  time.Time       `json:"occurred_at"`
	Version     int             `json:"version"`
	Data        json.RawMessage `json:"data"`
}

// NewEnvelope creates an envelope for a domain event.
func NewEnvelope(eventType, aggregateID string, version int, data any) (*Envelope, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return &Envelope{
		EventID:     uuid.New().String(),
		EventType:   eventType,
		AggregateID: aggregateID,
		OccurredAt:  time.Now().UTC(),
		Version:     version,
		Data:        raw,
	}, nil
}

// Serialize returns the JSON bytes of the full envelope.
func (e *Envelope) Serialize() ([]byte, error) {
	return json.Marshal(e)
}

// DeserializeEnvelope parses raw Kafka message bytes into an Envelope.
func DeserializeEnvelope(data []byte) (*Envelope, error) {
	var env Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, err
	}
	return &env, nil
}
