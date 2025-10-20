# Hippocampus

A production-ready semantic memory system for AI agents built on AWS. Simple, fast, and debuggable.

## What is this?

Hippocampus is a simple, stable vector database optimized for AI agent memory. No HNSW graphs, no quantization, no clustering—just sorted arrays and binary search. It runs as both a CLI tool for local development and a serverless Lambda API for production multi-agent systems.

Built for the **99% use case**: 5k-10k vectors per agent that need to be fast, deterministic, and debuggable.

## Why?

Most vector databases are over-engineered for billion-scale problems you don't have. Hippocampus embraces simplicity:

- **Simple**: Sorted arrays and binary search—algorithms from CS101
- **Stable**: Same query always returns same results, deterministic ordering
- **Fast**: O(log n × 512) binary searches, microsecond queries on 5k nodes
- **Debuggable**: No black-box indexes, inspect the data structures directly
- **Production-ready**: Serverless Lambda deployment with EFS persistence and S3 backups
- **Multi-agent**: Isolated storage per agent, lazy-loaded from S3

## Quick Start

### Local CLI Tool

```bash
# Build the CLI
make build-cli

# Insert memories
./bin/hippocampus insert -key "meeting_notes" -text "Discussed Q4 roadmap with engineering team"

# Semantic search
./bin/hippocampus search -text "engineering planning" -e 0.2

# Bulk import
./bin/hippocampus insert-csv -csv memories.csv
```

### Deploy to AWS Lambda

```bash
# Build and deploy serverless API
make deploy

# Test the deployed API
bash test.sh
```

### Use from Python Agents

```python
import requests

API = "https://your-api.execute-api.us-east-1.amazonaws.com"

# Insert a memory
requests.post(f"{API}/insert", json={
    "agent_id": "my_agent",
    "key": "user_preference",
    "text": "User prefers dark mode and condensed layouts"
})

# Search memories
response = requests.post(f"{API}/search", json={
    "agent_id": "my_agent",
    "text": "UI preferences",
    "epsilon": 0.3,
    "threshold": 0.5,
    "top_k": 5
})
memories = response.json()["data"]
```

## Deployment Modes

### CLI Mode (Local Development)
- Single flat file storage (default: `tree.bin`)
- Direct Bedrock API calls for embeddings
- Perfect for prototyping and testing

### Lambda Mode (Production)
- **EFS**: Per-agent `.bin` files for warm storage
- **S3**: Automatic async backups to S3 for durability
- **Memcached**: ElastiCache cluster for embedding caching
- **API Gateway**: HTTP API with `/insert`, `/search`, `/insert-csv` endpoints
- **VPC**: Private subnets with NAT gateway for Bedrock access
- **Multi-agent isolation**: Each agent gets its own namespace

## How It Works

1. **Embedding**: Text → AWS Titan Embed v2 → 512-dimensional vector
2. **Indexing**: Each dimension maintains a sorted array of node indices
3. **Search**: Binary search per dimension to find nodes within epsilon-ball
4. **Filtering**: Return only nodes present in ALL 512 dimensions (intersection)
5. **Ranking**: Calculate Euclidean distance, filter by threshold, sort by similarity
6. **Top-K**: Return the K most similar results

## Search Parameters

Understanding how to tune search for different use cases:

### Epsilon (Bounding Box Size)
Controls the per-dimension search radius. Smaller = stricter matching.
- **0.15-0.2**: Precise matches (e.g., exact fact retrieval)
- **0.3**: General queries (default, balanced)
- **0.4-0.5**: Broad exploration

### Threshold (Distance Filter)
Filters results by actual Euclidean distance. Higher = stricter.
- **0.7+**: Safety-critical queries (allergies, medical info)
- **0.5-0.6**: Standard semantic search (default: 0.5)
- **0.4**: Loose matching, discovery mode

### Top-K (Result Limit)
Number of results to return, sorted by similarity.
- **1-3**: Precise answers
- **5**: General queries (default)
- **10**: Comprehensive searches

### Example Tuning
```bash
# Precise fact lookup
./bin/hippocampus search -text "peanut allergy" -e 0.2 -threshold 0.7 -top-k 3

# Broad exploration
./bin/hippocampus search -text "travel experiences" -e 0.4 -threshold 0.4 -top-k 10
```

## Demo Agents

Four Python demos in `demo/python/` show real-world agent patterns:

### 1. Basic Agent (`basic_agent.py`)
Simple memory insert/search using AWS Bedrock Converse API.
```bash
cd demo/python
python basic_agent.py
```

