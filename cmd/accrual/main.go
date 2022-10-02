package main

import (
	"context"
	"database/sql"
	"runtime"
	"sync"

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
	storage, queue, err := storages.NewDBStorage(ctx, db)
	if err != nil {
		log.Panic(err)
	}

	wg := sync.WaitGroup{}
	service := services.NewService(storage, queue, runtime.NumCPU())
	wg.Add(1)
	go func() {
		service.Process(ctx)
		wg.Done()
	}()

	accrualServer := server.NewServer(service)
	wg.Add(1)
	go func() {
		err = accrualServer.Run(cfg.Address)
		if err != nil {
			log.Panic(err)
		}
		wg.Done()
	}()

	wg.Wait()
}
