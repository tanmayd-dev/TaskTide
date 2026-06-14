package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/tanmayd-dev/TaskTide/internal/api/worker"
	"github.com/tanmayd-dev/TaskTide/internal/db"
	"github.com/tanmayd-dev/TaskTide/internal/service/workerService"
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

	// Listen
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Printf("Could not create listener: %v\n", err.Error())
		return
	}
	address := listener.Addr().(*net.TCPAddr)

	// Register self
	workerID, err := workerService.RegisterSelf(address.Port)
	if err != nil {
		log.Fatalf("%v\n", err.Error())
	}

	// Create Router
	router := chi.NewRouter()

	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	router.Route("/", worker.Router(queries, ctx, workerID))

	// Server
	server := http.Server{
		Addr:    fmt.Sprintf("%v:%v", os.Getenv("WORKER_HOST"), os.Getenv("WORKER_PORT")),
		Handler: router,
	}

	// Serve
	log.Fatal(server.Serve(listener))
}
