# Hippocampus

**A vector database built for AI agents.** Not retrofitted from document search—designed from the ground up for how agents actually manage memory.

---

## Benchmark Summary: 4,200x Faster Than FAISS (Exact Search)

Hippocampus’s custom **O(512 log n)** binary search algorithm achieves **4,200× faster** pure search performance than FAISS’s **O(n)** brute-force IndexFlatL2 at 10,000 vectors, while maintaining deterministic, exact retrieval.

Full methodology, mathematical validation, and reproducible tests are available in [`faiss-comparison/`](faiss-comparison).

---

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
    "epsilon": 0.2,
    "threshold": 0.6,
    "top_k": 3
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

* No database servers to manage
* No connection pools or networking complexity
* Just files: load, search, done
* Perfect for Lambda's ephemeral execution model

### Built for Agents, Not Documents

**Traditional vector databases optimize for:**

* Billion-scale document retrieval
* Approximate nearest neighbors (HNSW, IVF)
* Distributed clusters

**Agents need:**

* Per-agent isolated memory (multi-tenancy)
* Deterministic, debuggable results
* Fast at 5k-10k scale
* Natural language control over precision
* File-based simplicity

## How It Works

### Core Architecture

**Custom 512-dimensional indexing**

* 512 sorted arrays (one per dimension)
* O(log n) binary search per dimension
* Guaranteed retrieval, no approximation

**Binary serialization**

* Custom format: 2KB per node
* Each agent gets isolated `.bin` file
* Rebuild index on load (~50ms)

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

Parent: "My daughter Emma has a shellfish allergy"
→ Agent stores it

Weeks later: "I'm buying shrimp for Emma's dinner"
→ Agent searches memory
→ Finds allergy
→ Warns parent, prevents harm

```bash
cd demo/python
python safety_demo.py
```

### 2. Intelligent Decomposition (`agentcore_agent.py`)

User shares complex paragraph (20+ facts)
→ External agent decides to curate
→ Calls internal curation agent (Lambda + Nova)
→ Internal agent decomposes into discrete memories
→ External agent queries later
→ Retrieves precisely what's needed

```bash
python agentcore_agent.py
python agentcore_agent.py --interactive
```

### 3. Agent-to-Agent Curation (`test_agent_curate.py`)

```bash
python test_agent_curate.py
```

## Architecture: Pure Lambda, File-Based, Serverless

### Deployment

**Infrastructure (Terraform):**

* Lambda: stateless, infinitely scalable
* EFS: per-agent `.bin` files for hot access
* S3: async backups for durability
* NAT Gateway: Lambda → Bedrock API calls
* API Gateway: `/insert`, `/search`, `/agent-curate` endpoints

```bash
make deploy
```

### File-Based Storage

```
EFS/S3 Structure:
/agents/
  ├── agent_abc123.bin
  ├── agent_def456.bin
  └── agent_ghi789.bin
```

## API Reference

### POST /insert

```json
{
  "agent_id": "user123",
  "key": "preference_theme",
  "text": "User prefers dark mode"
}
```

### POST /search

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

## Performance

### Benchmarks (5k nodes per agent)

* File size: ~10MB
* RAM: ~20MB in Lambda
* Cold start: ~200ms
* Search: <50ms (warm)
* Insert: ~100ms

### Scalability

* Per-agent: 5k-10k nodes
* Total agents: unlimited
* Lambda concurrency: 1000 concurrent agents
* Cost: ~$0.00012 per memory operation

## Limitations & Trade-offs

**Not for:**

* Billion-scale vector search
* Real-time collaborative editing
* Highly concurrent writes

**Ideal for:**

* AI agent memory
* Personal knowledge bases
* Few-shot retrieval
* Tool/API selection

## Vision: AWS-Native Agent Memory Service

Hippocampus demonstrates what a managed AWS service for agent memory could look like.

```python
import boto3
client = boto3.client('hippocampus')

client.insert(AgentId='my-agent', Key='pref', Text='Dark mode')
results = client.search(AgentId='my-agent', Query='UI preferences')
```

## Built For AWS AI Agent Global Hackathon

* Custom vector database built from scratch
* File-based architecture optimized for agent workloads
* Two interaction patterns: agent-controlled + database-curated
* Agent-to-agent orchestration (external → internal)

---

## Note on the UI

Hippocampus **does include a demo UI**, but it is designed purely to **showcase functionality**. The UI is not intended for production usage.

The **main product** is the underlying vector database and API, which is used by the demo agents to manage memory efficiently and safely. All agent workflows, memory insertions, searches, and curation happen programmatically through the API — the UI simply visualizes these operations for demonstration purposes.


**The SQLite of AI agents. Simple, reliable, production-ready.**
