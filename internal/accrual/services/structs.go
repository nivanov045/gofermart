package services

const (
	OrderInfoRegistered = "REGISTERED"
	OrderInfoInvalid    = "INVALID"
	OrderInfoProcessing = "PROCESSING"
	OrderInfoProcessed  = "PROCESSED"
)

type orderReward struct {
	ID      string `json:"order"`
	Status  string `json:"status"`
	Accrual int    `json:"accrual,omitempty"`
}

type orderProduct struct {
	Description string `json:"description"`
	Price       int    `json:"price"`
}

type orderList struct {
	ID    string         `json:"order"`
	Goods []orderProduct `json:"goods"`
}
