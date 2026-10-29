package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/tanmayd-dev/TaskTide/internal/api/queue"
	"github.com/tanmayd-dev/TaskTide/internal/db"
	"github.com/tanmayd-dev/TaskTide/internal/observability"
	"github.com/tanmayd-dev/TaskTide/internal/pb"
	"github.com/tanmayd-dev/TaskTide/internal/service/queueService"
	"google.golang.org/grpc"
)

func main() {
	// Load env
	err := godotenv.Load()
	if err != nil {
		log.Printf("Could not load .env\n")
	}

	// Initialize OpenTelemetry tracer before any Serve call so all servers
	// emit with tracing already set up
	tp, err := observability.InitTracer("queue-service")
	if err == nil {
		defer tp.Shutdown(context.Background())
	}

	// Setup PostgreSQL connection
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	conn := os.Getenv("POSTGRES_DBSTRING")
	pool, err := pgxpool.New(ctx, conn)
	if err != nil {
		log.Fatalf("Could not create PostgreSQL pool - %v.", err)
	}
	defer pool.Close()
	queries := db.New(pool)

	// Ensure default queue exists on startup
	_ = queries.EnsureQueueExists(ctx, "default")

	// Start background recovery loop
	queueService.StartRecoveryLoop(ctx, pool, queries)

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

	// Setup gRPC Server
	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "50051"
	}
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", grpcPort))
	if err != nil {
		log.Fatalf("failed to listen for gRPC: %v", err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterTaskTideServiceServer(grpcServer, &queueService.GrpcHandler{
		Pool:        pool,
		Queries:     queries,
		MinioClient: minioClient,
	})
	log.Printf("Starting gRPC server on port %s...\n", grpcPort)
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve gRPC: %v", err)
		}
	}()

	// Start background metrics collector
	observability.StartMetricsCollector(ctx, queries)

	// Setup router
	router := chi.NewRouter()

	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	// Register Prometheus metrics endpoint
	router.Handle("/metrics", promhttp.Handler())

	router.Route(
		"/", queue.Router(queries, ctx, minioClient),
	)

	server := http.Server{
		Addr:    fmt.Sprintf("%v:%v", os.Getenv("QUEUE_HOST"), os.Getenv("QUEUE_PORT")),
		Handler: router,
	}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("failed to serve HTTP: %v", err)
		}
	}()

	// Graceful shutdown: on SIGINT/SIGTERM stop the background loops, drain
	// both servers with a bound, then close the pool and exit normally.
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan
	log.Println("Received shutdown signal. Draining...")

	cancel() // Stop the recovery loop and metrics collector.

	grpcStopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(grpcStopped)
	}()
	select {
	case <-grpcStopped:
	case <-time.After(10 * time.Second):
		grpcServer.Stop()
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	pool.Close()
	log.Println("queue-service shut down cleanly.")
}

