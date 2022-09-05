package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"gofermart/internal/accrual/log"
	"gofermart/internal/accrual/products"
	"gofermart/internal/accrual/storages"
)

type Service struct {
	storage storages.Storage

	registeredQueue  []orderList
	registeredOrders map[string]bool
	registeredChan   chan orderList

	processingOrders map[string]bool
	processedChan    chan string

	mu sync.RWMutex

	maxWorker int
}

func NewService(storage storages.Storage, maxWorker int) *Service {
	return &Service{
		storage: storage,

		registeredQueue:  make([]orderList, 0),
		registeredOrders: make(map[string]bool),
		registeredChan:   make(chan orderList),

		processingOrders: make(map[string]bool),
		processedChan:    make(chan string),

		maxWorker: maxWorker,
	}
}

func (s *Service) Process(ctx context.Context) {
	workers := 0

	startWorker := func(order orderList) {
		workers++

		s.mu.Lock()
		defer s.mu.Unlock()
		s.processingOrders[order.ID] = true

		go func() {
			err := s.computeAccrual(ctx, order)
			if err != nil {
				log.Error(err)
			}
		}()
	}

	endWorker := func(orderName string) {
		workers--

		s.mu.Lock()
		defer s.mu.Unlock()
		delete(s.processingOrders, orderName)

	}

	registerOrder := func(order orderList) {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.registeredOrders[order.ID] = true
		s.registeredQueue = append(s.registeredQueue, order)
	}

	popOrder := func() *orderList {
		if len(s.registeredQueue) == 0 {
			return nil
		}

		s.mu.Lock()
		defer s.mu.Unlock()

		order := s.registeredQueue[0]
		s.registeredQueue = s.registeredQueue[1:]
		delete(s.registeredOrders, order.ID)
		return &order
	}

	for {
		select {
		case order := <-s.registeredChan:
			if workers < s.maxWorker {
				startWorker(order)
			} else {
				registerOrder(order)
			}
		case orderName := <-s.processedChan:
			endWorker(orderName)
			if nextOrder := popOrder(); nextOrder != nil {
				startWorker(*nextOrder)
			}
		case <-ctx.Done():
			break
		default:
			time.Sleep(time.Millisecond)
		}
	}
}

func (s *Service) computeAccrual(ctx context.Context, order orderList) error {
	defer func() {
		s.processedChan <- order.ID
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
	{
		s.mu.RLock()
		defer s.mu.RUnlock()
		if _, ok := s.registeredOrders[id]; ok {
			return json.Marshal(orderReward{ID: id, Status: OrderInfoRegistered})
		}
		if _, ok := s.processingOrders[id]; ok {
			return json.Marshal(orderReward{ID: id, Status: OrderInfoProcessing})
		}
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
	if order.ID == "" {
		return ErrIncorrectFormat
	}

	s.registeredChan <- order
	return nil
}

func (s *Service) RegisterProduct(ctx context.Context, request []byte) error {
	var product products.Product
	err := json.Unmarshal(request, &product)
	if err != nil {
		var errUnknownType products.UnknownTypeError
		if errors.As(err, &errUnknownType) {
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
