package storages

import (
	"context"
	"errors"

	"gofermart/internal/accrual/products"
)

var (
	ErrOrderNotFound   = errors.New("order not found")
	ErrProductNotFound = errors.New("product not found")
)

type Storage interface {
	GetOrder(ctx context.Context, id string) (int, error)
	StoreOrder(ctx context.Context, id string, accrual int) error

	GetProduct(ctx context.Context, name string) (*products.Product, error)
	RegisterProduct(ctx context.Context, name string, reward int, rewardType products.RewardType) error
}
