# Hippocampus

Semantic search without the bullshit. A flat-file vector store for AI agents that just works.

## What is this?

A simple, stable vector database for small-to-medium semantic search. No HNSW graphs, no quantization, no clustering. Just sorted arrays and binary search. Perfect for AI agent memory, caching, and context retrieval.

## Why?

Most vector databases are over-engineered for billion-scale problems. Hippocampus is designed for the 99% use case: **5k-10k vectors that need to be fast, deterministic, and debuggable.**

- **Simple**: 20MB of code, sorted arrays, binary search
- **Stable**: Same query = same results, always
- **Fast**: O(log n) searches across 512 dimensions
- **Portable**: Single flat file, no dependencies (except AWS SDK)
- **Debuggable**: No black box indexes, just data structures you learned in CS101

## Installation

```bash
git clone https://github.com/yourusername/Hippocampus
cd Hippocampus/src
go build -o hippocampus main.go
```

## Usage

### Insert text with semantic embedding
```bash
./hippocampus insert -key "doc1" -text "hello world"
./hippocampus insert -key "greeting" -text "good morning everyone"
```

### Search semantically
```bash
./hippocampus search -text "hi there" -epsilon 0.2
```

### Top-K search
```bash
./hippocampus search -text "hi there" -epsilon 0.2 -top 5
```

### Custom database file
```bash
./hippocampus insert -file custom.bin -key "doc2" -text "foo bar"
./hippocampus search -file custom.bin -text "foo" -epsilon 0.15
```

### CSV Bulk Insert
```bash
./hippocampus insert-csv -csv csvFile.csv
```

#### CSV Example Format
```csv
"doc1","Meeting about Q4 planning"
"doc2","Discussed project timeline"
"doc3","Coffee with team"
"doc4","Reviewed customer feedback on new product"
"doc5","Brainstormed ideas for marketing campaign"
"doc6","Follow-up email to vendor about pricing"
"doc7","Weekly sprint retrospective discussion"
"doc8","Finalized presentation for stakeholders"
"doc9","One-on-one meeting with team member"
"doc10","Analyzed quarterly sales data"
```

## How it works

1. **Embedding**: Text → AWS Titan → 512-dimensional vector
2. **Indexing**: Each dimension maintains a sorted list of node indices
3. **Search**: Binary search per dimension to find nodes within epsilon distance
4. **Results**: Return nodes that match across ALL 512 dimensions (strict bounding box)

## Configuration

- **Epsilon**: Controls search tolerance (0.1 = strict, 0.5 = loose)
- **Top-K**: Limit results to the K most relevant nodes (-top flag)
- **Region**: Hardcoded to `us-east-1` (Bedrock availability)
- **Model**: `amazon.titan-embed-text-v2:0` (512 dimensions)

## Performance

- **5k nodes**: ~20MB RAM, microsecond queries
- **Insert**: O(n log n) per dimension (512 insertions)
- **Search**: O(log n * 512) binary searches
- **File size**: ~10MB per 5k nodes

## Use Cases for AI Agents

- **Semantic memory**: Remember past conversations by meaning
- **Tool selection**: Find relevant APIs/functions for tasks
- **Few-shot retrieval**: Pull similar examples from history
- **Deduplication**: Check if you've seen something similar
- **Context loading**: Retrieve relevant knowledge on-demand
- **Output caching**: Store expensive LLM calls, retrieve similar queries

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

```
types/       - Core data structures (Node, Tree)
storage/     - Flat file serialization (Save/Load)
embedding/   - AWS Titan integration
main.go      - CLI interface
```

**Tree structure:**
```go
type Node struct {
    Key   [512]float32  // Embedding vector
    Value string        // Your data
}

type Tree struct {
    Nodes []Node
    Index [512][]int32  // Sorted indices per dimension
}
```

## Limitations

- **Not for billions of vectors**: Use Pinecone/Weaviate for that
- **Writes rewrite entire file**: Optimized for read-heavy workloads
- **No concurrent writes**: Single-writer model
- **Epsilon must be tuned**: No automatic relevance ranking
- **Squared Euclidean distance**: Only relative ranking matters, absolute value is not normalized