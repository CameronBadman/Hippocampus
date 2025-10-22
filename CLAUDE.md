# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Hippocampus is a vector database built specifically for AI agents, not retrofitted from document search. It's designed for the 5k-10k vector scale per agent—not billion-scale retrieval. Think "SQLite of AI agent memory": file-based, serverless, production-ready.

### Core Concept

Two interaction patterns:
1. **Agent-Controlled**: Agents directly manage their memory (insert/search with full parameter control)
2. **Database-Curated**: Simple agents pass raw text; internal AI agent decomposes it into discrete, searchable memories

## Development Commands

### Building

```bash
# Build CLI tool (default make target)
make build-cli
# Output: bin/hippocampus

# Build Lambda function (for deployment)
make build-lambda
# Output: terraform/bootstrap

# Build both
make all

# Compile manually
CGO_ENABLED=0 go build -o bin/hippocampus src/cmd/cli/main.go
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -tags lambda.norpc -o terraform/bootstrap src/lambda/main.go
```

### Testing

```bash
# Run all Go tests
make test
go test ./src/...
```

### Deployment

```bash
# Full AWS deployment (builds Lambda, runs terraform apply)
make deploy

# Manual Terraform workflow
cd terraform
terraform init
terraform plan
terraform apply
```

### Cleanup

```bash
make clean  # Removes bin/, terraform/bootstrap, terraform/lambda.zip, terraform/.terraform*
```

### CLI Usage

```bash
# After building with 'make build-cli', use the local CLI:
./bin/hippocampus

# Insert a memory
./bin/hippocampus insert -binary tree.bin -key "user_preference" -text "User prefers dark mode"

# Search with full control
./bin/hippocampus search -binary tree.bin -text "UI settings" -epsilon 0.3 -threshold 0.5 -top-k 5

# Agent curation (AI decomposes text into discrete memories)
./bin/hippocampus agent-curate -binary tree.bin -text "Sarah, 34, Google engineer, allergic to shellfish" -importance high

# Bulk insert from CSV
./bin/hippocampus insert-csv -binary tree.bin -csv data.csv

# All commands support custom AWS region
./bin/hippocampus insert -region us-west-2 -binary tree.bin -key "test" -text "sample"
```

### Demo Scripts

```bash
# Python demos (in demo/python/)
python safety_demo.py              # Safety-critical memory scenario (shellfish allergy)
python agentcore_agent.py          # Agent-to-agent orchestration demo
python agentcore_agent.py --interactive  # Interactive mode
python customer_support_demo.py    # Customer support agent scenario
python basic_agent.py              # Simple insert/search example
```

## Architecture

### Core Algorithm

Custom 512-dimensional vector indexing:
- **Not** HNSW/IVF approximation—guaranteed exact retrieval within epsilon-ball
- 512 sorted arrays (one per dimension) for O(log n) binary search
- Candidate filtering: nodes must appear in epsilon-ball for ALL 512 dimensions
- Final Euclidean distance check with threshold filter
- Results sorted by similarity, limited to top-k

### Data Structures (src/types/types.go)

```go
type Node struct {
    Key   [512]float32  // Titan embedding vector
    Value string        // Actual memory text
}

type Tree struct {
    Nodes []Node           // Linear array of all nodes
    Index [512][]int32    // Per-dimension sorted indices into Nodes
}
```

**Key operations:**
- `Insert()`: Adds node, updates all 512 sorted indices
- `RebuildIndex()`: Called after bulk load from disk (~50ms for 5k nodes)
- `Search()`: Binary search each dimension, compute distances, filter/sort

### Storage Layer (src/storage/)

**Binary serialization** (storage/storage.go):
- Custom format: ~2KB per node (512 floats × 4 bytes + value string)
- File structure: node count (8 bytes) + nodes (sequential)
- Each agent gets isolated `.bin` file

