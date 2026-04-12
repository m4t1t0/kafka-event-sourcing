package client

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

// --- Command side: accept requests, publish events ---

type CreateClientRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateClientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	clientID := uuid.New().String()

	env, err := events.NewEnvelope(events.ClientCreatedType, clientID, 1, events.ClientCreated{
		ClientID: clientID,
		Name:     req.Name,
		Email:    req.Email,
		Phone:    req.Phone,
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
		"client_id": clientID,
		"status":    "event_published",
	})
}

type UpdateClientRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req UpdateClientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	env, err := events.NewEnvelope(events.ClientUpdatedType, id, 0, events.ClientUpdated{
		ClientID: id,
		Name:     req.Name,
		Email:    req.Email,
		Phone:    req.Phone,
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

// --- Query side: read from projection ---

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id") // Go 1.22+ net/http routing

	var client Client
	if err := h.db.Where("id = ? AND deleted_at IS NULL", id).First(&client).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "client not found", http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(client)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	var clients []Client
	if err := h.db.Where("deleted_at IS NULL").Find(&clients).Error; err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(clients)
}
