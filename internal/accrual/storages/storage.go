package storages

import (
	"context"

	"github.com/nivanov045/gofermart/internal/accrual/models"
)

type Storage interface {
	GetOrderStatus(ctx context.Context, id string) (models.OrderStatus, error)
	UpdateOrderStatus(ctx context.Context, id string, orderStatus models.OrderStatus) error

	MatchProducts(ctx context.Context, description string) ([]models.Product, error)
	RegisterProduct(ctx context.Context, product models.Product) error
}

type OrderQueue interface {
	GetAllOrders(ctx context.Context) ([][]byte, error)
	RemoveOrder(ctx context.Context, id string) error
	RegisterOrder(ctx context.Context, id string, orderInfo []byte) error
}
