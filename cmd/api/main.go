package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	v1 "github.com/tanmayd-dev/TaskTide/internal/api/v1"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Default().Fatalf("Could not load .env\n")
	}

	router := chi.NewRouter()

	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	router.Route("/", v1.Router())

	server := http.Server{
		Addr:    fmt.Sprintf("%v:%v", os.Getenv("API_HOST"), os.Getenv("API_PORT")),
		Handler: router,
	}

	log.Fatal(server.ListenAndServe())
}
