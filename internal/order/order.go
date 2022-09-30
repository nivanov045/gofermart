package order

import "time"

const (
	ProcessingTypeNew        string = "NEW"
	ProcessingTypeProcessing string = "PROCESSING"
	ProcessingTypeInvalid    string = "INVALID"
	ProcessingTypeProcessed  string = "PROCESSED"
)

type InterfaceForAccrualSystem struct {
	Number  string  `json:"order"`
	Status  string  `json:"status"`
	Accrual float64 `json:"accrual,omitempty"`
}

type Interface struct {
	Number     string    `json:"number"`
	Status     string    `json:"status"`
	Accrual    float64   `json:"accrual,omitempty"` // accrual for processed orders only
	UploadedAt time.Time `json:"uploaded_at,omitempty"`
}

type Order struct {
	Number     string
	Status     string
	Accrual    int64
	UploadedAt time.Time
}
