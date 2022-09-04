package api

import (
	"io/ioutil"
	"log"
	"net/http"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
)

type Authenticator interface {
	Register([]byte) error
	Login([]byte) error
	CheckAuthentication([]byte) bool
}

type Service interface {
	AddOrder([]byte) (bool, error)
	GetOrders([]byte) ([]byte, error)
	GetBalance([]byte) ([]byte, error)
	MakeWithdraw([]byte) error
	GetWithdraws([]byte) ([]byte, error)
}

type api struct {
	authenticator Authenticator
	service       Service
}

func New(service Service, authenticator Authenticator) *api {
	return &api{service: service, authenticator: authenticator}
}

func (a *api) Run(address string) error {
	log.Println("api::Run::info: started with addr:", address)
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Post("/api/user/register", a.registerHandler)
	r.Post("/api/user/login", a.loginHandler)
	r.Post("/api/user/orders", a.addOrderHandler)
	r.Get("/api/user/orders", a.getOrdersHandler)
	r.Get("/api/user/balance", a.getBalanceHandler)
	r.Post("/api/user/balance/withdraw", a.makeWithdrawHandler)
	r.Get("/api/user/balance/withdrawals", a.getWithdrawsHandler)
	return http.ListenAndServe(address, r)
}

func (a *api) registerHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("api::registerHandler::info: started")
	w.Header().Set("content-type", "application/json")
	defer r.Body.Close()
	respBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println("api::registerHandler::warning: can't read response body with:", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("{}"))
		return
	}

	err = a.authenticator.Register(respBody)
	if err != nil {
		if err.Error() == "wrong request" {
			w.WriteHeader(http.StatusBadRequest)
		} else if err.Error() == "login is already in use" {
			w.WriteHeader(http.StatusConflict)
		} else {
			log.Println("api::registerHandler::error: unhandled:", err)
			w.WriteHeader(http.StatusInternalServerError)
		}
	} else {
		log.Println("api::registerHandler::info: StatusOK")
		w.WriteHeader(http.StatusOK)
	}
	w.Write([]byte("{}"))
}

func (a *api) loginHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("api::loginHandler::info: started")
	w.Header().Set("content-type", "application/json")
	defer r.Body.Close()
	respBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println("api::loginHandler::warning: can't read response body with:", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("{}"))
		return
	}

	err = a.authenticator.Login(respBody)
	if err != nil {
		if err.Error() == "wrong request" {
			w.WriteHeader(http.StatusBadRequest)
		} else if err.Error() == "wrong login or password" {
			w.WriteHeader(http.StatusUnauthorized)
		} else {
			log.Println("api::loginHandler::error: unhandled:", err)
			w.WriteHeader(http.StatusInternalServerError)
		}
	} else {
		log.Println("api::loginHandler::info: StatusOK")
		w.WriteHeader(http.StatusOK)
	}
	w.Write([]byte("{}"))
}

func (a *api) addOrderHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("api::addOrderHandler::info: started")
	w.Header().Set("content-type", "application/json")
	defer r.Body.Close()
	respBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println("api::addOrderHandler::warning: can't read response body with:", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("{}"))
		return
	}

	isAuthenticated := a.authenticator.CheckAuthentication(respBody)
	if !isAuthenticated {
		log.Println("api::addOrderHandler::warning: not authenticated user")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("{}"))
		return
	}

	isOrderAlreadyUploaded, err := a.service.AddOrder(respBody)
	if err != nil {
		if err.Error() == "wrong request" {
			w.WriteHeader(http.StatusBadRequest)
		} else if err.Error() == "order was uploaded by another user" {
			w.WriteHeader(http.StatusConflict)
		} else if err.Error() == "wrong format of order" {
			w.WriteHeader(http.StatusUnprocessableEntity)
		} else {
			log.Println("api::addOrderHandler::error: unhandled:", err)
			w.WriteHeader(http.StatusInternalServerError)
		}
	} else {
		if isOrderAlreadyUploaded {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusAccepted)
		}
	}
	w.Write([]byte("{}"))
}

