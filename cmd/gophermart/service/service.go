package service

type Storage interface {
}

type service struct {
	storage Storage
}

func New(storage Storage) *service {
	return &service{storage: storage}
}

func (s service) AddOrder(bytes []byte) error {
	//TODO implement me
	panic("implement me")
}

func (s service) GetOrders(bytes []byte) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (s service) GetBalance(bytes []byte) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (s service) MakeWithdraw(bytes []byte) error {
	//TODO implement me
	panic("implement me")
}

func (s service) GetWithdraws(bytes []byte) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}
