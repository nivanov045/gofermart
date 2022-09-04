package storages

import "errors"

var (
	ErrOrderNotFound            = errors.New("order not found")
	ErrProductNotFound          = errors.New("product not found")
	ErrProductAlreadyRegistered = errors.New("product already registered")
)
