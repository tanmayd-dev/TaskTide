package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	"github.com/tanmayd-dev/TaskTide/internal/api/worker"
	"github.com/tanmayd-dev/TaskTide/internal/service/workerService"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("Could not load .env\n")
	}

	go workerService.PollForTask()

	router := chi.NewRouter()

	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	router.Route("/", worker.Router())

	server := http.Server{
		Addr:    fmt.Sprintf("%v:%v", os.Getenv("WORKER_HOST"), os.Getenv("WORKER_PORT")),
		Handler: router,
	}

	log.Fatal(server.ListenAndServe())
}
