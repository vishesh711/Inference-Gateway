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
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/vishesh/inference-gateway/internal/backend"
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
	
	// Initialize backend registry and router
	registry := backend.NewRegistry()
	router := backend.NewRouter(registry)
	
	// Register backends from config
	if len(cfg.Backends) > 0 {
		for _, bcfg := range cfg.Backends {
			log.Printf("Registering backend: %s (%s)", bcfg.ID, bcfg.URL)
			
			// Create backend
			be := &backend.Backend{
				ID:       bcfg.ID,
				URL:      bcfg.URL,
				Model:    bcfg.Model,
				Capacity: bcfg.Capacity,
				Status:   backend.StatusHealthy,
			}
			registry.Register(be)
			
			// Create engine client for this backend
			client := engine.NewClient(bcfg.URL, bcfg.Timeout, bcfg.MaxRetries)
			router.RegisterClient(bcfg.ID, client)
		}
	} else {
		log.Fatal("No backends configured. Please add backends to config.yaml")
	}
	
	log.Printf("Max in-flight: %d, Queue size: %d", cfg.Scheduler.MaxInFlight, cfg.Scheduler.QueueSize)

	// Initialize components
	admission := scheduler.NewAdmissionController(cfg.Scheduler.QueueSize)
	
	// Use HybridScheduler for token-aware admission control
	var hybridScheduler *scheduler.HybridScheduler
	if cfg.Scheduler.UseTokenScheduling {
		log.Printf("Using token-aware scheduling: capacity=%d tokens", cfg.Scheduler.TokenCapacity)
		hybridScheduler = scheduler.NewHybridScheduler(
			cfg.Scheduler.TokenCapacity,
			int64(cfg.Scheduler.MaxInFlight),
			true, // use token scheduling
		)
	} else {
		log.Printf("Using request-based scheduling: max_in_flight=%d", cfg.Scheduler.MaxInFlight)
		hybridScheduler = scheduler.NewHybridScheduler(
			cfg.Scheduler.TokenCapacity,
			int64(cfg.Scheduler.MaxInFlight),
			false, // use request-based
		)
	}
	
	cacheInstance := cache.New(cfg.Cache.Enabled, cfg.Cache.MaxEntries, cfg.Cache.TTL)
	metricsInstance := metrics.New()
	accountant := cost.New(cfg.Cost.GPUHourlyRate)
	
	// Create a default engine client for embeddings batcher (uses first backend)
	var embeddingClient *engine.Client
	if len(cfg.Backends) > 0 {
		firstBackend := cfg.Backends[0]
		embeddingClient = engine.NewClient(firstBackend.URL, firstBackend.Timeout, firstBackend.MaxRetries)
	}
	batcher := scheduler.NewEmbeddingBatcher(cfg.Scheduler.EmbedMaxBatch, cfg.Scheduler.EmbedMaxWaitMs)

	log.Printf("Cost tracking: GPU hourly rate = $%.2f", cfg.Cost.GPUHourlyRate)
	
	// Start health checker
	healthChecker := backend.NewHealthChecker(
		registry,
		cfg.Health.CheckInterval,
		cfg.Health.CheckTimeout,
		func(ctx context.Context, url string) error {
			// Simple health check - try to reach /health endpoint
			req, err := http.NewRequestWithContext(ctx, "GET", url+"/health", nil)
			if err != nil {
				return err
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("unhealthy status: %d", resp.StatusCode)
			}
			return nil
		},
	)
	go healthChecker.Start()
	defer healthChecker.Stop()
	
	// Update backend metrics periodically
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			for _, be := range registry.GetAll() {
				if be.IsHealthy() {
					metricsInstance.BackendHealth.WithLabelValues(be.ID, "healthy").Set(1)
				} else {
					metricsInstance.BackendHealth.WithLabelValues(be.ID, "unhealthy").Set(0)
				}
				metricsInstance.BackendInFlight.WithLabelValues(be.ID).Set(float64(be.GetLoad()))
				// Check circuit state - need to add getter
				if be.IsCircuitOpen() {
					metricsInstance.BackendCircuitOpen.WithLabelValues(be.ID).Set(1)
				} else {
					metricsInstance.BackendCircuitOpen.WithLabelValues(be.ID).Set(0)
				}
			}
		}
	}()
	
	// Update token budget metrics if token scheduling is enabled
	if cfg.Scheduler.UseTokenScheduling {
		go func() {
			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				metricsInstance.TokenBudgetCapacity.Set(float64(hybridScheduler.GetStats()["capacity"].(int64)))
				metricsInstance.TokenBudgetInUse.Set(float64(hybridScheduler.GetStats()["in_use"].(int64)))
				metricsInstance.TokenBudgetUtilization.Set(hybridScheduler.GetStats()["utilization"].(float64))
			}
		}()
	}

	// Create handler with router
	h := handler.New(
		router,
		admission,
		hybridScheduler,
		cacheInstance,
		metricsInstance,
		accountant,
		120*time.Second, // Default timeout
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
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		batchReq := &engine.EmbeddingRequest{
			Model: model,
			Input: allInputs,
		}

		resp, err := embeddingClient.CreateEmbedding(ctx, batchReq)

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
