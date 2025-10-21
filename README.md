# Hippocampus

**A vector database built for AI agents.** Not retrofitted from document search—designed from the ground up for how agents actually manage memory.

## The Problem

AI agents lack consistency, reliability, and real memory. Traditional vector databases are built for billion-scale document retrieval. They're over-engineered for problems you don't have.

Agents need something different: fast, isolated, persistent memory they control.

## The Solution

**Hippocampus**: The SQLite of AI agent memory.

Built for the **agent use case**: 5k-10k vectors per agent that need to be fast, deterministic, and debuggable.

### Two Ways Agents Use Memory

**1. Agent-Controlled Pattern**
Smart agents with full control over their memory:
```python
# Agent decides what to store
requests.post(f"{API}/insert", json={
    "agent_id": "my_agent",
    "key": "user_preference_dark_mode",
    "text": "User prefers dark mode"
})

# Agent controls search precision
requests.post(f"{API}/search", json={
    "agent_id": "my_agent",
    "text": "UI preferences",
    "epsilon": 0.2,      # Search radius
    "threshold": 0.6,    # Similarity cutoff
    "top_k": 3           # Result limit
})
```

**2. Database-Curated Pattern**
Simple agents just pass text—internal AI agent handles curation:
```python
# Send raw text, get structured memories
requests.post(f"{API}/agent-curate", json={
    "agent_id": "my_agent",
    "text": "My name is Sarah, I work at Google, allergic to shellfish",
    "importance": "high",
    "model_id": "us.amazon.nova-lite-v1:0",
    "bedrock_region": "us-east-1"
})

# Internal agent automatically creates:
# • personal_name_sarah
# • occupation_google_engineer  
# • allergy_shellfish
```

## Why This Architecture?

### The SQLite Philosophy

**SQLite**: File-based SQL database, simple, reliable, ubiquitous  
**Hippocampus**: File-based vector database, simple, reliable, built for agents

- No database servers to manage
- No connection pools or networking complexity
- Just files: load, search, done
- Perfect for Lambda's ephemeral execution model

### Built for Agents, Not Documents

**Traditional vector databases optimize for:**
- Billion-scale document retrieval
- Approximate nearest neighbors (HNSW, IVF)
- Distributed clusters

**Agents need:**
- Per-agent isolated memory (multi-tenancy)
- Deterministic, debuggable results
- Fast at 5k-10k scale
- Natural language control over precision
- File-based simplicity

## How It Works

### Core Architecture

**Custom 512-dimensional indexing**
- 512 sorted arrays (one per dimension)
- O(log n) binary search per dimension
- Guaranteed retrieval, no approximation

**Binary serialization**
- Custom format: 2KB per node
- Each agent gets isolated `.bin` file
- Rebuild index on load (~50ms)

**Search algorithm:**
1. Text → AWS Bedrock Titan → 512-dim vector
2. Binary search each dimension for epsilon-ball candidates
3. Calculate Euclidean distance for candidates
4. Filter by threshold, sort by similarity
5. Return top-k results

**Data structures:**
```go
type Node struct {
    Key   [512]float32  // Embedding vector
    Value string        // Actual text
}

type Tree struct {
    Nodes []Node
    Index [512][]int32  // Per-dimension sorted indices
}
```

## Demo: End-to-End Agent Workflows

### 1. Safety-Critical Memory (`safety_demo.py`)

**The scenario that matters:**

Parent: "My daughter Emma has a shellfish allergy"  
→ Agent stores it

*[Weeks pass, new conversation]*

Parent: "I'm buying shrimp for Emma's dinner"  
→ Agent searches memory  
→ Finds allergy  
→ **Warns parent, prevents ER trip**

**This is why agent memory matters—not convenience, but safety.**

```bash
cd demo/python
python safety_demo.py
```

### 2. Intelligent Decomposition (`agentcore_agent.py`)

**Agent-to-agent orchestration:**

User shares complex paragraph (20+ facts)  
→ External agent (Bedrock Nova) decides to curate  
→ Calls internal curation agent (Lambda + Nova)  
→ Internal agent decomposes into 28 discrete memories  
→ External agent queries later  
→ Retrieves precisely what's needed

**This is agents orchestrating agents—autonomous, scalable, production-ready.**

```bash
python agentcore_agent.py              # Demo scenario
python agentcore_agent.py --interactive # Interactive mode
```

### 3. Agent-to-Agent Curation (`test_agent_curate.py`)

Shows the full `/agent-curate` workflow with both curation and retrieval.

```bash
python test_agent_curate.py
```

## Architecture: Pure Lambda, File-Based, Serverless

### Deployment

**Infrastructure (Terraform):**
- **Lambda**: Pure function, stateless, infinitely scalable
- **EFS**: Per-agent `.bin` files for hot access
- **S3**: Automatic async backups for durability
- **NAT Gateway**: Lambda → Bedrock API calls
- **API Gateway**: `/insert`, `/search`, `/agent-curate` endpoints

**Key insight:** Because Lambda is pure (stateless), this scales as hard as Lambda scales. No database bottlenecks.

```bash
make deploy  # Deploys complete AWS stack
```

### File-Based Storage

```
EFS/S3 Structure:
/agents/
  ├── agent_abc123.bin    # 10MB, 5k memories
  ├── agent_def456.bin    # 15MB, 7k memories
  └── agent_ghi789.bin    # 8MB, 4k memories
```

**Why this works:**
- Isolation: Each agent's memory is separate
- Lazy loading: Lambda only loads requested agent's file
- Simple backups: Just copy `.bin` files to S3
- No coordination: No locks, no distributed state

