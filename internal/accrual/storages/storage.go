package storages

import (
	"context"

	"gofermart/internal/accrual/products"
)

type Storage interface {
	RegisterProduct(ctx context.Context, name string, reward int, rewardType products.RewardType) error
}
