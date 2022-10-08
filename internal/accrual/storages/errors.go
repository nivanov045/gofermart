package storages

import "errors"

var (
	ErrOrderNotFound            = errors.New("order not found")
	ErrProductAlreadyRegistered = errors.New("product already registered")
)
