package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/nivanov045/gofermart/internal/accrual/log"
	"github.com/nivanov045/gofermart/internal/accrual/models"
	"github.com/nivanov045/gofermart/internal/accrual/storages"
	"github.com/nivanov045/gofermart/internal/checksums"
)

type Service struct {
	storage storages.Storage
	queue   storages.OrderQueue

	orderQueueCh chan models.OrderList
	processedCh  chan string

	maxWorker int
}

type accrualResult struct {
	id      string
	accrual float64
	err     error
}

func NewService(storage storages.Storage, queue storages.OrderQueue, maxWorker int) *Service {
	return &Service{
		storage: storage,
		queue:   queue,

		orderQueueCh: make(chan models.OrderList),

		maxWorker: maxWorker,
	}
}

func (s *Service) Run(ctx context.Context) {
	fanOutChs := fanOut(s.orderQueueCh, s.maxWorker)

	workerChs := make([]chan accrualResult, 0, s.maxWorker)
	for _, fanOutCh := range fanOutChs {
		workerCh := make(chan accrualResult)
		s.newWorker(ctx, fanOutCh, workerCh)
		workerChs = append(workerChs, workerCh)
	}

	s.process(ctx, workerChs)
}

func (s *Service) GetOrderReward(ctx context.Context, id string) ([]byte, error) {
	orderStatus, err := s.storage.GetOrderStatus(ctx, id)
	if err != nil {
		return nil, err
	}

	var orderReward models.OrderReward
	if orderStatus.Status == models.OrderStatusProcessed {
		orderReward = models.OrderReward{ID: id, Accrual: orderStatus.Accrual, Status: models.OrderStatusText(orderStatus.Status)}
	} else {
		orderReward = models.OrderReward{ID: id, Status: models.OrderStatusText(orderStatus.Status)}
	}

	return json.Marshal(orderReward)
}

func (s *Service) RegisterOrder(ctx context.Context, request []byte) error {
	var order models.OrderList
	err := json.Unmarshal(request, &order)
	if err != nil {
		return err
	}

	if order.ID == "" {
		return ErrIncorrectFormat
	} else if id, err := strconv.ParseInt(order.ID, 10, 64); err != nil || !checksums.Luhn(id) {
		return ErrIncorrectFormat
	}

	orderStatus, err := s.storage.GetOrderStatus(ctx, order.ID)
	if err != nil && !errors.Is(err, storages.ErrOrderNotFound) {
		return err
	}
	if orderStatus.Status != models.OrderStatusInvalid {
		return ErrOrderAlreadyRegistered
	}

	err = s.queue.RegisterOrder(ctx, order.ID, request)
	if err != nil {
		return err
	}

	log.Debug(fmt.Sprintf("Order '%v' registered", order.ID))
	go func() {
		s.orderQueueCh <- order
	}()

	return nil
}

func (s *Service) RegisterProduct(ctx context.Context, request []byte) error {
	var product models.Product
	err := json.Unmarshal(request, &product)
	if err != nil {
		var errUnknownType *models.UnknownRewardTypeError
		if errors.As(err, &errUnknownType) {
			return ErrIncorrectFormat
		}
		if errors.Is(err, models.ErrIncorrectRewardValue) {
			return ErrIncorrectFormat
		}
		return err
	}

	if product.Match == "" {
		return ErrIncorrectFormat
	}

	err = s.storage.RegisterProduct(ctx, product)
	if err != nil {
		return err
	}
	log.Debug(fmt.Sprintf("Product '%v' '%v'('%v') registered", product.Match, product.Reward, product.RewardType))

	return nil
}