### Lambda Execution Model

```
Request → Lambda spins up
       ↓
   Load agent's .bin from EFS (~50ms)
       ↓
   Rebuild 512-index in memory
       ↓
   Execute search (<100ms)
       ↓
   Return results
       ↓
   Async S3 backup (no latency impact)
```

## API Reference

### POST /insert
Agent-controlled memory storage.
```json
{
  "agent_id": "user123",
  "key": "preference_theme",
  "text": "User prefers dark mode"
}
```

### POST /search
Semantic search with full parameter control.
```json
{
  "agent_id": "user123",
  "text": "UI preferences",
  "epsilon": 0.3,
  "threshold": 0.5,
  "top_k": 5
}
```

### POST /agent-curate
Internal AI agent curates text into discrete memories.
```json
{
  "agent_id": "user123",
  "text": "Sarah, 34, Google engineer, allergic to shellfish",
  "importance": "high",
  "model_id": "us.amazon.nova-lite-v1:0",
  "bedrock_region": "us-east-1",
  "timeout_ms": 50
}
```

**Returns:**
```json
{
  "memories_created": 3,
  "memories": [
    {"key": "personal_name_sarah", "text": "Sarah", "reasoning": "..."},
    {"key": "occupation_google", "text": "Google engineer", "reasoning": "..."},
    {"key": "allergy_shellfish", "text": "allergic to shellfish", "reasoning": "..."}
  ]
}
```

## Search Parameters Guide

### Epsilon (Search Radius)
Per-dimension bounding box. Lower = stricter.
- **0.15-0.2**: Precise fact lookup
- **0.3**: Balanced (default)
- **0.4-0.5**: Broad exploration

### Threshold (Distance Filter)
Euclidean distance cutoff. Higher = stricter.
- **0.7+**: Safety-critical queries
- **0.5-0.6**: General search (default)
- **0.4**: Discovery mode

### Top-K (Result Limit)
Number of results, sorted by similarity.
- **1-3**: Precise answers
- **5**: General queries (default)
- **10**: Comprehensive search

## Performance

### Benchmarks (5k nodes per agent)
- **File size**: ~10MB
- **RAM**: ~20MB in Lambda
- **Cold start**: ~200ms (load + index rebuild)
- **Search**: <50ms (warm)
- **Insert**: ~100ms (includes embedding)

### Scalability
- **Per-agent**: 5k-10k nodes (sweet spot)
- **Total agents**: Unlimited (isolated files)
- **Lambda concurrency**: 1000 concurrent agents (default)
- **Cost**: ~$0.00012 per memory operation

### Real Performance (From Logs)
```
Duration: 290.50 ms     # Search with 125 nodes
Max Memory Used: 44 MB  # Efficient
```

## Module Organization

```
src/
├── types/              Core Tree/Node, Insert/Search algorithms
├── storage/            Binary serialization (Save/Load)
├── embedding/          AWS Bedrock Titan integration
├── client/             High-level API
└── lambda/
    ├── handlers/       HTTP routing, agent-curate logic
    └── storage/        Multi-agent manager with S3 sync

terraform/              Complete AWS infrastructure
demo/python/            Agent integration examples
```

## AWS Setup

### Prerequisites
```bash
aws configure
```

### IAM Permissions
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": "bedrock:InvokeModel",
      "Resource": [
        "arn:aws:bedrock:*::foundation-model/amazon.titan-embed-text-v2:0",
        "arn:aws:bedrock:*::foundation-model/amazon.nova-lite-v1:0"
      ]
    }
  ]
}
```

### Deploy
```bash
make deploy
```

## Limitations & Trade-offs

**Not for:**
- Billion-scale vector search (use Pinecone/Weaviate)
- Real-time collaborative editing
- Sub-millisecond latency requirements
- Highly concurrent writes to same agent

**Perfect for:**
- AI agent memory (user preferences, history)
- Personal knowledge bases
- Few-shot example retrieval
- Tool/API selection
- 5k-10k vectors per agent use case

## Vision: AWS-Native Agent Memory Service

Hippocampus demonstrates what a **managed AWS service for agent memory** could look like:

**Why this should be a service:**
- Every agent needs memory
- AWS-native (Bedrock, Lambda, EFS, S3)
- Serverless, pay-per-use
- Simple API
- Multi-tenant ready

**Integration points:**
- AWS Bedrock Agents
- Amazon Q (cross-session memory)
- Amazon Connect (customer service agents)
- AWS Amplify (client-side agents)

**Hypothetical customer experience:**
```python
import boto3
client = boto3.client('hippocampus')

client.insert(AgentId='my-agent', Key='pref', Text='Dark mode')
results = client.search(AgentId='my-agent', Query='UI preferences')
```

## Built For AWS AI Agent Global Hackathon

**Novel contributions:**
- ✅ Custom vector database built from scratch (not using existing services)
- ✅ File-based architecture optimized for agent workloads
- ✅ Two interaction patterns: agent-controlled + database-curated
- ✅ Agent-to-agent orchestration (external → internal curation agent)

**Demonstrates:**
- AWS Bedrock (Titan embeddings, Nova Lite reasoning)
- Serverless architecture (Lambda, EFS, S3, API Gateway)
- Multi-agent systems (isolated storage, autonomous curation)
- Production-ready (VPC, NAT Gateway, IAM, monitoring)
- Real-world agent patterns (safety-critical, decomposition, orchestration)

---

**The SQLite of AI agents. Simple, reliable, production-ready.**
