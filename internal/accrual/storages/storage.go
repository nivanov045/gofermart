package storages

import (
	"context"

	"gofermart/internal/accrual/models"
)

type Storage interface {
	GetOrderStatus(ctx context.Context, id string) (models.OrderStatus, error)
	UpdateOrderStatus(ctx context.Context, id string, orderStatus models.OrderStatus) error

	GetProduct(ctx context.Context, name string) (*models.Product, error)
	RegisterProduct(ctx context.Context, name string, reward int, rewardType models.RewardType) error
}

type OrderQueue interface {
	GetOrder(ctx context.Context) ([]byte, error)
	RemoveOrder(ctx context.Context, id string) error
	RegisterOrder(ctx context.Context, id string, orderInfo []byte) error
}