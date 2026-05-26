package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/tanmayd-dev/TaskTide/internal/api/queue"
	"github.com/tanmayd-dev/TaskTide/internal/db"
	"github.com/tanmayd-dev/TaskTide/internal/models"
)

func main() {
	// Load eng
	err := godotenv.Load()
	if err != nil {
		log.Printf("Could not load .env\n")
	}

	// Setup PostgreSQL connection
	ctx := context.Background()
	conn := os.Getenv("POSTGRES_DBSTRING")
	pool, err := pgxpool.New(ctx, conn)
	if err != nil {
		log.Fatalf("Could not create PostgreSQL pool - %v.", err)
	}
	defer pool.Close()
	queries := db.New(pool)

	// In memory storage of tasks for now
	taskQueue := []models.TaskDefinition{}

	// Setup router
	router := chi.NewRouter()

	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	router.Route(
		"/", queue.Router(&taskQueue, queries, ctx),
	)

	server := http.Server{
		Addr:    fmt.Sprintf("%v:%v", os.Getenv("QUEUE_HOST"), os.Getenv("QUEUE_PORT")),
		Handler: router,
	}

	log.Fatal(server.ListenAndServe())
}
