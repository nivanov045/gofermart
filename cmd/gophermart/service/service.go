package service

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strconv"
	"time"

	"github.com/nivanov045/gofermart/internal/checksums"
	"github.com/nivanov045/gofermart/internal/order"
	"github.com/nivanov045/gofermart/internal/withdraw"
)

type Storage interface {
	FindOrderByUser(login string, number string) (bool, error)
	FindOrder(number string) (bool, error)
	AddOrder(login string, number string) error
	UpdateOrder(order2 order.Order) error
	GetOrders(login string) ([]order.Order, error)
	MakeWithdraw(login string, order string, sum int64) error
	GetWithdraws(login string) ([]withdraw.Withdraw, error)
}

type AccrualSystem interface {
	SetChannelToResponseToService(chan order.Order)
	RunListenToService(<-chan string)
}

type service struct {
	storage           Storage
	isDebug           bool
	accrualSystem     AccrualSystem
	toAccrualSystem   chan string
	fromAccrualSystem chan order.Order
}

func New(storage Storage, accrualSystem AccrualSystem, isDebug bool) *service {
	resultService := &service{
		storage:           storage,
		isDebug:           isDebug,
		accrualSystem:     accrualSystem,
		toAccrualSystem:   make(chan string),
		fromAccrualSystem: make(chan order.Order),
	}
	resultService.accrualSystem.SetChannelToResponseToService(resultService.fromAccrualSystem)
	go resultService.RunListenToAccrual()
	go resultService.accrualSystem.RunListenToService(resultService.toAccrualSystem)
	return resultService
}

func (s *service) RunListenToAccrual() {
	ctx := context.Background()
	for {
		select {
		case <-ctx.Done():
			return
		case ord := <-s.fromAccrualSystem:
			log.Println("service::RunListenToAccrual::info: received value")
			err := s.storage.UpdateOrder(ord)
			if err != nil {
				log.Println("service::RunListenToAccrual::error:", err)
			}
		default:
			time.Sleep(1 * time.Second)
		}
	}
}

func (s *service) checkOrderNumber(orderNumber string) bool {
	if s.isDebug {
		return true
	}
	n, err := strconv.ParseInt(orderNumber, 10, 64)
	if err != nil {
		return false
	}
	return checksums.Luhn(n)
}

// AddOrder returns true if order didn't exist before call, false if existed
func (s *service) AddOrder(login string, requestBody []byte) (bool, error) {
	orderNumber := string(requestBody)
	if !s.checkOrderNumber(orderNumber) {
		return true, errors.New("wrong format of order")
	}
	isExists, err := s.storage.FindOrderByUser(login, orderNumber)
	if err != nil || isExists {
		return false, err
	}
	isExists, err = s.storage.FindOrder(orderNumber)
	if err != nil {
		return false, err
	}
	if isExists {
		return false, errors.New("order was uploaded by another user")
	}
	err = s.storage.AddOrder(login, orderNumber)
	if err != nil {
		return true, err
	}
	s.toAccrualSystem <- orderNumber
	return true, nil
}

func (s *service) GetOrders(login string) ([]byte, error) {
	orders, err := s.storage.GetOrders(login)
	if err != nil {
		return nil, err
	}
	if len(orders) == 0 {
		return nil, errors.New("no orders")
	}
	var ordersToResponse []order.Interface
	for _, ord := range orders {
		ordersToResponse = append(ordersToResponse, order.Interface{
			Number:     ord.Number,
			Status:     ord.Status,
			Accrual:    float64(ord.Accrual) / 100,
			UploadedAt: ord.UploadedAt,
		})
	}
	marshal, err := json.Marshal(ordersToResponse)
	if err != nil {
		return nil, err
	}
	return marshal, nil
}

func (s *service) calculateBalance(login string) (current int64, withdrawn int64, err error) {
	current = 0
	withdrawn = 0
	err = nil
	withdraws, err := s.storage.GetWithdraws(login)
	if err != nil {
		return current, withdrawn, err
	}
	orders, err := s.storage.GetOrders(login)
	if err != nil {
		return current, withdrawn, err
	}
	for _, w := range withdraws {
		withdrawn += w.Sum
	}
	for _, o := range orders {
		current += o.Accrual
	}
	current -= withdrawn
	return current, withdrawn, err
}

func (s *service) GetBalance(login string) ([]byte, error) {
	current, withdrawn, err := s.calculateBalance(login)
	if err != nil {
		return nil, err
	}
	type balance struct {
		Current   float64 `json:"current"`
		Withdrawn float64 `json:"withdrawn"`
	}
	bal := balance{
		Current:   float64(current) / 100,
		Withdrawn: float64(withdrawn) / 100,
	}
	marshal, err := json.Marshal(bal)
	if err != nil {
		return nil, err
	}
	return marshal, nil
}

func (s *service) MakeWithdraw(login string, requestBody []byte) error {
	type request struct {
		Order string  `json:"order"`
		Sum   float64 `json:"sum"`
	}
	var currentRequest request
	err := json.Unmarshal(requestBody, &currentRequest)
	if err != nil {
		return errors.New("wrong query")
	}
	isOrderOk := s.checkOrderNumber(currentRequest.Order)
	if !isOrderOk {
		return errors.New("wrong format of order")
	}
	current, _, err := s.calculateBalance(login)
	if err != nil {
		return err
	}
	sumFromRequest := int64(currentRequest.Sum * 100)
	if current < sumFromRequest {
		return errors.New("not enough balance")
	}
	err = s.storage.MakeWithdraw(login, currentRequest.Order, sumFromRequest)
	return err

}

func (s *service) GetWithdraws(login string) ([]byte, error) {
	withdraws, err := s.storage.GetWithdraws(login)
	if err != nil {
		return nil, err
	}
	if len(withdraws) == 0 {
		return nil, errors.New("no withdraws")
	}
	var resutlWithdrawInterface []withdraw.Interface
	for _, w := range withdraws {
		el := withdraw.Interface{
			Order:       w.Order,
			Sum:         float64(w.Sum) / 100,
			ProcessedAt: w.ProcessedAt,
		}
		resutlWithdrawInterface = append(resutlWithdrawInterface, el)
	}
	marshal, err := json.Marshal(resutlWithdrawInterface)
	if err != nil {
		return nil, err
	}
	return marshal, nil
}
