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

	// Setup PostgreSQL connection
	ctx := context.Background()
	conn := os.Getenv("POSTGRES_DBSTRING")
	pool, err := pgxpool.New(ctx, conn)
	if err != nil {
		log.Fatalf("Could not create PostgreSQL pool - %v.", err)
	}
	defer pool.Close()
	queries := db.New(pool)

	// Start background recovery loop
	queueService.StartRecoveryLoop(ctx, queries)

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
	pb.RegisterTaskTideServiceServer(grpcServer, &queueService.GrpcHandler{Queries: queries, MinioClient: minioClient})
	log.Printf("Starting gRPC server on port %s...\n", grpcPort)
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve gRPC: %v", err)
		}
	}()

	// Initialize OpenTelemetry tracer
	tp, err := observability.InitTracer("queue-service")
	if err == nil {
		defer tp.Shutdown(ctx)
	}

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

	log.Fatal(server.ListenAndServe())
}

