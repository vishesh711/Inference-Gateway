#!/usr/bin/env python3
"""Mock LLM engine for testing the Inference Gateway"""
import json
import time
import random
from http.server import HTTPServer, BaseHTTPRequestHandler

class MockLLMHandler(BaseHTTPRequestHandler):
    def do_POST(self):
        if self.path in ['/v1/completions', '/v1/chat/completions', '/v1/embeddings']:
            content_length = int(self.headers['Content-Length'])
            body = self.rfile.read(content_length)
            req = json.loads(body)
            
            # Simulate realistic processing time (50-200ms)
            time.sleep(random.uniform(0.05, 0.2))
            
            if self.path == '/v1/completions':
                response = self.handle_completion(req)
            elif self.path == '/v1/chat/completions':
                response = self.handle_chat_completion(req)
            elif self.path == '/v1/embeddings':
                response = self.handle_embeddings(req)
            
            self.send_response(200)
            self.send_header('Content-type', 'application/json')
            self.end_headers()
            self.wfile.write(json.dumps(response).encode())
        
        elif self.path == '/health':
            self.send_response(200)
            self.send_header('Content-type', 'text/plain')
            self.end_headers()
            self.wfile.write(b'OK')
        
        else:
            self.send_response(404)
            self.end_headers()
    
    def do_GET(self):
        if self.path == '/health':
            self.send_response(200)
            self.send_header('Content-type', 'text/plain')
            self.end_headers()
            self.wfile.write(b'OK')
        else:
            self.send_response(404)
            self.end_headers()
    
    def handle_completion(self, req):
        prompt = req.get("prompt", "")
        max_tokens = req.get("max_tokens", 50)
        
        # Generate mock completion
        completions = [
            " Paris. It is known for the Eiffel Tower and the Louvre Museum.",
            " the center of art, culture, and cuisine in Europe.",
            " a beautiful city with rich history and architecture."
        ]
        
        completion_text = random.choice(completions)
        prompt_tokens = len(prompt.split())
        completion_tokens = len(completion_text.split())
        
        return {
            "id": f"mock-{random.randint(1000, 9999)}",
            "object": "text_completion",
            "created": int(time.time()),
            "model": req.get("model", "mock-model"),
            "choices": [{
                "text": completion_text,
                "index": 0,
                "finish_reason": "stop"
            }],
            "usage": {
                "prompt_tokens": prompt_tokens,
                "completion_tokens": completion_tokens,
                "total_tokens": prompt_tokens + completion_tokens
            }
        }
    
    def handle_chat_completion(self, req):
        messages = req.get("messages", [])
        
        # Generate mock response based on last message
        last_msg = messages[-1].get("content", "") if messages else ""
        
        responses = [
            "Hello! I'm a mock AI assistant. How can I help you today?",
            "That's an interesting question! In a real system, I would provide a detailed answer.",
            "I understand. Let me help you with that.",
            "Great question! This mock engine simulates realistic response times."
        ]
        
        response_text = random.choice(responses)
        prompt_tokens = sum(len(m.get("content", "").split()) for m in messages)
        completion_tokens = len(response_text.split())
        
        return {
            "id": f"mock-chat-{random.randint(1000, 9999)}",
            "object": "chat.completion",
            "created": int(time.time()),
            "model": req.get("model", "mock-model"),
            "choices": [{
                "message": {
                    "role": "assistant",
                    "content": response_text
                },
                "index": 0,
                "finish_reason": "stop"
            }],
            "usage": {
                "prompt_tokens": prompt_tokens,
                "completion_tokens": completion_tokens,
                "total_tokens": prompt_tokens + completion_tokens
            }
        }
    
    def handle_embeddings(self, req):
        inputs = req.get("input", [])
        if isinstance(inputs, str):
            inputs = [inputs]
        
        # Generate mock embeddings (dimension 384)
        data = []
        for i, text in enumerate(inputs):
            embedding = [random.uniform(-1, 1) for _ in range(384)]
            data.append({
                "object": "embedding",
                "embedding": embedding,
                "index": i
            })
        
        prompt_tokens = sum(len(text.split()) for text in inputs)
        
        return {
            "object": "list",
            "data": data,
            "model": req.get("model", "mock-embedding"),
            "usage": {
                "prompt_tokens": prompt_tokens,
                "total_tokens": prompt_tokens
            }
        }
    
    def log_message(self, format, *args):
        timestamp = time.strftime('%Y/%m/%d %H:%M:%S')
        print(f"[{timestamp}] Mock Engine: {format % args}")

if __name__ == '__main__':
    server = HTTPServer(('localhost', 8080), MockLLMHandler)
    print("=" * 70)
    print("Mock LLM Engine")
    print("=" * 70)
    print("Listening on: http://localhost:8080")
    print("")
    print("Endpoints:")
    print("  POST /v1/completions")
    print("  POST /v1/chat/completions")
    print("  POST /v1/embeddings")
    print("  GET  /health")
    print("")
    print("This mock engine simulates realistic response times (50-200ms)")
    print("and returns mock data for testing the gateway.")
    print("")
    print("Press Ctrl+C to stop")
    print("=" * 70)
    print("")
    
    try:
        server.serve_forever()
    except KeyboardInterrupt:
        print("\nShutting down mock engine...")
        server.shutdown()
