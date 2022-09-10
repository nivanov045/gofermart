package authenticator

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

type Storage interface {
	AddUser(login string, passwordHash string) error
	AddSession(login string, sessionToken string, expiresAt time.Time) error
	GetUserBySessionToken(sessionToken string) (string, error)
	CheckPassword(login string, passwordHash string) (bool, error)
}

type authenticator struct {
	storage Storage
	isDebug bool
}

func New(storage Storage, isDebug bool) *authenticator {
	return &authenticator{storage: storage, isDebug: isDebug}
}

func (a *authenticator) CheckAuthentication(sessionToken string) (string, error) {
	login, err := a.storage.GetUserBySessionToken(sessionToken)
	if err != nil {
		return "", err
	}
	//TODO: Check date of expiration
	return login, nil
}

func (a *authenticator) Register(requestBody []byte) (string, error) {
	type request struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}
	var currentRequest request
	err := json.Unmarshal(requestBody, &currentRequest)
	if err != nil {
		return "", errors.New("wrong query")
	}
	// TODO: Add password hashing
	err = a.storage.AddUser(currentRequest.Login, currentRequest.Password)
	if err != nil {
		return "", err
	}
	var newSessionToken string
	if a.isDebug {
		newSessionToken = currentRequest.Login + "_s"
	} else {
		newSessionToken = uuid.NewString()
	}
	expiresAt := time.Now().Add(120 * time.Hour)
	err = a.storage.AddSession(currentRequest.Login, newSessionToken, expiresAt)
	return newSessionToken, err
}

func (a *authenticator) Login(requestBody []byte) (string, error) {
	//TODO: Move struct from this and Register function
	type request struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}
	var currentRequest request
	err := json.Unmarshal(requestBody, &currentRequest)
	if err != nil {
		return "", errors.New("wrong query")
	}
	// TODO: Add password hashing
	res, err := a.storage.CheckPassword(currentRequest.Login, currentRequest.Password)
	if err != nil {
		return "", err
	}
	if !res {
		return "", nil
	}
	var newSessionToken string
	if a.isDebug {
		newSessionToken = currentRequest.Login + "_s"
	} else {
		newSessionToken = uuid.NewString()
	}
	expiresAt := time.Now().Add(120 * time.Hour)
	err = a.storage.AddSession(currentRequest.Login, newSessionToken, expiresAt)
	return newSessionToken, err
}
