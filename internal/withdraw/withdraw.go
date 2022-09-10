package withdraw

import "time"

type Withdraw struct {
	Order       string    `json:"order"`
	Sum         int64     `json:"sum"`
	ProcessedAt time.Time `json:"processed_at"`
}
