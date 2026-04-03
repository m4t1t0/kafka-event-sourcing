package order

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/youruser/kafka-event-sourcing/internal/events"
	k "github.com/youruser/kafka-event-sourcing/internal/kafka"
)

type Handler struct {
	db       *gorm.DB
	producer *k.Producer
}

func NewHandler(db *gorm.DB, producer *k.Producer) *Handler {
	return &Handler{db: db, producer: producer}
}

// --- Command side ---

type PlaceOrderRequest struct {
	ClientID string              `json:"client_id"`
	Items    []events.OrderItem  `json:"items"`
}

func (h *Handler) Place(w http.ResponseWriter, r *http.Request) {
	var req PlaceOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if req.ClientID == "" || len(req.Items) == 0 {
		http.Error(w, "client_id and items are required", http.StatusBadRequest)
		return
	}

	orderID := uuid.New().String()

	var total float64
	for _, item := range req.Items {
		total += float64(item.Quantity) * item.Price
	}

	env, err := events.NewEnvelope(events.OrderPlacedType, orderID, 1, events.OrderPlaced{
		OrderID:  orderID,
		ClientID: req.ClientID,
		Items:    req.Items,
		Total:    total,
	})
	if err != nil {
		http.Error(w, "failed to create event", http.StatusInternalServerError)
		return
	}

	if err := h.producer.Publish(Topic, env); err != nil {
		http.Error(w, "failed to publish event", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"order_id": orderID,
		"status":   "event_published",
	})
}

type ConfirmOrderRequest struct {
	ConfirmedBy string `json:"confirmed_by"`
}

func (h *Handler) Confirm(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req ConfirmOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	env, err := events.NewEnvelope(events.OrderConfirmedType, id, 0, events.OrderConfirmed{
		OrderID:     id,
		ConfirmedBy: req.ConfirmedBy,
	})
	if err != nil {
		http.Error(w, "failed to create event", http.StatusInternalServerError)
		return
	}

	if err := h.producer.Publish(Topic, env); err != nil {
		http.Error(w, "failed to publish event", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "event_published"})
}

type CancelOrderRequest struct {
	Reason string `json:"reason"`
}

func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req CancelOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	env, err := events.NewEnvelope(events.OrderCancelledType, id, 0, events.OrderCancelled{
		OrderID: id,
		Reason:  req.Reason,
	})
	if err != nil {
		http.Error(w, "failed to create event", http.StatusInternalServerError)
		return
	}

	if err := h.producer.Publish(Topic, env); err != nil {
		http.Error(w, "failed to publish event", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "event_published"})
}

// --- Query side ---

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var o Order
	if err := h.db.Preload("Items").Where("id = ?", id).First(&o).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "order not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(o)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	var orders []Order
	if err := h.db.Preload("Items").Find(&orders).Error; err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}
