package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/kerucko/diploma/gateway/internal/config"
	"github.com/kerucko/diploma/gateway/internal/handlers"
	"github.com/kerucko/diploma/gateway/internal/repository"
	"github.com/kerucko/diploma/gateway/internal/service/tasks"
	"github.com/kerucko/diploma/gateway/internal/service/users"
	"github.com/kerucko/diploma/gateway/internal/utils"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.MustLoad()
	log.Printf("config: %v", cfg)

	conn, err := repository.NewConnection(ctx, cfg.PostgresConfig)
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}

	userRepo := repository.NewUserRepository(conn)
	taskRepo := repository.NewTaskRepository(conn)
	authManager := utils.NewAuthManager(cfg.JWTSecret, time.Hour*24)

	userService := users.NewService(userRepo, authManager)
	taskService := tasks.NewService(taskRepo)

	h := handlers.NewHandler(userService, taskService, authManager)

	router := chi.NewRouter()
	router.Use(middleware.Logger)

	router.Post("/register", h.Register)
	router.Post("/login", h.Login)

	router.Route("/tasks", func(r chi.Router) {
		r.Use(h.AuthMiddleware)
		r.Post("/", h.CreateTask)
		r.Get("/{id}", h.GetTask)
		r.Get("/", h.ListMyTasks)
	})

	router.Route("/admin", func(r chi.Router) {
		r.Use(h.AuthMiddleware)
		r.Use(h.AdminOnly)
		r.Get("/tasks", h.ListAllTasks)
		r.Get("/users", h.ListUsers)
		r.Get("/users/{userID}", h.GetUserProfile)
		r.Get("/users/{userID}/tasks", h.ListUserTasks)
	})

	log.Print("start listening")
	log.Fatal(http.ListenAndServe("[::]:"+cfg.ServerConfig.Port, router))
}
