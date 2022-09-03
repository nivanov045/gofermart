package main

import (
	"github.com/nivanov045/gofermart/cmd/gophermart/authenticator"
	"log"

	"github.com/nivanov045/gofermart/cmd/gophermart/api"
	"github.com/nivanov045/gofermart/cmd/gophermart/config"
	"github.com/nivanov045/gofermart/cmd/gophermart/service"
	"github.com/nivanov045/gofermart/cmd/gophermart/storage"
)

func main() {
	cfg, err := config.BuildConfig()
	if err != nil {
		log.Fatalln("service::main::error: in env parsing:", err)
	}
	log.Println("service::main::info: cfg:", cfg)

	myStorage := storage.New()
	serv := service.New(myStorage)
	auth := authenticator.New(myStorage)
	myapi := api.New(serv, auth)
	log.Fatalln(myapi.Run(cfg.ServiceAddress))
}
