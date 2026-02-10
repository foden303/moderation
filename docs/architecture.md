# Content Moderation Service - Architecture

## System Overview

```mermaid
graph TB
    subgraph CLIENT["Clients"]
        HTTP[HTTP Client]
        GRPC[gRPC Client]
    end

    subgraph API["API Layer"]
        direction TB
        SRV_HTTP[HTTP Server :8000]
        SRV_GRPC[gRPC Server :9000]
        MS[ModerationService]
        AS[AdminService]
    end

    subgraph BIZ["Business Layer"]
        MU[ModerationUsecase]
        BU[BadwordUsecase]
    end

    subgraph MODERATORS["Moderation Engines"]
        TM[TextModerator]
        IM[ImageModerator]
        VM[VideoModerator]
    end

    subgraph PKG["Core Packages"]
        BF[BloomFilter<br/>~1ms]
        AC[Aho-Corasick<br/>~5ms]
        PH[pHash<br/>~10ms]
        OL[Ollama Client<br/>~500ms]
        NSFW[NSFW Detector<br/>~100ms]
    end

    subgraph INFRA["Infrastructure"]
        PG[(PostgreSQL<br/>bad_words/bad_images)]
        RD[(Redis<br/>bloom bits)]
        OLLAMA[Ollama<br/>LlamaGuard/Qwen]
        NSFW_API[NSFW API<br/>Falconsai]
    end

    HTTP --> SRV_HTTP
    GRPC --> SRV_GRPC
    SRV_HTTP --> MS
    SRV_HTTP --> AS
    SRV_GRPC --> MS
    SRV_GRPC --> AS

    MS --> MU
    AS --> BU
    AS --> MU

    MU --> TM
    MU --> IM
    MU --> VM

    TM --> BF
    TM --> AC
    TM --> OL
    IM --> PH
    IM --> BF
    IM --> NSFW
    VM --> IM

    BF -.-> RD
    BU --> PG
    OL --> OLLAMA
    NSFW --> NSFW_API
```

## Moderation Flow

```mermaid
flowchart TD
    A[Text Input] --> B{Bloom Filter<br/>~1ms}
    B -->|Maybe| C{Aho-Corasick<br/>~5ms}
    B -->|No| D{LLM Check<br/>~500ms}
    
    C -->|Match| E[REJECT]
    C -->|No Match| D
    
    D -->|Unsafe| F[Save to DB]
    F --> G[Update Bloom]
    G --> E
    
    D -->|Safe| H[ALLOW]
    
    style E fill:#f66,color:#fff
    style H fill:#6f6,color:#fff
    style B fill:#69f,color:#fff
    style C fill:#69f,color:#fff
    style D fill:#f96,color:#fff
```

## Feedback Loop

```mermaid
graph LR
    subgraph FIRST["First Occurrence ~500ms"]
        A1[Input] --> B1[Bloom: NO]
        B1 --> C1[LLM: UNSAFE]
        C1 --> D1[Save DB]
        D1 --> E1[Update Bloom]
    end
    
    subgraph SECOND["Second Occurrence ~5ms"]
        A2[Same Input] --> B2[Bloom: YES]
        B2 --> C2[Aho: MATCH]
        C2 --> D2[REJECT]
    end
    
    E1 -.->|learns| B2
    
    style D2 fill:#f66,color:#fff
```

## Text Moderation Sequence

```mermaid
sequenceDiagram
    participant C as Client
    participant T as TextModerator
    participant B as BloomFilter
    participant A as Aho-Corasick
    participant O as Ollama
    participant DB as PostgreSQL

    C->>T: Moderate(text)

    Note over T,B: Step 1: Fast Check
    T->>B: Exists(words)?
    B-->>T: maybe/no

    alt Bloom = YES
        T->>A: Search(text)
        A-->>T: matches[]
        T-->>C: REJECT
    else Bloom = NO
        T->>O: ModerateText(text)
        alt LLM = UNSAFE
            O-->>T: unsafe
            T->>DB: SaveBadWord
            T->>B: Add(phrase)
            T-->>C: REJECT
        else LLM = SAFE
            O-->>T: safe
            T-->>C: ALLOW
        end
    end
```

## Package Structure

```
moderation/
├── internal/
│   ├── biz/                   # Business logic
│   │   ├── moderation.go
│   │   └── badword.go
│   ├── pkg/                   # Core packages
│   │   ├── bloom/             # Redis Bloom filter
│   │   ├── filter/            # Aho-Corasick
│   │   ├── hash/              # pHash
│   │   ├── llm/               # Ollama client
│   │   └── moderator/         # Engines
│   ├── service/
│   └── data/
└── docker-compose.yml
```

## Components

| Component | Purpose | Latency |
|-----------|---------|---------|
| BloomFilter | Fast probabilistic check | ~1ms |
| Aho-Corasick | Multi-pattern matching | ~5ms |
| pHash | Image duplicate detection | ~10ms |
| VLLM | Deep content analysis | ~50ms |
| Ollama LLM (Optional) | Deep content analysis | ~500ms |

## Docker Stack

```yaml
services:
  ollama:       # LLM Server (11434)
  postgres:     # Database (5432)
  redis:        # Bloom Filter (6379)
  moderation:   # Service (8000/9000)
```

## Performance Summary

| Occurrence | Path | Latency |
|------------|------|---------|
| First | Bloom→Aho→LLM→Save | ~500ms |
| Second+ | Bloom→Aho | ~5ms |

