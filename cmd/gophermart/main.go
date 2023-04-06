package main

import (
	"log"

	"github.com/nivanov045/gofermart/cmd/gophermart/accrualsystem"
	"github.com/nivanov045/gofermart/cmd/gophermart/api"
	"github.com/nivanov045/gofermart/cmd/gophermart/authenticator"
	"github.com/nivanov045/gofermart/cmd/gophermart/config"
	"github.com/nivanov045/gofermart/cmd/gophermart/crypto"
	"github.com/nivanov045/gofermart/cmd/gophermart/service"
	"github.com/nivanov045/gofermart/cmd/gophermart/storage"
)

func main() {
	cfg, err := config.BuildConfig()
	if err != nil {
		log.Fatalln("service::main::error: in env parsing:", err)
	}
	log.Println("service::main::info: cfg:", cfg)

	myStorage, err := storage.New(cfg.DatabaseURI)
	if err != nil {
		log.Fatalln("service::main::error: in storage creation:", err)
	}
	accrualSystem, err := accrualsystem.New(cfg.AccrualAddress, cfg.DebugMode)
	if err != nil {
		log.Fatalln("service::main::error: in accrual system creation:", err)
	}
	serv := service.New(myStorage, accrualSystem, cfg.DebugMode)
	myCrypto := crypto.New(cfg.Key)
	auth := authenticator.New(myStorage, cfg.DebugMode, myCrypto)
	myAPI := api.New(serv, auth)
	log.Fatalln(myAPI.Run(cfg.ServiceAddress))
}
