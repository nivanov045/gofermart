package service

import (
	"encoding/json"
	"errors"

	"github.com/nivanov045/gofermart/internal/order"
	"github.com/nivanov045/gofermart/internal/withdraw"
)

type Storage interface {
	FindOrderByUser(login string, number string) (bool, error)
	FindOrder(number string) (bool, error)
	AddOrder(login string, number string) error
	GetOrders(login string) ([]order.Order, error)
	MakeWithdraw(login string, order string, sum int64) error
	GetWithdraws(login string) ([]withdraw.Withdraw, error)
}

type service struct {
	storage Storage
	isDebug bool
}

func New(storage Storage, isDebug bool) *service {
	return &service{storage: storage, isDebug: isDebug}
}

func checkOrderNumber(orderNumber string) bool {
	// TODO: Realise algorithm
	return true
}

// AddOrder returns true if order didn't exist before call, false if existed
func (s *service) AddOrder(login string, requestBody []byte) (bool, error) {
	orderNumber := string(requestBody)
	if !checkOrderNumber(orderNumber) {
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
	return true, err
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
	//TODO: find all finished orders and withdraws and calculate balance
	return 0, 0, nil
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
	isExists, err := s.storage.FindOrderByUser(login, currentRequest.Order)
	if err != nil {
		return err
	}
	if !isExists {
		return errors.New("no such order")
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
	marshal, err := json.Marshal(withdraws)
	if err != nil {
		return nil, err
	}
	return marshal, nil
}
