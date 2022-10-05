package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/nivanov045/gofermart/internal/accrual/log"
	"github.com/nivanov045/gofermart/internal/accrual/models"
	"github.com/nivanov045/gofermart/internal/accrual/storages"
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

func (s *Service) Process(ctx context.Context) {
	fanOutChs := fanOut(s.orderQueueCh, s.maxWorker)

	workerChs := make([]chan accrualResult, 0, s.maxWorker)
	for _, fanOutCh := range fanOutChs {
		workerCh := make(chan accrualResult)
		s.newWorker(ctx, fanOutCh, workerCh)
		workerChs = append(workerChs, workerCh)
	}

	// TODO: Mode to separate func
	for resultAccrual := range fanIn(workerChs...) {
		if resultAccrual.err != nil {
			// TODO: What we have to do with failed computation? Set to invalid status?
			log.Error(resultAccrual.err)
			continue
		}

		err := s.storage.UpdateOrderStatus(ctx, resultAccrual.id, models.OrderStatus{Status: models.OrderStatusProcessed, Accrual: resultAccrual.accrual})
		if err != nil {
			log.Error(err)

			err := s.storage.UpdateOrderStatus(ctx, resultAccrual.id, models.OrderStatus{Status: models.OrderStatusRegistered, Accrual: 0})
			if err != nil {
				log.Error(err)
			}
			continue
		}
		log.Debug(fmt.Sprintf("Order '%v' processed", resultAccrual.id))

		err = s.queue.RemoveOrder(ctx, resultAccrual.id)
		if err != nil {
			log.Error(err)
		}
	}
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

	// TODO: Check with Luhn algorithm
	if order.ID == "" {
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
		var errUnknownType models.UnknownTypeError
		if errors.As(err, &errUnknownType) {
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

	return nil
}
