package models

type OrderStatusCode int

const (
	OrderStatusRegistered OrderStatusCode = iota
	OrderStatusInvalid
	OrderStatusProcessing
	OrderStatusProcessed
)

func OrderStatusText(status OrderStatusCode) string {
	switch status {
	case OrderStatusRegistered:
		return "REGISTERED"
	case OrderStatusInvalid:
		return "INVALID"
	case OrderStatusProcessing:
		return "PROCESSING"
	case OrderStatusProcessed:
		return "PROCESSED"
	}

	return "UNKNOWN_TYPE"
}

type OrderStatus struct {
	Status  OrderStatusCode
	Accrual float64
}

type OrderReward struct {
	ID      string  `json:"order"`
	Status  string  `json:"status"`
	Accrual float64 `json:"accrual,omitempty"`
}

type OrderProduct struct {
	Description string  `json:"description"`
	Price       float64 `json:"price"`
}

type OrderList struct {
	ID    string         `json:"order"`
	Goods []OrderProduct `json:"goods"`
}
