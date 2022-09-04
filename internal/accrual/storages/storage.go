package storages

import (
	"context"

	"gofermart/internal/accrual/products"
)

type Storage interface {
	GetOrder(ctx context.Context, id string) (int, error)
	StoreOrder(ctx context.Context, id string, accrual int) error

	GetProduct(ctx context.Context, name string) (*products.Product, error)
	RegisterProduct(ctx context.Context, name string, reward int, rewardType products.RewardType) error
}
