package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/raphaelreis/go-event-ingestor/internal/config"
	internalHttp "github.com/raphaelreis/go-event-ingestor/internal/http"
	"github.com/raphaelreis/go-event-ingestor/internal/ingest"
	"github.com/raphaelreis/go-event-ingestor/internal/kafka"
	"github.com/raphaelreis/go-event-ingestor/internal/metrics"
	"github.com/raphaelreis/go-event-ingestor/internal/rate"
	"github.com/raphaelreis/go-event-ingestor/pkg/logger"
)

func main() {
	cfg := config.LoadFromEnv()
	log := logger.New(cfg.LogLevel)

	log.Info("Starting Event Ingestor", "config", cfg)

	mets := metrics.New()

	producer := kafka.NewProducer(
		cfg.KafkaBrokers,
		cfg.KafkaTopic,
		cfg.KafkaDLQTopic,
		cfg.KafkaWriteTimeout,
	)
	defer producer.Close()

	svc := ingest.NewService(
		cfg.QueueSize,
		cfg.WorkerPoolSize,
		producer,
		log,
		mets,
	)
	defer svc.Shutdown()

	limiter := rate.NewTokenLimiter(cfg.RateLimitRPS, cfg.RateLimitBurst)
	handler := internalHttp.NewHandler(svc, limiter, log, mets)

	mux := http.NewServeMux()
	mux.HandleFunc("/events", handler.Ingest)
	mux.Handle("/metrics", promhttp.Handler())

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	srv := &http.Server{
		Addr:         ":" + cfg.HTTPPort,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Info("Server listening", "port", cfg.HTTPPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("Server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("Server forced to shutdown", "error", err)
	}

	log.Info("Server exited properly")
}