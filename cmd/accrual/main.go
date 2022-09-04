package main

import (
	"context"
	"database/sql"

	_ "github.com/jackc/pgx/v4/stdlib"

	"gofermart/internal/accrual/log"
	"gofermart/internal/accrual/server"
	"gofermart/internal/accrual/services"
	"gofermart/internal/accrual/storages"
)

func main() {
	log.Init()

	cfg, err := server.NewConfig()
	if err != nil {
		log.Panic(err)
	}

	db, err := sql.Open("pgx", cfg.DatabaseURI)
	if err != nil {
		log.Panic(err)
	}
	defer db.Close()

	ctx := context.Background()
	storage, err := storages.NewDBStorage(ctx, db)
	if err != nil {
		log.Panic(err)
	}

	service := services.NewService(storage)
	accrualServer := server.NewServer(service)

	err = accrualServer.Run(cfg.Address)
	if err != nil {
		log.Panic(err)
	}
}
