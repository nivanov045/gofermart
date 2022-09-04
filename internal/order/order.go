package order

import "time"

const (
	ProcessingTypeNew        string = "NEW"
	ProcessingTypeProcessing        = "PROCESSING"
	ProcessingTypeInvalid           = "INVALID"
	ProcessingTypeProcessed         = "PROCESSED"
)

type Order struct {
	Number     string    `json:"number"`
	Status     string    `json:"status"`
	Accrual    float64   `json:"accrual,omitempty"` // accrual for processed orders only
	UploadedAt time.Time `json:"uploaded_at,omitempty"`
}
