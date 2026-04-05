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

const Topic = "orders"
const ConsumerGroup = "order-projector"

type Projector struct {
	db *gorm.DB
}

func NewProjector(db *gorm.DB) *Projector {
	return &Projector{db: db}
}

// Handle processes an order event and updates the projection atomically
// with the Kafka offset.
func (p *Projector) Handle(ctx context.Context, env *events.Envelope, partition int32, offset int64) error {
	return p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := p.apply(tx, env); err != nil {
			return fmt.Errorf("applying event %s: %w", env.EventType, err)
		}

		if err := projection.SaveOffset(tx, ConsumerGroup, Topic, partition, offset); err != nil {
			return fmt.Errorf("saving offset: %w", err)
		}

		log.Printf("[order-projector] Processed %s for %s (partition=%d, offset=%d)",
			env.EventType, env.AggregateID, partition, offset)
		return nil
	})
}

func (p *Projector) apply(tx *gorm.DB, env *events.Envelope) error {
	switch env.EventType {
	case events.OrderPlacedType:
		var e events.OrderPlaced
		if err := json.Unmarshal(env.Data, &e); err != nil {
			return err
		}

		items := make([]OrderItem, len(e.Items))
		for i, item := range e.Items {
			items[i] = OrderItem{
				OrderID:   e.OrderID,
				ProductID: item.ProductID,
				Name:      item.Name,
				Quantity:  item.Quantity,
				Price:     item.Price,
			}
		}

		return tx.Create(&Order{
			ID:        e.OrderID,
			ClientID:  e.ClientID,
			Status:    "placed",
			Total:     e.Total,
			Version:   env.Version,
			CreatedAt: env.OccurredAt,
			UpdatedAt: env.OccurredAt,
			Items:     items,
		}).Error

	case events.OrderConfirmedType:
		var e events.OrderConfirmed
		if err := json.Unmarshal(env.Data, &e); err != nil {
			return err
		}
		return tx.Model(&Order{}).Where("id = ?", e.OrderID).Updates(map[string]any{
			"status":       "confirmed",
			"confirmed_by": e.ConfirmedBy,
			"version":      env.Version,
			"updated_at":   env.OccurredAt,
		}).Error

	case events.OrderCancelledType:
		var e events.OrderCancelled
		if err := json.Unmarshal(env.Data, &e); err != nil {
			return err
		}
		return tx.Model(&Order{}).Where("id = ?", e.OrderID).Updates(map[string]any{
			"status":        "cancelled",
			"cancel_reason": e.Reason,
			"version":       env.Version,
			"updated_at":    env.OccurredAt,
		}).Error

	default:
		log.Printf("[order-projector] Unknown event type: %s", env.EventType)
		return nil
	}
}
