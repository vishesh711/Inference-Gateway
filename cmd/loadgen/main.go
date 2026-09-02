package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// CompletionRequest matches the gateway's expected format
type CompletionRequest struct {
	Model       string  `json:"model"`
	Prompt      string  `json:"prompt"`
	MaxTokens   int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
}

// CompletionResponse matches the gateway's response
type CompletionResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Text string `json:"text"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// Result captures metrics for a single request
type Result struct {
	Latency time.Duration
	Success bool
	Tokens  int
}

func main() {
	url := flag.String("url", "http://localhost:8000/v1/completions", "Gateway URL")
	workers := flag.Int("workers", 4, "Number of concurrent workers")
	duration := flag.Duration("duration", 60*time.Second, "Test duration")
	warmup := flag.Duration("warmup", 10*time.Second, "Warmup period to discard")
	model := flag.String("model", "default", "Model name")
	prompt := flag.String("prompt", "Write a short story about a robot.", "Prompt text")
	maxTokens := flag.Int("max-tokens", 100, "Max tokens to generate")
	flag.Parse()

	log.Printf("Load Generator Configuration:")
	log.Printf("  URL: %s", *url)
	log.Printf("  Workers: %d", *workers)
	log.Printf("  Duration: %s (warmup: %s)", *duration, *warmup)
	log.Printf("  Model: %s", *model)
	log.Printf("  Prompt length: %d chars", len(*prompt))
	log.Printf("")

	// Prepare request body
	reqBody := CompletionRequest{
		Model:       *model,
		Prompt:      *prompt,
		MaxTokens:   *maxTokens,
		Temperature: 0.7,
	}

	// Shared state
	var (
		results     []Result
		resultsMu   sync.Mutex
		totalReqs   int64
		successReqs int64
		errorReqs   int64
	)

	ctx, cancel := context.WithTimeout(context.Background(), *duration+*warmup)
	defer cancel()

	startTime := time.Now()
	warmupEnd := startTime.Add(*warmup)

	log.Println("Starting load test...")

	// Launch workers
	var wg sync.WaitGroup
	for i := 0; i < *workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			client := &http.Client{
				Timeout: 180 * time.Second,
			}

			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				// Send request
				result := sendRequest(client, *url, reqBody)
				atomic.AddInt64(&totalReqs, 1)

				if result.Success {
					atomic.AddInt64(&successReqs, 1)
				} else {
					atomic.AddInt64(&errorReqs, 1)
				}

				// Only record results after warmup
				if time.Now().After(warmupEnd) {
					resultsMu.Lock()
					results = append(results, result)
					resultsMu.Unlock()
				}
			}
		}(i)
	}

	// Progress reporter
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				elapsed := time.Since(startTime).Seconds()
				total := atomic.LoadInt64(&totalReqs)
				success := atomic.LoadInt64(&successReqs)
				errors := atomic.LoadInt64(&errorReqs)
				rps := float64(total) / elapsed
				log.Printf("[%.0fs] Total: %d, Success: %d, Errors: %d, RPS: %.1f",
					elapsed, total, success, errors, rps)
			}
		}
	}()

	// Wait for completion
	wg.Wait()

	// Calculate and display results
	displayResults(results, time.Since(warmupEnd))
}

func sendRequest(client *http.Client, url string, reqBody CompletionRequest) Result {
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return Result{Success: false}
	}

	start := time.Now()
	resp, err := client.Post(url, "application/json", bytes.NewReader(jsonData))
	latency := time.Since(start)

	if err != nil {
		return Result{
			Latency: latency,
			Success: false,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Result{
			Latency: latency,
			Success: false,
		}
	}

	var completionResp CompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&completionResp); err != nil {
		return Result{
			Latency: latency,
			Success: false,
		}
	}

	return Result{
		Latency: latency,
		Success: true,
		Tokens:  completionResp.Usage.TotalTokens,
	}
}

func displayResults(results []Result, actualDuration time.Duration) {
	if len(results) == 0 {
		log.Println("No results collected")
		return
	}

	// Filter successful requests
	var latencies []time.Duration
	var totalTokens int64
	successCount := 0

	for _, r := range results {
		if r.Success {
			latencies = append(latencies, r.Latency)
			totalTokens += int64(r.Tokens)
			successCount++
		}
	}

	if len(latencies) == 0 {
		log.Println("No successful requests")
		return
	}

	// Sort for percentiles
	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})

	// Calculate percentiles
	p50 := percentile(latencies, 0.50)
	p90 := percentile(latencies, 0.90)
	p95 := percentile(latencies, 0.95)
	p99 := percentile(latencies, 0.99)

	// Calculate mean
	var sum time.Duration
	for _, lat := range latencies {
		sum += lat
	}
	mean := sum / time.Duration(len(latencies))

	// Calculate throughput
	totalReqs := len(results)
	successRate := float64(successCount) / float64(totalReqs) * 100
	throughput := float64(successCount) / actualDuration.Seconds()
	tokensPerSec := float64(totalTokens) / actualDuration.Seconds()

	// Display results
	fmt.Println("\n" + strings("=", 70))
	fmt.Println("LOAD TEST RESULTS")
	fmt.Println(strings("=", 70))
	fmt.Printf("Duration (post-warmup): %s\n", actualDuration.Round(time.Second))
	fmt.Printf("Total Requests:         %d\n", totalReqs)
	fmt.Printf("Successful:             %d (%.1f%%)\n", successCount, successRate)
	fmt.Printf("Failed:                 %d\n", totalReqs-successCount)
	fmt.Println(strings("-", 70))
	fmt.Printf("Throughput:             %.2f req/s\n", throughput)
	fmt.Printf("Tokens/sec:             %.2f tok/s\n", tokensPerSec)
	fmt.Println(strings("-", 70))
	fmt.Printf("Latency Mean:           %s\n", mean.Round(time.Millisecond))
	fmt.Printf("Latency p50:            %s\n", p50.Round(time.Millisecond))
	fmt.Printf("Latency p90:            %s\n", p90.Round(time.Millisecond))
	fmt.Printf("Latency p95:            %s\n", p95.Round(time.Millisecond))
	fmt.Printf("Latency p99:            %s\n", p99.Round(time.Millisecond))
	fmt.Printf("Latency Min:            %s\n", latencies[0].Round(time.Millisecond))
	fmt.Printf("Latency Max:            %s\n", latencies[len(latencies)-1].Round(time.Millisecond))
	fmt.Println(strings("=", 70))
}

func percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	index := int(math.Ceil(float64(len(sorted))*p)) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}

func strings(s string, n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}
