package server

import (
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/nivanov045/gofermart/internal/accrual/log"
	"github.com/nivanov045/gofermart/internal/accrual/services"
)

type Server struct {
	service *services.Service
}

func NewServer(service *services.Service) *Server {
	return &Server{service: service}
}

func (a *Server) Run(address string) error {
	r := chi.NewRouter()

	r.Route("/api/", func(r chi.Router) {
		r.Get("/orders/{number}", a.getOrderStatus)
		r.Post("/orders", a.registerOrder)
		r.Post("/goods", a.registerProduct)
	})

	return http.ListenAndServe(address, r)
}

func (a *Server) getOrderStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "number")

	response, err := a.service.GetOrderReward(r.Context(), id)
	if err != nil {
		log.Error(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.Write(response)
}

func (a *Server) registerOrder(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error(err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	err = a.service.RegisterOrder(r.Context(), body)
	if err != nil {
		log.Error(err)
		if errors.Is(err, services.ErrOrderAlreadyRegistered) {
			http.Error(w, http.StatusText(http.StatusConflict), http.StatusConflict)
			return
		}
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (a *Server) registerProduct(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	err = a.service.RegisterProduct(r.Context(), body)
	if err != nil {
		log.Error(err)

		if errors.Is(err, services.ErrProductAlreadyRegistered) {
			http.Error(w, http.StatusText(http.StatusConflict), http.StatusConflict)
		} else if errors.Is(err, services.ErrIncorrectFormat) {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		} else {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}