func (a *api) getOrdersHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("api::getOrdersHandler::info: started")
	w.Header().Set("content-type", "application/json")
	defer r.Body.Close()
	respBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println("api::getOrdersHandler::warning: can't read response body with:", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("{}"))
		return
	}

	isAuthenticated := a.authenticator.CheckAuthentication(respBody)
	if !isAuthenticated {
		log.Println("api::getOrdersHandler::warning: not authenticated user")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("{}"))
		return
	}

	res, err := a.service.GetOrders(respBody)
	if err != nil {
		if err.Error() == "wrong request" {
			w.WriteHeader(http.StatusBadRequest)
		} else if err.Error() == "no orders" {
			w.WriteHeader(http.StatusNoContent)
		} else {
			log.Println("api::getOrdersHandler::error: unhandled:", err)
			w.WriteHeader(http.StatusInternalServerError)
		}
		w.Write([]byte("{}"))
		return
	}
	w.Write(res)
	w.WriteHeader(http.StatusOK)
}

func (a *api) getBalanceHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("api::getBalanceHandler::info: started")
	w.Header().Set("content-type", "application/json")
	defer r.Body.Close()
	respBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println("api::getBalanceHandler::warning: can't read response body with:", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("{}"))
		return
	}

	isAuthenticated := a.authenticator.CheckAuthentication(respBody)
	if !isAuthenticated {
		log.Println("api::getBalanceHandler::warning: not authenticated user")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("{}"))
		return
	}

	res, err := a.service.GetBalance(respBody)
	if err != nil {
		if err.Error() == "wrong request" {
			w.WriteHeader(http.StatusBadRequest)
		} else {
			log.Println("api::getBalanceHandler::error: unhandled:", err)
			w.WriteHeader(http.StatusInternalServerError)
		}
		w.Write([]byte("{}"))
		return
	}
	w.Write(res)
	w.WriteHeader(http.StatusOK)
}

func (a *api) makeWithdrawHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("api::makeWithdrawHandler::info: started")
	w.Header().Set("content-type", "application/json")
	defer r.Body.Close()
	respBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println("api::makeWithdrawHandler::warning: can't read response body with:", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("{}"))
		return
	}

	isAuthenticated := a.authenticator.CheckAuthentication(respBody)
	if !isAuthenticated {
		log.Println("api::makeWithdrawHandler::warning: not authenticated user")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("{}"))
		return
	}

	err = a.service.MakeWithdraw(respBody)
	if err != nil {
		if err.Error() == "wrong request" {
			w.WriteHeader(http.StatusBadRequest)
		} else if err.Error() == "not enough balance" {
			w.WriteHeader(http.StatusPaymentRequired)
		} else if err.Error() == "no such order" {
			w.WriteHeader(http.StatusUnprocessableEntity)
		} else {
			log.Println("api::makeWithdrawHandler::error: unhandled:", err)
			w.WriteHeader(http.StatusInternalServerError)
		}
	} else {
		w.WriteHeader(http.StatusOK)
	}
	w.Write([]byte("{}"))
}

func (a *api) getWithdrawsHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("api::getWithdrawsHandler::info: started")
	w.Header().Set("content-type", "application/json")
	defer r.Body.Close()
	respBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println("api::getWithdrawsHandler::warning: can't read response body with:", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("{}"))
		return
	}

	isAuthenticated := a.authenticator.CheckAuthentication(respBody)
	if !isAuthenticated {
		log.Println("api::getWithdrawsHandler::warning: not authenticated user")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("{}"))
		return
	}

	res, err := a.service.GetWithdraws(respBody)
	if err != nil {
		if err.Error() == "wrong request" {
			w.WriteHeader(http.StatusBadRequest)
		} else if err.Error() == "no withdraws" {
			w.WriteHeader(http.StatusNoContent)
		} else {
			log.Println("api::getWithdrawsHandler::error: unhandled:", err)
			w.WriteHeader(http.StatusInternalServerError)
		}
		w.Write([]byte("{}"))
		return
	}
	w.Write(res)
	w.WriteHeader(http.StatusOK)
}

type API interface {
	Run(string2 string) error
}

var _ API = &api{}
