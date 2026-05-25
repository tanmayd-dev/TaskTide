package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	"github.com/tanmayd-dev/TaskTide/internal/api/queue"
	"github.com/tanmayd-dev/TaskTide/internal/models"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Panicf("Could not load .env\n")
	}

	taskQueue := []models.TaskDefinition{}

	router := chi.NewRouter()

	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	router.Route(
		"/", queue.Router(&taskQueue),
	)

	server := http.Server{
		Addr:    fmt.Sprintf("%v:%v", os.Getenv("QUEUE_HOST"), os.Getenv("QUEUE_PORT")),
		Handler: router,
	}

	log.Fatal(server.ListenAndServe())
}
