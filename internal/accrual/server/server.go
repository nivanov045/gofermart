package server

import (
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	"gofermart/internal/accrual/log"
	"gofermart/internal/accrual/services"
)

type Server struct {
	service *services.Service
}

func NewServer(service *services.Service) *Server {
	a := Server{service: service}

	return &a
}

func (a *Server) Run(address string) error {
	r := chi.NewRouter()

	r.Route("/api/", func(r chi.Router) {
		r.Get("/orders/{number}", a.getInfo)
		r.Post("/orders", a.registerOrder)
		r.Post("/goods", a.registerProduct)
	})

	return http.ListenAndServe(address, r)
}

func (a *Server) getInfo(w http.ResponseWriter, r *http.Request) {
	number := chi.URLParam(r, "number")
	fmt.Println(number)
}

func (a *Server) registerOrder(w http.ResponseWriter, r *http.Request) {

}

func (a *Server) registerProduct(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error(err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	err = a.service.RegisterProduct(r.Context(), body)
	// TODO: Parse errors
	if err != nil {
		log.Error(err)
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	w.Header().Add("Content-Type", "application/json")
}
