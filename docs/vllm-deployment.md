# vLLM + Ray Deployment for Content Moderation

## Docker Compose

```yaml
version: "3.8"

services:
  # vLLM Server with Ray
  vllm:
    image: vllm/vllm-openai:latest
    container_name: vllm-server
    ports:
      - "8000:8000"
    volumes:
      - vllm_cache:/root/.cache/huggingface
    environment:
      - HF_TOKEN=${HF_TOKEN}  # Hugging Face token for gated models
    command: |
      --model meta-llama/Llama-Guard-3-1B
      --tensor-parallel-size 1
      --max-model-len 4096
      --gpu-memory-utilization 0.9
    deploy:
      resources:
        reservations:
          devices:
            - driver: nvidia
              count: 1
              capabilities: [gpu]
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8000/v1/models"]
      interval: 30s
      timeout: 10s
      retries: 3

  # Redis for Bloom Filter
  redis:
    image: redis:7-alpine
    container_name: moderation-redis
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    command: redis-server --appendonly yes

  # PostgreSQL
  postgres:
    image: postgres:16-alpine
    container_name: moderation-postgres
    ports:
      - "5432:5432"
    environment:
      POSTGRES_USER: moderation
      POSTGRES_PASSWORD: moderation123
      POSTGRES_DB: moderation
    volumes:
      - postgres_data:/var/lib/postgresql/data

  # Moderation Service
  moderation:
    build: .
    container_name: moderation-service
    ports:
      - "8080:8000"
      - "9000:9000"
    environment:
      - DATABASE_URL=postgres://moderation:moderation123@postgres:5432/moderation?sslmode=disable
      - REDIS_ADDR=redis:6379
      - VLLM_URL=http://vllm:8000
      - VLLM_MODEL=meta-llama/Llama-Guard-3-1B
    depends_on:
      - postgres
      - redis
      - vllm

volumes:
  vllm_cache:
  redis_data:
  postgres_data:
```

## Commands

```bash
# Start vLLM with LlamaGuard
docker-compose up -d vllm

# Check if model is loaded
curl http://localhost:8000/v1/models

# Test moderation
curl http://localhost:8000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "meta-llama/Llama-Guard-3-1B",
    "messages": [{"role": "user", "content": "Hello world"}]
  }'
```

## Performance Comparison

| Backend | Latency | Throughput | GPU Memory |
|---------|---------|------------|------------|
| Ollama | ~500ms | 2 req/s | 4GB |
| vLLM | ~50ms | 50+ req/s | 6GB |

## Ray Cluster (Optional)

For horizontal scaling with Ray:

```bash
# Start Ray head node
ray start --head --port=6379

# Start vLLM with Ray
vllm serve meta-llama/Llama-Guard-3-1B \
  --tensor-parallel-size 2 \
  --pipeline-parallel-size 2
```