**Multi-agent manager** (lambda/storage/manager.go):
- Lazy client loading (only loads requested agent's file)
- Per-agent client caching (map[string]*client.Client)
- Automatic S3 async backup after every insert (no latency impact)
- S3→EFS download if agent file not found locally

### Lambda Execution Flow (src/lambda/)

1. **API Gateway** receives POST to `/insert`, `/search`, `/agent-curate`, `/insert-csv`
2. **handlers/handlers.go**: Routes request to appropriate handler
3. **storage/manager.go**: Gets or creates per-agent client
4. **Load agent's .bin** from EFS (or S3 if not cached)
5. **RebuildIndex()** in memory (512 sorts)
6. **Execute operation** (embedding via Bedrock Titan, then search/insert)
7. **Async S3 backup** (goroutine, doesn't block response)

### Agent Curation Flow (client/client.go)

When `AgentCurate()` is called (via CLI or Lambda API):
1. Internal AI agent (AWS Bedrock Nova) analyzes text
2. Extracts discrete facts based on importance level (high/medium/low)
3. Generates structured memories with keys (e.g., `allergy_shellfish`)
4. Inserts each memory individually with configurable delay (prevents rate limiting)
5. Returns all created memories with reasoning

This is **agent-to-agent orchestration**: external agent calls Hippocampus, which uses its own internal agent for curation.

**Location**: Logic lives in `client/client.go` (client.AgentCurate), shared by both CLI and Lambda handlers.

### AWS Infrastructure (terraform/main.tf)

**Serverless stack:**
- **VPC**: Public/private subnets across 2 AZs
- **NAT Gateway**: Lambda → Bedrock API calls
- **EFS**: Hot storage for per-agent `.bin` files (mounted at `/mnt/efs`)
- **S3**: Cold backup storage (versioning enabled)
- **Lambda**: 1024MB RAM, 60s timeout, 2GB ephemeral storage
- **API Gateway v2**: HTTP API with routes for all endpoints

**Why this scales:** Lambda is stateless (pure function). No database bottlenecks. Scales as hard as Lambda scales (default: 1000 concurrent executions).

### Embedding Integration (src/embedding/titan.go)

- AWS Bedrock Titan Text Embeddings v2
- Hardcoded to 512 dimensions
- Called for every insert/search (text → vector)
- Requires IAM permission: `bedrock:InvokeModel`

## Module Organization

```
src/
├── types/              Core Tree/Node, Insert/Search algorithms
├── storage/            Binary file serialization (Save/Load)
├── embedding/          AWS Bedrock Titan integration
├── client/             High-level API wrapping tree + storage + embedding + agent curation
├── cmd/cli/            Command-line interface (full feature parity with Lambda API)
└── lambda/
    ├── handlers/       HTTP routing, request validation, delegates to client
    └── storage/        Multi-agent manager, S3 sync, client caching

terraform/              Complete AWS infrastructure
demo/python/            Agent integration examples
bin/                    Built CLI binary (hippocampus)
```

## Search Parameter Tuning

**Epsilon** (per-dimension bounding box):
- 0.15-0.2: Precise fact lookup
- 0.3: Balanced (default)
- 0.4-0.5: Broad exploration

**Threshold** (distance filter, higher = stricter):
- 0.7+: Safety-critical queries
- 0.5-0.6: General search (default)
- 0.4: Discovery mode

**Top-K** (result limit):
- 1-3: Precise answers
- 5: General queries (default)
- 10: Comprehensive search

## Important Implementation Details

### CLI vs Lambda Architecture
The codebase supports two execution modes with shared core logic:

1. **CLI Mode** (src/cmd/cli/main.go):
   - Local file-based storage (tree.bin by default)
   - Direct client library usage
   - Single-agent per file (no multi-tenancy needed)
   - Uses client.AgentCurate() for curation
   - Perfect for local development, testing, and embedded use cases

2. **Lambda Mode** (src/lambda/main.go):
   - Multi-agent storage manager (EFS + S3)
   - HTTP API Gateway routing
   - Per-agent isolation via storage.Manager
   - Delegates to client methods for operations
   - Production serverless deployment

**Key insight**: All business logic (search, insert, agent curation) lives in `client/` package. Lambda handlers are just thin HTTP wrappers around the client.

### File-Based Multi-Tenancy
Each agent gets isolated storage: `/agents/agent_abc123.bin`. No shared state, no locks, no coordination. Lambda loads only the requested agent's file.

### Index Rebuild on Load
Binary files store nodes sequentially, NOT sorted indices. On load, we rebuild all 512 sorted indices in memory (~50ms). This is faster than serializing/deserializing sorted arrays.

### Candidate Set Filtering
Search requires nodes to appear in ALL 512 dimension's epsilon-balls (count == 512). This drastically reduces false positives before distance calculation.

### Async S3 Backup
After insert, we `go m.s3Sync.Upload()` in goroutine. Lambda response returns immediately; backup happens in background.

### Agent Curation Timeout
`timeout_ms` param in `/agent-curate` controls delay between memory insertions. Prevents hitting Bedrock rate limits when decomposing large texts into 20+ memories.

## AWS Credentials & Configuration

**Prerequisites:**
- `aws configure` with valid credentials
- IAM permissions for Bedrock (Titan + Nova), Lambda, EFS, S3, VPC
- Region with Bedrock model access (default: ap-southeast-2 for API, us-east-1 for Bedrock)

**Terraform variables** (terraform/terraform.tfvars):
- `aws_region`: Where to deploy infrastructure
- `s3_bucket_name`: Unique S3 bucket for backups

## Common Gotchas

- **Cold start**: First request to new agent takes ~200ms (EFS mount + index rebuild)
- **Lambda timeout**: Default 60s, can be increased if processing large texts
- **Bedrock rate limits**: Use `timeout_ms` in agent-curate for bulk insertions
- **VPC networking**: Lambda must be in private subnet with NAT Gateway for Bedrock access
- **File size**: ~10MB per 5k nodes. Monitor EFS usage at scale.

## Performance Benchmarks (5k nodes per agent)

- **File size**: ~10MB
- **RAM usage**: ~20MB in Lambda (warm)
- **Cold start**: ~200ms (load + index rebuild)
- **Search**: <50ms (warm)
- **Insert**: ~100ms (includes Bedrock embedding call)
- **Cost**: ~$0.00012 per memory operation

## Design Philosophy

This is NOT a billion-scale vector database. It's optimized for:
- Per-agent isolated memory (5k-10k vectors)
- Deterministic, debuggable results (exact search, not approximate)
- File-based simplicity (no servers, no clusters)
- Serverless execution (Lambda + EFS + S3)

If you need billion-scale, use Pinecone/Weaviate. If you need agent memory that "just works", use Hippocampus.
