# Content Moderation Service

A high-performance content moderation service built with Go (Kratos framework) featuring multi-layer text filtering and AI-powered image NSFW detection.

## Features

- **Text Moderation**: Bloom filter → Aho-Corasick → LLM (LlamaGuard via vLLM)
- **Image Moderation**: NSFW detection using Falconsai/nsfw_image_detection
- **Feedback Loop**: Flagged content is saved and added to Bloom filter for faster future detection
- **Self-hosted**: All AI models run locally, no external API calls

## Architecture
![architecture](./docs/architecture.png)
## Quick Start

### 1. Start Services

```bash
# Clone and enter directory
cd moderation

# Start all services with Docker Compose
docker-compose up -d

# Check status
docker-compose ps
```

### 2. Services

| Service | Port | Description |
|---------|------|-------------|
| vLLM | 8000 | Text moderation (LlamaGuard-3-1B) |
| nsfw-detector | 8081 | Image NSFW detection |
| moderation | 8082/9000 | Main service (HTTP/gRPC) |
| postgres | 5432 | Database for bad words |
| redis | 6379 | Bloom filter storage |

## API Usage

### Text Moderation (vLLM)

```bash
curl http://localhost:8000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "meta-llama/Llama-Guard-3-1B",
    "messages": [{"role": "user", "content": "Hello, how are you?"}]
  }'
```

### Image NSFW Detection

**Single URL:**
```bash
curl -X POST http://localhost:8081/predict/url \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com/image.jpg"}'
```

**Batch URLs:**
```bash
curl -X POST http://localhost:8081/predict/batch/url \
  -H "Content-Type: application/json" \
  -d '{"urls": ["https://example.com/img1.jpg", "https://example.com/img2.jpg"]}'
```

**Upload File:**
```bash
curl -X POST http://localhost:8081/predict \
  -F "file=@image.jpg"
```

**Response:**
```json
{
  "is_nsfw": false,
  "nsfw_score": 0.02,
  "normal_score": 0.98,
  "label": "normal",
  "confidence": 0.98
}
```

## Go SDK Usage

### Text Moderation

```go
import "moderation/internal/pkg/llm"

// Create client
client := llm.NewVLLMClient(llm.VLLMConfig{
    BaseURL: "http://localhost:8000",
    Model:   "meta-llama/Llama-Guard-3-1B",
})

// Moderate text
result, err := client.ModerateText(ctx, "user message")
if err != nil {
    log.Fatal(err)
}

if !result.IsSafe {
    fmt.Println("Unsafe content:", result.ViolatedCategories)
}
```

### Image NSFW Detection

```go
import "moderation/internal/pkg/nsfw"

// Create client
client := nsfw.NewClient(nsfw.Config{
    BaseURL: "http://localhost:8081",
})

// Single URL
result, err := client.DetectFromURL(ctx, "https://example.com/image.jpg")
if result.IsNSFW {
    fmt.Printf("NSFW detected! Score: %.2f\n", result.NSFWScore)
}

// Batch URLs
results, err := client.DetectFromURLs(ctx, []string{
    "https://example.com/img1.jpg",
    "https://example.com/img2.jpg",
})
for _, r := range results {
    if r.Error != nil {
        log.Printf("Error: %s - %v", r.URL, r.Error)
    } else {
        log.Printf("%s: NSFW=%v, Score=%.2f", r.URL, r.Result.IsNSFW, r.Result.NSFWScore)
    }
}

// From bytes
result, err := client.DetectFromBytes(ctx, imageBytes)
```

## Text Moderation Flow

![text-moderation-flow](./docs/text-moderation-flow.png)

**Feedback Loop Benefit:**
- First occurrence: ~50ms (full LLM check)
- Second occurrence: ~5ms (Bloom + Aho-Corasick, no LLM!)

## Configuration

### Environment Variables

```bash
# HuggingFace token (for gated models)
export HF_TOKEN=your_token

# Database
DATABASE_URL=postgres://user:pass@localhost:5432/moderation

# Redis
REDIS_ADDR=localhost:6379

# vLLM
VLLM_URL=http://localhost:8000
VLLM_MODEL=meta-llama/Llama-Guard-3-1B

# NSFW Detector
NSFW_URL=http://localhost:8081
```

### CPU Only Mode

Remove GPU reservations from `docker-compose.yml`:

```yaml
# Remove this section from vllm and nsfw-detector services:
deploy:
  resources:
    reservations:
      devices:
        - driver: nvidia
          count: 1
          capabilities: [gpu]
```

## Development

### Build

```bash
make build
```

### Run Tests

```bash
go test ./...
```

### Generate Proto

```bash
make config
make api
```

## Project Structure

```
moderation/
├── api/moderation/v1/       # Proto definitions
├── cmd/moderation/          # Entry point
├── configs/                 # Configuration files
├── deployments/
│   └── nsfw-detector/       # Python NSFW API
├── docs/                    # Documentation
├── internal/
│   ├── biz/                 # Business logic
│   ├── conf/                # Config proto
│   ├── data/                # Data layer
│   ├── pkg/
│   │   ├── bloom/           # Redis Bloom filter
│   │   ├── filter/          # Aho-Corasick
│   │   ├── hash/            # pHash
│   │   ├── llm/             # vLLM/Ollama client
│   │   ├── moderator/       # Moderation engines
│   │   └── nsfw/            # NSFW detection client
│   ├── server/              # HTTP/gRPC servers
│   └── service/             # Service implementations
└── docker-compose.yml       # Full stack deployment
```

## License

MIT
