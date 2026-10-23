package api

import (
	"io"
	"log"
	"net/http"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
)

type Authenticator interface {
	Register([]byte) (string, error)
	Login([]byte) (string, error)
	CheckAuthentication(string) (string, error)
	Logout(string) error
}

type Service interface {
	AddOrder(string, []byte) (bool, error)
	GetOrders(string) ([]byte, error)
	GetBalance(string) ([]byte, error)
	MakeWithdraw(string, []byte) error
	GetWithdraws(string) ([]byte, error)
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

	// Specificated
	r.Route("/api/user/", func(r chi.Router) {
		r.Post("/register", a.registerHandler)
		r.Post("/login", a.loginHandler)
		r.Post("/orders", a.addOrderHandler)
		r.Get("/orders", a.getOrdersHandler)
		r.Get("/balance", a.getBalanceHandler)
		r.Post("/balance/withdraw", a.makeWithdrawHandler)
		r.Get("/withdrawals", a.getWithdrawsHandler)
	})

	// Not specificated
	r.Post("/api/user/logout", a.logoutHandler)

	return http.ListenAndServe(address, r)
}

func (a *api) registerHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	defer r.Body.Close()
	respBody, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println("api::registerHandler::warning: can't read response body with:", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("{}"))
		return
	}

	token, err := a.authenticator.Register(respBody)
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
		http.SetCookie(w, &http.Cookie{
			Name:  "session_token",
			Value: token,
		})
		w.WriteHeader(http.StatusOK)
	}
	w.Write([]byte("{}"))
}

func (a *api) loginHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	defer r.Body.Close()
	respBody, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println("api::loginHandler::warning: can't read response body with:", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("{}"))
		return
	}

	token, err := a.authenticator.Login(respBody)
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
		http.SetCookie(w, &http.Cookie{
			Name:  "session_token",
			Value: token,
		})
		w.WriteHeader(http.StatusOK)
	}
	w.Write([]byte("{}"))
}

func (a *api) logoutHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	c, err := r.Cookie("session_token")
	if err != nil {
		if err == http.ErrNoCookie {
			log.Println("api::logoutHandler::warning:", err)
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("{}"))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	sessionToken := c.Value
	err = a.authenticator.Logout(sessionToken)
	if err != nil {
		if err.Error() == "no such token" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("{}"))
			return
		}
		log.Println("api::logoutHandler::error: unhandled in auth check:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{}"))
}

func (a *api) addOrderHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	c, err := r.Cookie("session_token")
	if err != nil {
		if err == http.ErrNoCookie {
			log.Println("api::addOrderHandler::warning:", err)
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("{}"))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	sessionToken := c.Value
	login, err := a.authenticator.CheckAuthentication(sessionToken)
	if err != nil {
		if err.Error() == "no such token" || err.Error() == "session token expired" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("{}"))
			return
		}
		log.Println("api::addOrderHandler::error: unhandled in auth check:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	defer r.Body.Close()
	respBody, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println("api::addOrderHandler::warning: can't read response body with:", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("{}"))
		return
	}

	isOrderNotExisted, err := a.service.AddOrder(login, respBody)
	if err != nil {
		log.Println("api::addOrderHandler::warning in order adding:", err)
		if err.Error() == "wrong request" {
			w.WriteHeader(http.StatusBadRequest)
		} else if err.Error() == "order was uploaded by another user" {
			w.WriteHeader(http.StatusConflict)
		} else if err.Error() == "wrong format of order" {
			w.WriteHeader(http.StatusUnprocessableEntity)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	} else {
		if isOrderNotExisted {
			w.WriteHeader(http.StatusAccepted)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}
	w.Write([]byte("{}"))
}

func (a *api) getOrdersHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("api::getOrdersHandler::info: started")
	w.Header().Set("content-type", "application/json")

	c, err := r.Cookie("session_token")
	if err != nil {
		if err == http.ErrNoCookie {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("{}"))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	sessionToken := c.Value
	login, err := a.authenticator.CheckAuthentication(sessionToken)
	if err != nil {
		if err.Error() == "no such token" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("{}"))
			return
		}
		log.Println("api::getOrdersHandler::error: unhandled in auth check:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	res, err := a.service.GetOrders(login)
	if err != nil {
		if err.Error() == "no orders" {
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

	c, err := r.Cookie("session_token")
	if err != nil {
		if err == http.ErrNoCookie {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("{}"))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	sessionToken := c.Value
	login, err := a.authenticator.CheckAuthentication(sessionToken)
	if err != nil {
		if err.Error() == "no such token" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("{}"))
			return
		}
		log.Println("api::getOrdersHandler::error: unhandled in auth check:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	res, err := a.service.GetBalance(login)
	if err != nil {
		log.Println("api::getBalanceHandler::error: unhandled:", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("{}"))
		return
	}
	w.Write(res)
	w.WriteHeader(http.StatusOK)
}

func (a *api) makeWithdrawHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("api::makeWithdrawHandler::info: started")
	w.Header().Set("content-type", "application/json")

	c, err := r.Cookie("session_token")
	if err != nil {
		if err == http.ErrNoCookie {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("{}"))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	sessionToken := c.Value
	login, err := a.authenticator.CheckAuthentication(sessionToken)
	if err != nil {
		if err.Error() == "no such token" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("{}"))
			return
		}
		log.Println("api::getOrdersHandler::error: unhandled in auth check:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	defer r.Body.Close()
	respBody, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println("api::makeWithdrawHandler::warning: can't read response body with:", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("{}"))
		return
	}

	err = a.service.MakeWithdraw(login, respBody)
	if err != nil {
		if err.Error() == "wrong request" {
			w.WriteHeader(http.StatusBadRequest)
		} else if err.Error() == "not enough balance" {
			w.WriteHeader(http.StatusPaymentRequired)
		} else if err.Error() == "wrong format of order" {
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

	c, err := r.Cookie("session_token")
	if err != nil {
		if err == http.ErrNoCookie {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("{}"))
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	sessionToken := c.Value
	login, err := a.authenticator.CheckAuthentication(sessionToken)
	if err != nil {
		if err.Error() == "no such token" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("{}"))
			return
		}
		log.Println("api::getOrdersHandler::error: unhandled in auth check:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	res, err := a.service.GetWithdraws(login)
	if err != nil {
		if err.Error() == "no withdraws" {
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
	Run(serviceAddress string) error
}

var _ API = &api{}
