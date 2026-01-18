package http

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/raphaelreis/go-event-ingestor/internal/ingest"
	"github.com/raphaelreis/go-event-ingestor/internal/metrics"
	"github.com/raphaelreis/go-event-ingestor/internal/model"
	"github.com/raphaelreis/go-event-ingestor/internal/rate"
)

type Handler struct {
	service *ingest.Service
	limiter rate.Limiter
	logger  *slog.Logger
	metrics *metrics.Metrics
}

func NewHandler(service *ingest.Service, limiter rate.Limiter, logger *slog.Logger, m *metrics.Metrics) *Handler {
	return &Handler{
		service: service,
		limiter: limiter,
		logger:  logger,
		metrics: m,
	}
}

func (h *Handler) Ingest(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	
	// 1. Rate Limit
	if !h.limiter.Allow() {
		h.metrics.HTTPRequests.WithLabelValues("429").Inc()
		http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
		return
	}

	// 2. Parse & Validate
	var event model.Event
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		h.metrics.HTTPRequests.WithLabelValues("400").Inc()
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Basic validation / augmentation
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// 3. Ingest (Backpressure handled here)
	err := h.service.Ingest(r.Context(), event)
	if err != nil {
		if err == ingest.ErrQueueFull {
			h.metrics.HTTPRequests.WithLabelValues("503").Inc()
			// Return 503 Service Unavailable when queue is full
			w.Header().Set("Retry-After", "5")
			http.Error(w, "Service Unavailable: Backpressure", http.StatusServiceUnavailable)
			return
		}
		// Unexpected error
		h.metrics.HTTPRequests.WithLabelValues("500").Inc()
		h.logger.Error("Internal server error during ingest", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// 4. Success
	h.metrics.HTTPRequests.WithLabelValues("202").Inc()
	w.Header().Set("X-Request-ID", event.ID)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "accepted", "id": event.ID})

	h.logger.Debug("Request processed", 
		"duration_ms", time.Since(start).Milliseconds(),
		"event_id", event.ID,
	)
}
