package authenticator

type Storage interface {
}

type authenticator struct {
	storage Storage
}

func New(storage Storage) *authenticator {
	return &authenticator{storage: storage}
}

func (a authenticator) CheckAuthentication(bytes []byte) bool {
	//TODO implement me
	panic("implement me")
}

func (a authenticator) Register(bytes []byte) error {
	//TODO implement me
	panic("implement me")
}

func (a authenticator) Login(bytes []byte) error {
	//TODO implement me
	panic("implement me")
}
