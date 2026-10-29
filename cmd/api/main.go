package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/tanmayd-dev/TaskTide/internal/api/api"
	"github.com/tanmayd-dev/TaskTide/internal/db"
)

func main() {
	// Load env
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

	// Setup MinIO connection
	minioClient, err := minio.New(
		fmt.Sprintf("%v:%v", os.Getenv("MINIO_HOST"), os.Getenv("MINIO_PORT")),
		&minio.Options{
			Creds: credentials.NewStaticV4(
				os.Getenv("MINIO_USER"),
				os.Getenv("MINIO_PASSWORD"),
				"",
			),
		},
	)
	if err != nil {
		log.Fatalf("Could not create minioClient: %#v", err)
	}

	// Create server
	router := chi.NewRouter()

	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(corsMiddleware)

	router.Route("/", api.Router(queries, ctx, minioClient))

	server := http.Server{
		Addr:    fmt.Sprintf("%v:%v", os.Getenv("API_HOST"), os.Getenv("API_PORT")),
		Handler: router,
	}

	log.Fatal(server.ListenAndServe())
}

func corsMiddleware(next http.Handler) http.Handler {
	allowedOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	if allowedOrigins == "" {
		allowedOrigins = "http://localhost:5173,http://localhost:3001,http://localhost:3000"
	}
	allowed := make(map[string]bool)
	for _, origin := range strings.Split(allowedOrigins, ",") {
		if origin = strings.TrimSpace(origin); origin != "" {
			allowed[origin] = true
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only echo the Origin header when it is explicitly allowed.
		if origin := r.Header.Get("Origin"); allowed[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
