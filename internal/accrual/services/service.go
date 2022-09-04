package services

import (
	"context"
	"encoding/json"

	"gofermart/internal/accrual/products"
	"gofermart/internal/accrual/storages"
)

type Service struct {
	storage storages.Storage
}

func NewService(storage storages.Storage) *Service {
	return &Service{storage: storage}
}

func (s *Service) RegisterProduct(ctx context.Context, request []byte) error {
	var product products.Product
	err := json.Unmarshal(request, &product)
	if err != nil {
		return err
	}

	err = s.storage.RegisterProduct(ctx, product.Match, product.Reward, product.RewardType)
	if err != nil {
		return err
	}

	return nil
}
