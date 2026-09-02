package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/vishesh/inference-gateway/internal/cache"
	"github.com/vishesh/inference-gateway/internal/config"
	"github.com/vishesh/inference-gateway/internal/cost"
	"github.com/vishesh/inference-gateway/internal/engine"
	"github.com/vishesh/inference-gateway/internal/handler"
	"github.com/vishesh/inference-gateway/internal/metrics"
	"github.com/vishesh/inference-gateway/internal/scheduler"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Starting Inference Gateway on port %d", cfg.Server.Port)
	log.Printf("Engine URL: %s", cfg.Engine.URL)
	log.Printf("Max in-flight: %d, Queue size: %d", cfg.Scheduler.MaxInFlight, cfg.Scheduler.QueueSize)

	// Initialize components
	engineClient := engine.NewClient(cfg.Engine.URL, cfg.Engine.Timeout, cfg.Engine.MaxRetries)
	admission := scheduler.NewAdmissionController(cfg.Scheduler.QueueSize)
	sched := scheduler.NewScheduler(int64(cfg.Scheduler.MaxInFlight))
	cacheInstance := cache.New(cfg.Cache.Enabled, cfg.Cache.MaxEntries, cfg.Cache.TTL)
	metricsInstance := metrics.New()
	accountant := cost.New(cfg.Cost.GPUHourlyRate)
	batcher := scheduler.NewEmbeddingBatcher(cfg.Scheduler.EmbedMaxBatch, cfg.Scheduler.EmbedMaxWaitMs)

	log.Printf("Cost tracking: GPU hourly rate = $%.2f", cfg.Cost.GPUHourlyRate)

	// Create handler
	h := handler.New(
		engineClient,
		admission,
		sched,
		cacheInstance,
		metricsInstance,
		accountant,
		cfg.Engine.Timeout,
	)
	h.SetBatcher(batcher)

	// Start embeddings batcher with dispatch function
	batcher.Start(func(requests []*scheduler.EmbeddingRequest) {
		// Record batch size
		metricsInstance.BatchSize.Observe(float64(len(requests)))

		// Coalesce all inputs into one engine request
		var allInputs []string
		var model string
		for _, req := range requests {
			allInputs = append(allInputs, req.Inputs...)
			if model == "" {
				model = req.Model
			}
		}

		// Call engine once for the entire batch
		ctx, cancel := context.WithTimeout(context.Background(), cfg.Engine.Timeout)
		defer cancel()

		batchReq := &engine.EmbeddingRequest{
			Model: model,
			Input: allInputs,
		}

		resp, err := engineClient.CreateEmbedding(ctx, batchReq)

		// Distribute results back to individual requests
		if err != nil {
			for _, req := range requests {
				select {
				case req.Response <- scheduler.EmbeddingResponse{Err: err}:
				default:
				}
			}
			return
		}

		// Split embeddings back to original requests
		offset := 0
		for _, req := range requests {
			count := len(req.Inputs)
			if offset+count > len(resp.Data) {
				select {
				case req.Response <- scheduler.EmbeddingResponse{
					Err: fmt.Errorf("insufficient embeddings in batch response"),
				}:
				default:
				}
				continue
			}

			var embeddings [][]float64
			for i := 0; i < count; i++ {
				embeddings = append(embeddings, resp.Data[offset+i].Embedding)
			}
			offset += count

			// Estimate tokens per request proportionally
			tokensPerInput := resp.Usage.PromptTokens / len(allInputs)
			tokens := tokensPerInput * count

			select {
			case req.Response <- scheduler.EmbeddingResponse{
				Embeddings: embeddings,
				Tokens:     tokens,
			}:
			default:
			}
		}
	})

	// Setup HTTP routes
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/completions", h.HandleCompletions)
	mux.HandleFunc("/v1/chat/completions", h.HandleChatCompletions)
	mux.HandleFunc("/v1/embeddings", h.HandleEmbeddings)
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK")
	})

	// Create HTTP server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: mux,
	}

	// Start server in background
	go func() {
		log.Printf("Listening on http://localhost:%d", cfg.Server.Port)
		log.Printf("Metrics available at http://localhost:%d/metrics", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down gracefully...")

	// Stop accepting new requests
	admission.Shutdown(cfg.Server.ShutdownTimeout)
	
	// Stop batcher
	batcher.Shutdown(cfg.Server.ShutdownTimeout)

	// Shutdown HTTP server
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Gateway stopped")
}