### 2. AgentCore Demo (`agentcore_agent.py`)
Advanced intelligent memory decomposition. The agent automatically:
- Analyzes input to identify multiple distinct facts
- Creates targeted, searchable memories with descriptive keys
- Adapts search parameters based on query type
- Demonstrates handling dense personal profiles (15+ facts from one input)

```bash
python agentcore_agent.py              # Run demo scenario
python agentcore_agent.py --interactive # Interactive mode
```

### 3. Safety Demo (`safety_demo.py`)
**Critical**: Shows how persistent memory prevents dangerous mistakes.

Scenario: Parent mentions child's shellfish allergy in one conversation. Days later, in a new conversation with zero context, they mention buying shrimp for dinner. The agent:
1. Proactively searches memory when food is mentioned
2. Uses high threshold (0.7+) for safety-critical queries
3. Retrieves the allergy information
4. Warns the parent before a life-threatening mistake

```bash
python safety_demo.py
```

This demonstrates why semantic memory matters for AI agents—not just convenience, but actual safety.

### 4. Customer Support Demo (`customer_support_demo.py`)
**Production-scale**: AI support agent with 200-vector knowledge base.

Demonstrates real-world enterprise use case:
- 200+ support articles (authentication, API docs, billing, troubleshooting, integrations)
- Semantic search across technical documentation
- Agent adapts search parameters based on question complexity
- Production-ready multi-agent architecture

```bash
# Populate knowledge base with 200 articles
python customer_support_demo.py --populate

# Run automated demo scenarios
python customer_support_demo.py --demo

# Interactive support chat
python customer_support_demo.py --interactive
```

Perfect for video demos—shows real business value at scale.

## Use Cases

- **Long-term agent memory**: Remember user preferences, facts, and context across sessions
- **Safety-critical retrieval**: Allergies, medical info, restrictions that must never be forgotten
- **Few-shot learning**: Retrieve relevant examples from history for in-context learning
- **Tool/API selection**: Find the right function based on semantic task description
- **Deduplication**: Check if similar content already exists before processing
- **Conversation context**: Load relevant past interactions on-demand
- **Output caching**: Store expensive LLM outputs, retrieve for similar queries

## AWS Setup

Requires AWS credentials with Bedrock access:

```bash
aws configure
# or
export AWS_ACCESS_KEY_ID=your_key
export AWS_SECRET_ACCESS_KEY=your_secret
```

IAM policy needed:
```json
{
    "Effect": "Allow",
    "Action": "bedrock:InvokeModel",
    "Resource": "arn:aws:bedrock:us-east-1::foundation-model/amazon.titan-embed-text-v2:0"
}
```

## Architecture

### Core Data Structures

```go
type Node struct {
    Key   [512]float32  // Embedding vector from AWS Titan
    Value string        // Actual text content
}

type Tree struct {
    Nodes []Node
    Index [512][]int32  // Per-dimension sorted indices for binary search
}
```

The Index is the key innovation: 512 arrays of node indices, each sorted by that dimension's value. This enables O(log n) binary search per dimension to find all nodes within the epsilon-ball.

### Module Organization

```
src/
├── types/              Tree/Node structs, Insert/Search algorithms
├── storage/            Binary file serialization (Save/Load)
├── embedding/          AWS Bedrock Titan embedding integration
├── client/             High-level API (combines storage + embedding)
├── cmd/cli/            CLI entry point
└── lambda/
    ├── main.go         Lambda entry point
    ├── handlers/       HTTP routing (insert, search, insert-csv)
    ├── storage/        Multi-agent storage manager with S3 sync
    └── cache/          Memcached integration

terraform/              AWS infrastructure (VPC, EFS, S3, Lambda, API Gateway)
demo/python/            Agent integration examples
```

### AWS Infrastructure (Lambda Mode)

Terraform deploys complete serverless infrastructure:
- **Lambda**: Function in VPC with 1GB RAM, 60s timeout
- **EFS**: Network file system mounted at /mnt/efs/agents/
- **S3**: Cold storage with versioning enabled
- **ElastiCache**: Memcached cluster for embedding cache
- **VPC**: Public/private subnets, NAT gateway for Bedrock API access
- **API Gateway**: HTTP API endpoints

Each agent gets isolated storage: `{agent_id}.bin` on EFS, backed up to S3.

## AWS Requirements

Configure AWS credentials with Bedrock access:
```bash
aws configure
```

