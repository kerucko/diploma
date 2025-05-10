package main

import (
	"context"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/kerucko/diploma/internal/config"

	"github.com/kerucko/diploma/gateway/internal/repository"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.MustLoad()
	log.Printf("config: %v", cfg)

	conn, err := repository.NewConnection(ctx, cfg.PostgresConfig)
	if err != nil {
		log.Printf("failed to connect to postgres: %v", err)
	}

	router := chi.NewRouter()
	router.Use(middleware.Logger)

	// fs := http.FileServer(http.Dir("public"))
	// router.Handle("/*", fs)

	log.Print("start listening")
	log.Fatal(http.ListenAndServe("[::]:"+cfg.ServerConfig.Port, router))
}
