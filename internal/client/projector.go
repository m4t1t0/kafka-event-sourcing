package client

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"

	"github.com/youruser/kafka-event-sourcing/internal/events"
	"github.com/youruser/kafka-event-sourcing/internal/projection"
)

const Topic = "clients"
const ConsumerGroup = "client-projector"

type Projector struct {
	db *gorm.DB
}

func NewProjector(db *gorm.DB) *Projector {
	return &Projector{db: db}
}

// Handle processes a client event and updates the projection atomically
// with the Kafka offset. This is the core of the projection pattern.
func (p *Projector) Handle(ctx context.Context, env *events.Envelope, partition int32, offset int64) error {
	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Apply the projection
		if err := p.apply(tx, env); err != nil {
			return fmt.Errorf("applying event %s: %w", env.EventType, err)
		}

		// 2. Save offset in the same transaction → exactly-once projection
		if err := projection.SaveOffset(tx, ConsumerGroup, Topic, partition, offset); err != nil {
			return fmt.Errorf("saving offset: %w", err)
		}

		log.Printf("[client-projector] Processed %s for %s (partition=%d, offset=%d)",
			env.EventType, env.AggregateID, partition, offset)
		return nil
	})
}

func (p *Projector) apply(tx *gorm.DB, env *events.Envelope) error {
	switch env.EventType {
	case events.ClientCreatedType:
		var e events.ClientCreated
		if err := json.Unmarshal(env.Data, &e); err != nil {
			return err
		}
		return tx.Create(&Client{
			ID:        e.ClientID,
			Name:      e.Name,
			Email:     e.Email,
			Phone:     e.Phone,
			Version:   env.Version,
			CreatedAt: env.OccurredAt,
			UpdatedAt: env.OccurredAt,
		}).Error

	case events.ClientUpdatedType:
		var e events.ClientUpdated
		if err := json.Unmarshal(env.Data, &e); err != nil {
			return err
		}
		return tx.Model(&Client{}).Where("id = ?", e.ClientID).Updates(map[string]any{
			"name":       e.Name,
			"email":      e.Email,
			"phone":      e.Phone,
			"version":    env.Version,
			"updated_at": env.OccurredAt,
		}).Error

	case events.ClientDeletedType:
		var e events.ClientDeleted
		if err := json.Unmarshal(env.Data, &e); err != nil {
			return err
		}
		now := time.Now().UTC()
		return tx.Model(&Client{}).Where("id = ?", e.ClientID).Updates(map[string]any{
			"deleted_at": &now,
			"version":    env.Version,
		}).Error

	default:
		log.Printf("[client-projector] Unknown event type: %s", env.EventType)
		return nil
	}
}
