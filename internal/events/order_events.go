package events

const (
	OrderPlacedType    = "order.placed"
	OrderConfirmedType = "order.confirmed"
	OrderCancelledType = "order.cancelled"
)

type OrderItem struct {
	ProductID string  `json:"product_id"`
	Name      string  `json:"name"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

type OrderPlaced struct {
	OrderID  string      `json:"order_id"`
	ClientID string      `json:"client_id"`
	Items    []OrderItem `json:"items"`
	Total    float64     `json:"total"`
}

type OrderConfirmed struct {
	OrderID     string `json:"order_id"`
	ConfirmedBy string `json:"confirmed_by"`
}

type OrderCancelled struct {
	OrderID string `json:"order_id"`
	Reason  string `json:"reason"`
}
