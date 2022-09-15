package authenticator

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
)

type Storage interface {
	AddUser(login string, passwordHash string) error
	AddSession(login string, sessionToken string, expiresAt time.Time) error
	GetSessionInfo(sessionToken string) (string, time.Time, error)
	CheckPassword(login string, passwordHash string) (bool, error)
	RemoveSession(sessionToken string) error
}

type Crypto interface {
	CreateHash(s string) string
}

type authenticator struct {
	storage Storage
	isDebug bool
	crypto  Crypto
}

func New(storage Storage, isDebug bool, crypto Crypto) *authenticator {
	return &authenticator{storage: storage, isDebug: isDebug, crypto: crypto}
}

func (a *authenticator) CheckAuthentication(sessionToken string) (string, error) {
	login, expiredAt, err := a.storage.GetSessionInfo(sessionToken)
	if err != nil {
		return "", err
	}
	if expiredAt.Before(time.Now()) {
		return "", errors.New("session token expired")
	}
	return login, nil
}

type userAuthData struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (a *authenticator) Register(requestBody []byte) (string, error) {
	var authData userAuthData
	err := json.Unmarshal(requestBody, &authData)
	if err != nil {
		return "", errors.New("wrong query")
	}
	hash := a.crypto.CreateHash(authData.Password)
	log.Println(hash)
	err = a.storage.AddUser(authData.Login, hash)
	if err != nil {
		if err.Error() == "login is already in use" {
			return "", err
		}
		return "", fmt.Errorf("authenticator::regitster: at storage.AddUser: [%w]", err)
	}
	var newSessionToken string
	if a.isDebug {
		newSessionToken = authData.Login + "_s"
	} else {
		newSessionToken = uuid.NewString()
	}
	expiresAt := time.Now().Add(120 * time.Hour)
	err = a.storage.AddSession(authData.Login, newSessionToken, expiresAt)
	if err != nil {
		return "", fmt.Errorf("authenticator::regitster: at storage.AddSession: [%w]", err)
	}
	return newSessionToken, nil
}

func (a *authenticator) Login(requestBody []byte) (string, error) {
	var userAuthData userAuthData
	err := json.Unmarshal(requestBody, &userAuthData)
	if err != nil {
		return "", errors.New("wrong query")
	}
	res, err := a.storage.CheckPassword(userAuthData.Login, a.crypto.CreateHash(userAuthData.Password))
	if err != nil {
		return "", err
	}
	if !res {
		return "", nil
	}
	var newSessionToken string
	if a.isDebug {
		newSessionToken = userAuthData.Login + "_s"
	} else {
		newSessionToken = uuid.NewString()
	}
	expiresAt := time.Now().Add(120 * time.Hour)
	err = a.storage.AddSession(userAuthData.Login, newSessionToken, expiresAt)
	return newSessionToken, err
}

func (a *authenticator) Logout(sessionToken string) error {
	_, _, err := a.storage.GetSessionInfo(sessionToken)
	if err != nil {
		return err
	}
	err = a.storage.RemoveSession(sessionToken)
	return err
}
