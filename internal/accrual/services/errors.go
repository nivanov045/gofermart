package services

import (
	"errors"

	"github.com/nivanov045/gofermart/internal/accrual/storages"
)

var (
	ErrIncorrectFormat          = errors.New("request has incorrect format")
	ErrProductAlreadyRegistered = storages.ErrProductAlreadyRegistered
	ErrOrderAlreadyRegistered   = errors.New("order already registered")
)
