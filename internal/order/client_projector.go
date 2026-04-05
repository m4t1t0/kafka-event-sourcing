package order

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"gorm.io/gorm"

	"github.com/youruser/kafka-event-sourcing/internal/events"
	"github.com/youruser/kafka-event-sourcing/internal/projection"
)

const ClientTopic = "clients"
const ClientConsumerGroup = "order-client-projector"

// ClientProjector consumes client events to denormalize client_name into orders.
type ClientProjector struct {
	db *gorm.DB
}

func NewClientProjector(db *gorm.DB) *ClientProjector {
	return &ClientProjector{db: db}
}

func (p *ClientProjector) Handle(ctx context.Context, env *events.Envelope, partition int32, offset int64) error {
	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := p.apply(tx, env); err != nil {
			return fmt.Errorf("applying client event %s: %w", env.EventType, err)
		}

		if err := projection.SaveOffset(tx, ClientConsumerGroup, ClientTopic, partition, offset); err != nil {
			return fmt.Errorf("saving offset: %w", err)
		}

		log.Printf("[order-client-projector] Processed %s for %s (partition=%d, offset=%d)",
			env.EventType, env.AggregateID, partition, offset)
		return nil
	})
}

func (p *ClientProjector) apply(tx *gorm.DB, env *events.Envelope) error {
	switch env.EventType {
	case events.ClientCreatedType, events.ClientUpdatedType:
		var e events.ClientCreated
		if err := json.Unmarshal(env.Data, &e); err != nil {
			return err
		}
		// Update client_name on all orders belonging to this client
		return tx.Model(&Order{}).Where("client_id = ?", e.ClientID).
			Update("client_name", e.Name).Error

	default:
		// client.deleted and unknown types — no action needed on orders
		return nil
	}
}
