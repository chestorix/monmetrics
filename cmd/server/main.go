package main

import (
	"github.com/chestorix/monmetrics/internal/config"
	"github.com/chestorix/monmetrics/internal/server"
	"github.com/chestorix/monmetrics/internal/storage/memory"
)

func main() {
	cfg := config.ServerConfig{
		Address: ":8080",
	}

	storage := memory.NewMemStorage()
	srv := server.New(&cfg, storage)

	if err := srv.Start(); err != nil {
		panic(err)
	}

}
