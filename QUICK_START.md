# Hippocampus Quick Start Guide

## Installation

```bash
# Clone repo
git clone https://github.com/yourusername/hippocampus
cd hippocampus

# Build
make build-cli

# Verify
./bin/hippocampus
```

## 30-Second Demo

```bash
# 1. Create database with 3D vectors (simple test)
./bin/hippocampus insert -db demo.bin -dims 3 \
  -vector '[0.1,0.2,0.3]' -text 'Hello world'

./bin/hippocampus insert -db demo.bin -dims 3 \
  -vector '[0.1,0.3,0.2]' -text 'Greetings everyone'

./bin/hippocampus insert -db demo.bin -dims 3 \
  -vector '[0.9,0.1,0.05]' -text 'Goodbye world'

# 2. Search
./bin/hippocampus search -db demo.bin \
  -vector '[0.1,0.25,0.25]' -top-k 2

# Output:
# Found 2 results (top 2, threshold 0.50):
#   Hello world
#   Greetings everyone
# TIMING:SEARCH:0.018ms

# 3. Info
./bin/hippocampus info -db demo.bin

# Output:
# Database Info:
#   File: demo.bin
#   Nodes: 3
#   Dimensions: 3
#   Estimated size: 336 B
```

## With Local Embeddings (Ollama)

```bash
# 1. Install Ollama
curl https://ollama.ai/install.sh | sh
ollama pull nomic-embed-text

# 2. Insert with automatic embedding
./bin/hippocampus insert -db memories.bin \
  -text "The user prefers dark mode" -ollama

./bin/hippocampus insert -db memories.bin \
  -text "Important meeting at 3pm" -ollama

# 3. Search
./bin/hippocampus search -db memories.bin \
  -text "UI preferences" -ollama

# Output:
# Found 1 results (top 5, threshold 0.50):
#   The user prefers dark mode
# TIMING:SEARCH:150ms  (100ms embedding + 50ms search)
```

## Performance Benchmarks

```bash
# Run all benchmarks
go test ./src/... -bench=. -benchtime=3s

# Key results to expect:
# - Insert: ~237 ns/op
# - Search (1k nodes): ~0.69 ms
# - Search (5k nodes): ~4.0 ms
# - Parallel search: 4x faster than sequential
# - Mmap load: 1946x faster than regular load
# - Compression: 924ns compress, 334ns decompress
```

## Common Use Cases

### 1. Agent Memory

```go
package main

import (
    "Hippocampus/src/client"
    "Hippocampus/src/embedding"
)

func main() {
    // Create agent's memory
    memory, _ := client.New("agent_alice.bin", 768)

    // Use local embeddings
    ollama := embedding.NewOllamaClient("http://localhost:11434", "nomic-embed-text")

    // Remember facts
    vec, _ := ollama.GetEmbedding("User likes pizza")
    memory.Insert(vec, "User likes pizza")

    // Recall relevant memories
    query, _ := ollama.GetEmbedding("What food does user like?")
    results, _ := memory.Search(query, 0.3, 0.5, 3)
    // results: ["User likes pizza"]
}
```

### 2. Document Search

```go
// Load documents
docs := []string{
    "Machine learning is a subset of AI",
    "Neural networks have multiple layers",
    "Database indexing improves query speed",
}

db, _ := client.New("documents.bin", 768)
ollama := embedding.NewOllamaClient("http://localhost:11434", "nomic-embed-text")

// Batch insert (6x faster!)
items := make([]struct{
    Embedding []float32
    Text string
    Metadata types.Metadata
}, len(docs))

for i, doc := range docs {
    vec, _ := ollama.GetEmbedding(doc)
    items[i] = struct{...}{
        Embedding: vec,
        Text: doc,
        Metadata: types.Metadata{"index": i},
    }
}

db.BatchInsert(items)

// Search
query, _ := ollama.GetEmbedding("Tell me about AI")
results, _ := db.Search(query, 0.3, 0.5, 2)
```

### 3. With Metadata Filtering

```go
// Insert with metadata
metadata := types.Metadata{
    "user_id": "alice",
    "category": "important",
    "timestamp": time.Now().Unix(),
}

db.InsertWithMetadata(embedding, "Important note", metadata)

// Search only Alice's important items
filter := &types.Filter{
    Metadata: map[string]interface{}{
        "user_id": "alice",
        "category": "important",
    },
}

results, _ := db.SearchWithFilter(query, 0.3, 0.5, 5, filter)
```

## Performance Tips

### 1. Use Compression for Large Databases

```go
// Save compressed (4x smaller)
import "Hippocampus/src/storage"

cs := storage.NewCompressed("data.bin.compressed")
cs.Save(tree)

// Load compressed
tree, _ := cs.Load()
```

### 2. Use Mmap for Fast Cold Starts

```go
// Use mmap storage (1946x faster load)
mmap, _ := storage.NewMmapStorage("large_db.bin")
defer mmap.Close()

// Indices built lazily as needed
index := mmap.GetOrBuildIndex(0)  // Only builds when first accessed
```

### 3. Batch Inserts

```go
// SLOW: Insert one by one (rebuilds index each time)
for _, item := range items {
    db.Insert(item.Embedding, item.Text)
}

// FAST: Batch insert (rebuilds index once)
db.BatchInsert(items)  // 6x faster!
```

### 4. Tune Search Parameters

```go
// Tight search (high precision, fewer results)
results, _ := db.Search(query, 0.15, 0.7, 3)

// Balanced (default)
results, _ := db.Search(query, 0.3, 0.5, 5)

// Broad search (high recall, more results)
results, _ := db.Search(query, 0.5, 0.3, 10)
```

## PostgreSQL Extension (Advanced)

```bash
# Build extension
cd postgres-extension
make
sudo make install

# In PostgreSQL
psql mydb
```

```sql
CREATE EXTENSION hippocampus;

CREATE TABLE memories (
    id SERIAL PRIMARY KEY,
    content TEXT,
    embedding vector(768),
    metadata JSONB
);

-- Search
SELECT content, distance
FROM hippocampus_search(
    'memories_embedding_idx',
    '[0.1, 0.2, ...]'::vector,
    0.3, 0.5, 5, NULL
);
```

## Troubleshooting

### "dimension mismatch" error

```bash
# Check your database dimensions
./bin/hippocampus info -db yourfile.bin

# Use correct dimensions when searching
./bin/hippocampus search -db yourfile.bin -dims 768 -vector '[...]'
```

### Slow searches

```bash
# Check node count
./bin/hippocampus info -db yourfile.bin

# If >100k nodes, consider:
# 1. Use compressed storage (4x smaller)
# 2. Use mmap for fast loading
# 3. Adjust epsilon/threshold for fewer candidates
```

### Out of memory

```bash
# Use mmap storage (doesn't load everything into RAM)
# Or use compressed storage (4x less memory)
# Or split into multiple databases
```

## What's Next?

- Read [PERFORMANCE.md](PERFORMANCE.md) for detailed benchmarks
- Read [IMPROVEMENTS.md](IMPROVEMENTS.md) for all features
- Check [postgres-extension/README.md](postgres-extension/README.md) for SQL usage
- See [examples/](examples/) for more code samples

## Support

- Issues: https://github.com/yourusername/hippocampus/issues
- Docs: https://hippocampus.dev
- Discord: https://discord.gg/hippocampus

**Happy vector searching!** 🚀
