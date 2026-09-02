package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client manages communication with the LLM engine
type Client struct {
	baseURL    string
	httpClient *http.Client
	maxRetries int
}

// NewClient creates an engine client with timeout and retry config
func NewClient(baseURL string, timeout time.Duration, maxRetries int) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		maxRetries: maxRetries,
	}
}

// CompletionRequest matches OpenAI's completion request format
type CompletionRequest struct {
	Model       string   `json:"model"`
	Prompt      string   `json:"prompt"`
	MaxTokens   int      `json:"max_tokens,omitempty"`
	Temperature float64  `json:"temperature,omitempty"`
	Stream      bool     `json:"stream,omitempty"`
	Stop        []string `json:"stop,omitempty"`
}

// CompletionResponse matches OpenAI's completion response format
type CompletionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Text         string `json:"text"`
		Index        int    `json:"index"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// ChatCompletionRequest matches OpenAI's chat completion format
type ChatCompletionRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
	Stop        []string  `json:"stop,omitempty"`
}

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionResponse matches OpenAI's chat completion response
type ChatCompletionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Message      Message `json:"message"`
		Index        int     `json:"index"`
		FinishReason string  `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// EmbeddingRequest matches OpenAI's embedding request format
type EmbeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// EmbeddingResponse matches OpenAI's embedding response
type EmbeddingResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Embedding []float64 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

// CreateCompletion sends a completion request to the engine
func (c *Client) CreateCompletion(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	var resp CompletionResponse
	err := c.doRequest(ctx, "/v1/completions", req, &resp)
	return &resp, err
}

// CreateChatCompletion sends a chat completion request to the engine
func (c *Client) CreateChatCompletion(ctx context.Context, req *ChatCompletionRequest) (*ChatCompletionResponse, error) {
	var resp ChatCompletionResponse
	err := c.doRequest(ctx, "/v1/chat/completions", req, &resp)
	return &resp, err
}

// CreateEmbedding sends an embedding request to the engine
func (c *Client) CreateEmbedding(ctx context.Context, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	var resp EmbeddingResponse
	err := c.doRequest(ctx, "/v1/embeddings", req, &resp)
	return &resp, err
}

// doRequest performs the HTTP request with retry logic
func (c *Client) doRequest(ctx context.Context, path string, reqBody, respBody interface{}) error {
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshaling request: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(attempt) * 100 * time.Millisecond):
			}
		}

		httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+path, bytes.NewReader(jsonData))
		if err != nil {
			return fmt.Errorf("creating request: %w", err)
		}
		httpReq.Header.Set("Content-Type", "application/json")

		httpResp, err := c.httpClient.Do(httpReq)
		if err != nil {
			lastErr = err
			// Only retry on connection errors
			continue
		}
		defer httpResp.Body.Close()

		body, err := io.ReadAll(httpResp.Body)
		if err != nil {
			return fmt.Errorf("reading response: %w", err)
		}

		if httpResp.StatusCode != http.StatusOK {
			return fmt.Errorf("engine returned status %d: %s", httpResp.StatusCode, string(body))
		}

		if err := json.Unmarshal(body, respBody); err != nil {
			return fmt.Errorf("unmarshaling response: %w", err)
		}

		return nil
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

// StreamChunk represents a single SSE chunk in a stream
type StreamChunk struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Delta struct {
			Content string `json:"content,omitempty"`
			Role    string `json:"role,omitempty"`
		} `json:"delta"`
		Index        int    `json:"index"`
		FinishReason string `json:"finish_reason,omitempty"`
	} `json:"choices"`
}

// StreamCompletionChunk represents completion stream chunk
type StreamCompletionChunk struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Text         string `json:"text"`
		Index        int    `json:"index"`
		FinishReason string `json:"finish_reason,omitempty"`
	} `json:"choices"`
}