Required IAM permissions:
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": "bedrock:InvokeModel",
      "Resource": "arn:aws:bedrock:us-east-1::foundation-model/amazon.titan-embed-text-v2:0"
    }
  ]
}
```

For Terraform deployment, update `terraform/terraform.tfvars`:
```hcl
aws_region       = "us-east-1"
s3_bucket_name   = "your-unique-bucket-name"
memcached_node_type = "cache.t3.micro"
```

Then deploy:
```bash
make deploy
```

## Performance & Limitations

### Performance
- **5k nodes**: ~20MB RAM, microsecond queries
- **10k nodes**: ~40MB RAM, sub-millisecond queries
- **Insert**: O(n log n) per dimension (512 sorted insertions)
- **Search**: O(log n × 512) binary searches + distance calculations
- **File size**: ~10MB per 5k nodes (2KB per node average)

### Limitations
- **Not for billions of vectors**: Designed for 5k-10k per agent. Use Pinecone/Weaviate for larger scale.
- **Single-writer**: No concurrent write support. Lambda writes are serialized per agent.
- **No automatic relevance**: Epsilon/threshold must be tuned for your domain.
- **Region locked**: Hardcoded to us-east-1 for Bedrock Titan availability.
- **Full rewrites**: CLI mode rewrites entire file on insert (Lambda mode uses EFS for better performance).

### When to Use Hippocampus
**Good for:**
- AI agent memory (user preferences, conversation history)
- Personal knowledge bases (notes, documents, memories)
- Few-shot example retrieval
- Tool/function selection
- Small-to-medium semantic search (under 10k vectors per agent)
- Deterministic, debuggable results required

**Not good for:**
- Billion-scale vector search
- Real-time collaborative editing
- High-frequency writes (>100/sec)
- Multi-modal embeddings (only text via Titan)

## CSV Format

Bulk import expects CSV with "key","text" format:
```csv
"preference_ui_theme","User prefers dark mode with high contrast"
"allergy_food_peanuts","Severe peanut allergy, carries EpiPen"
"travel_history_japan_2024","Visited Tokyo and Kyoto in March 2024"
"work_role_engineer","Software engineer focused on cloud infrastructure"
```

## Vision: Managed AWS Service for Agent Memory

Hippocampus is designed as a potential **managed AWS service** for AI agent memory:

### Why This Should Be a Service
- **Every agent needs memory**: Current solutions (Pinecone, Weaviate) are over-engineered or expensive for typical agent use cases
- **AWS-native architecture**: Built entirely on AWS primitives (Bedrock, Lambda, EFS, S3)
- **Serverless and scalable**: Pay-per-use model, automatic scaling per agent
- **Simple API**: Just insert and search—no complex configuration or tuning
- **Multi-tenancy ready**: Built-in agent isolation, per-agent billing possible

### Service Integration Points
- **AWS Bedrock**: Natural extension of Bedrock's agent capabilities
- **Amazon Q**: Could power Q's long-term memory across sessions
- **Amazon Connect**: Customer service agents with persistent context
- **AWS Amplify**: Client-side agent memory for mobile/web apps

### Deployment Model
```
┌─────────────────────────────────────────────────┐
│  AWS Hippocampus Service (Hypothetical)         │
├─────────────────────────────────────────────────┤
│  • Multi-tenant Lambda functions                │
│  • Shared EFS with per-tenant encryption        │
│  • S3 for cold storage and durability           │
│  • CloudWatch for monitoring and billing        │
│  • API Gateway for public endpoints             │
│  • Bedrock integration for embeddings           │
└─────────────────────────────────────────────────┘
```

Simple customer experience:
```python
import boto3

client = boto3.client('hippocampus')  # Hypothetical service

# Insert memory
client.insert(AgentId='my-agent', Key='user_pref', Text='User prefers dark mode')

# Search memory
results = client.search(AgentId='my-agent', Query='UI preferences')
```

## Built For AWS AI Agent Global Hackathon

This project demonstrates:
- **AWS Bedrock integration**: Titan Embed v2 for embeddings, Nova Lite for agent intelligence
- **Serverless architecture**: Lambda, EFS, S3, ElastiCache deployed via Terraform
- **Multi-agent systems**: Isolated per-agent storage with S3 backup
- **Production-ready**: VPC networking, IAM roles, API Gateway integration
- **Agent memory patterns**: Four demo agents showing real-world use cases (personal memory, safety-critical retrieval, customer support at scale)
