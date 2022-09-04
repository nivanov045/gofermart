package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gofermart/internal/accrual/log"
	"gofermart/internal/accrual/products"
	"gofermart/internal/accrual/storages"
)

type Service struct {
	storage storages.Storage

	queue            []orderList
	registeredOrders map[string]bool
	processingOrders map[string]bool

	regChan       chan orderList
	completedChan chan string

	maxWorker int
}

func NewService(storage storages.Storage, maxWorker int) *Service {
	return &Service{
		storage: storage,
		queue:   make([]orderList, 0),

		registeredOrders: make(map[string]bool),
		processingOrders: make(map[string]bool),

		regChan:       make(chan orderList),
		completedChan: make(chan string),

		maxWorker: maxWorker,
	}
}

func (s *Service) Process(ctx context.Context) {
	workers := 0

	startWorker := func(order orderList) {
		workers++
		delete(s.registeredOrders, order.ID)
		s.processingOrders[order.ID] = true
		go func() {
			err := s.ComputeAccrual(ctx, order)
			if err != nil {
				log.Error(err)
			}
		}()
	}

	for {
		select {
		case order := <-s.regChan:
			if workers < s.maxWorker {
				startWorker(order)
			} else {
				s.registeredOrders[order.ID] = true
				s.queue = append(s.queue, order)
			}
		case orderName := <-s.completedChan:
			workers--
			delete(s.processingOrders, orderName)

			if len(s.queue) > 0 {
				order := s.queue[0]
				s.queue = s.queue[1:]
				startWorker(order)
			}
		case <-ctx.Done():
			break
		default:
			time.Sleep(time.Millisecond)
		}
	}
}

func (s *Service) ComputeAccrual(ctx context.Context, order orderList) error {
	defer func() {
		s.completedChan <- order.ID
	}()

	accrual := 0
	for _, orderProduct := range order.Goods {
		product, err := s.storage.GetProduct(ctx, orderProduct.Description)
		if errors.Is(err, storages.ErrProductNotFound) {
			continue
		}
		if err != nil {
			return err
		}

		switch product.RewardType {
		case products.RewardTypePoints:
			accrual += product.Reward
			break
		case products.RewardTypePercent:
			accrual += int(0.01 * float64(product.Reward) * float64(orderProduct.Price))
			break
		default:
			return fmt.Errorf("unknown reward type: '%v'", product.RewardType)
		}
	}

	err := s.storage.StoreOrder(ctx, order.ID, accrual)
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) GetOrderStatus(ctx context.Context, id string) ([]byte, error) {
	if _, ok := s.registeredOrders[id]; ok {
		return json.Marshal(orderReward{ID: id, Status: OrderInfoRegistered})
	}
	if _, ok := s.processingOrders[id]; ok {
		return json.Marshal(orderReward{ID: id, Status: OrderInfoProcessing})
	}

	accrual, err := s.storage.GetOrder(ctx, id)
	if err != nil {
		if errors.Is(err, storages.ErrOrderNotFound) {
			return json.Marshal(orderReward{ID: id, Status: OrderInfoInvalid})
		}
		return nil, err
	}

	return json.Marshal(orderReward{ID: id, Status: OrderInfoProcessed, Accrual: accrual})
}

func (s *Service) RegisterOrder(_ context.Context, request []byte) error {
	var order orderList
	err := json.Unmarshal(request, &order)
	if err != nil {
		return err
	}

	// TODO: Check with Luhn algorithm
	if order.ID != "" {
		return ErrIncorrectFormat
	}

	s.regChan <- order
	return nil
}

func (s *Service) RegisterProduct(ctx context.Context, request []byte) error {
	var product products.Product
	err := json.Unmarshal(request, &product)
	if err != nil {
		e := products.UnknownTypeError{}
		if errors.As(err, &e) {
			return ErrIncorrectFormat
		}
		return err
	}

	if product.Match == "" {
		return ErrIncorrectFormat
	}

	err = s.storage.RegisterProduct(ctx, product.Match, product.Reward, product.RewardType)
	if err != nil {
		return err
	}

	return nil
}