// CreateCompletionStream sends a streaming completion request
func (c *Client) CreateCompletionStream(ctx context.Context, req *CompletionRequest) (<-chan StreamCompletionChunk, <-chan error) {
	chunkChan := make(chan StreamCompletionChunk)
	errChan := make(chan error, 1)

	go func() {
		defer close(chunkChan)
		defer close(errChan)

		// Ensure stream is enabled
		req.Stream = true
		jsonData, err := json.Marshal(req)
		if err != nil {
			errChan <- fmt.Errorf("marshaling request: %w", err)
			return
		}

		httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/completions", bytes.NewReader(jsonData))
		if err != nil {
			errChan <- fmt.Errorf("creating request: %w", err)
			return
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Accept", "text/event-stream")

		httpResp, err := c.httpClient.Do(httpReq)
		if err != nil {
			errChan <- fmt.Errorf("sending request: %w", err)
			return
		}
		defer httpResp.Body.Close()

		if httpResp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(httpResp.Body)
			errChan <- fmt.Errorf("engine returned status %d: %s", httpResp.StatusCode, string(body))
			return
		}

		// Read SSE stream
		buffer := make([]byte, 4096)
		var leftover []byte

		for {
			select {
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			default:
			}

			n, err := httpResp.Body.Read(buffer)
			if n > 0 {
				data := append(leftover, buffer[:n]...)
				chunks, remaining := c.parseSSEChunks(data)
				leftover = remaining

				for _, chunk := range chunks {
					if chunk == "[DONE]" {
						return
					}

					var streamChunk StreamCompletionChunk
					if err := json.Unmarshal([]byte(chunk), &streamChunk); err != nil {
						continue // Skip malformed chunks
					}

					select {
					case chunkChan <- streamChunk:
					case <-ctx.Done():
						return
					}
				}
			}

			if err == io.EOF {
				return
			}
			if err != nil {
				errChan <- fmt.Errorf("reading stream: %w", err)
				return
			}
		}
	}()

	return chunkChan, errChan
}

// CreateChatCompletionStream sends a streaming chat completion request
func (c *Client) CreateChatCompletionStream(ctx context.Context, req *ChatCompletionRequest) (<-chan StreamChunk, <-chan error) {
	chunkChan := make(chan StreamChunk)
	errChan := make(chan error, 1)

	go func() {
		defer close(chunkChan)
		defer close(errChan)

		// Ensure stream is enabled
		req.Stream = true
		jsonData, err := json.Marshal(req)
		if err != nil {
			errChan <- fmt.Errorf("marshaling request: %w", err)
			return
		}

		httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/chat/completions", bytes.NewReader(jsonData))
		if err != nil {
			errChan <- fmt.Errorf("creating request: %w", err)
			return
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Accept", "text/event-stream")

		httpResp, err := c.httpClient.Do(httpReq)
		if err != nil {
			errChan <- fmt.Errorf("sending request: %w", err)
			return
		}
		defer httpResp.Body.Close()

		if httpResp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(httpResp.Body)
			errChan <- fmt.Errorf("engine returned status %d: %s", httpResp.StatusCode, string(body))
			return
		}

		// Read SSE stream
		buffer := make([]byte, 4096)
		var leftover []byte

		for {
			select {
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			default:
			}

			n, err := httpResp.Body.Read(buffer)
			if n > 0 {
				data := append(leftover, buffer[:n]...)
				chunks, remaining := c.parseSSEChunks(data)
				leftover = remaining

				for _, chunk := range chunks {
					if chunk == "[DONE]" {
						return
					}

					var streamChunk StreamChunk
					if err := json.Unmarshal([]byte(chunk), &streamChunk); err != nil {
						continue // Skip malformed chunks
					}

					select {
					case chunkChan <- streamChunk:
					case <-ctx.Done():
						return
					}
				}
			}

			if err == io.EOF {
				return
			}
			if err != nil {
				errChan <- fmt.Errorf("reading stream: %w", err)
				return
			}
		}
	}()

	return chunkChan, errChan
}

// parseSSEChunks parses Server-Sent Events format
func (c *Client) parseSSEChunks(data []byte) (chunks []string, remaining []byte) {
	lines := bytes.Split(data, []byte("\n\n"))
	
	// Last element might be incomplete
	if len(lines) > 0 {
		remaining = lines[len(lines)-1]
		lines = lines[:len(lines)-1]
	}

	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		// SSE format: "data: {json}\n\n"
		if bytes.HasPrefix(line, []byte("data: ")) {
			chunk := bytes.TrimPrefix(line, []byte("data: "))
			chunks = append(chunks, string(bytes.TrimSpace(chunk)))
		}
	}

	return chunks, remaining
}
