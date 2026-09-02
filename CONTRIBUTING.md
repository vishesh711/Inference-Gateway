# Contributing to Inference Gateway

Thank you for your interest in contributing! This document provides guidelines for contributing to the project.

## Development Setup

1. **Prerequisites**
   - Go 1.21 or higher
   - Git
   - Make (optional, but recommended)

2. **Clone and build**
   ```bash
   git clone https://github.com/vishesh/inference-gateway
   cd inference-gateway
   make deps
   make build
   ```

3. **Run tests**
   ```bash
   make test
   ```

## Code Style

- Follow standard Go conventions
- Run `go fmt` before committing
- Add comments for exported functions and types
- Keep functions focused and under 50 lines when possible

## Project Structure

```
.
├── cmd/              # Command-line applications
│   ├── gateway/      # Main gateway service
│   └── loadgen/      # Load testing tool
├── internal/         # Private application code
│   ├── cache/        # Response caching
│   ├── config/       # Configuration management
│   ├── cost/         # Cost accounting
│   ├── engine/       # LLM engine client
│   ├── handler/      # HTTP request handlers
│   ├── metrics/      # Prometheus metrics
│   └── scheduler/    # Admission control and batching
└── scripts/          # Helper scripts
```

## Testing Guidelines

- Write unit tests for new functionality
- Ensure tests are deterministic
- Use table-driven tests where appropriate
- Mock external dependencies

Example test structure:
```go
func TestAdmissionController(t *testing.T) {
    tests := []struct {
        name    string
        queueSize int
        wantErr bool
    }{
        {"accepts when queue not full", 10, false},
        {"rejects when queue full", 0, true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test implementation
        })
    }
}
```

## Pull Request Process

1. **Create a feature branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes**
   - Write clean, documented code
   - Add tests for new functionality
   - Update documentation if needed

3. **Verify everything works**
   ```bash
   make fmt
   make test
   make build
   ```

4. **Commit with clear messages**
   ```bash
   git commit -m "feat: add priority queueing support"
   ```

   Follow conventional commits format:
   - `feat:` for new features
   - `fix:` for bug fixes
   - `docs:` for documentation
   - `refactor:` for code refactoring
   - `test:` for adding tests
   - `perf:` for performance improvements

5. **Push and create PR**
   ```bash
   git push origin feature/your-feature-name
   ```

6. **PR Requirements**
   - Clear description of changes
   - Tests pass
   - Code is formatted
   - No unnecessary dependencies added

## Feature Ideas

Looking for something to work on? Here are some ideas:

**High Priority:**
- [ ] Streaming response support via SSE
- [ ] Per-client rate limiting with token buckets
- [ ] Priority queueing (high/low priority requests)
- [ ] Circuit breaker with exponential backoff

**Medium Priority:**
- [ ] Prefix-aware request routing
- [ ] Multiple backend engine support
- [ ] Request tracing with OpenTelemetry
- [ ] Admin API for runtime configuration

**Nice to Have:**
- [ ] WebUI for monitoring
- [ ] Request replay from logs
- [ ] A/B testing between models
- [ ] Request validation middleware

## Architectural Principles

When contributing, keep these principles in mind:

1. **Honesty about what each layer does**  
   The gateway controls admission and concurrency. The engine does continuous batching and KV caching. Be explicit about this distinction.

2. **Fail fast and explicitly**  
   Bounded queues with 429s are better than unbounded queues that timeout. Circuit breakers should fail fast when backends are down.

3. **Context propagation**  
   Every request should carry a context. When a client disconnects, the work should stop immediately.

4. **Measure everything**  
   If you can't measure it, you can't optimize it. Add metrics for new features.

5. **Simplicity over cleverness**  
   Go's standard library is usually sufficient. Avoid dependencies unless they solve a real problem.

## Questions?

- Open an issue for bugs or feature requests
- Start a discussion for design questions
- Check existing issues before creating new ones

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
