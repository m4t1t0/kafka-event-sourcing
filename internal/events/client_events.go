package events

const (
	ClientCreatedType = "client.created"
	ClientUpdatedType = "client.updated"
	ClientDeletedType = "client.deleted"
)

type ClientCreated struct {
	ClientID string `json:"client_id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
}

type ClientUpdated struct {
	ClientID string `json:"client_id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
}

type ClientDeleted struct {
	ClientID string `json:"client_id"`
}
