# Hippocampus vs FAISS Benchmark

Comprehensive scaling benchmarks comparing Hippocampus to FAISS (the industry standard for vector search) at 100, 500, 1k, 5k, and 10k nodes.

## 🎯 **Results Summary**

**At 10,000 nodes, Hippocampus's pure search algorithm is 4,247x faster than FAISS brute force.**

| Nodes | Hippocampus | FAISS IndexFlatL2 | Speedup |
|-------|-------------|-------------------|---------|
| 1,000 | 0.49 μs | 283 μs | **583x faster** |
| 10,000 | 0.47 μs | 1,975 μs | **4,247x faster** |

**Full analysis, methodology, and mathematical proof:** [BENCHMARK_RESULTS.md](BENCHMARK_RESULTS.md)

---

## Quick Start

### Run Complete Scaling Benchmark (~40 minutes)

```bash
# Setup EC2 instance (one-time)
./setup-ec2.sh

# Run full scaling benchmark (100 → 10k nodes)
./deploy.sh

# Terminate EC2 when done
aws ec2 terminate-instances --region ap-southeast-2 --instance-ids <ID>
```

### Run Local Test (Fast)

```bash
# Build CLI
cd .. && make build-cli && cd faiss-comparison

# Copy binary
cp ../bin/hippocampus .

# Run benchmark
./benchmark_ec2.sh --hippocampus  # Just Hippocampus (no FAISS setup)
```

---

## What Gets Tested

Both systems use **identical AWS Bedrock Titan embeddings** (512 dimensions), isolating pure algorithm performance:

1. **Insert performance** at multiple scales
2. **Search performance** with timing breakdown:
   - Bedrock embedding time
   - File load / index build time
   - Pure search algorithm time
3. **Scaling characteristics** (linear vs logarithmic)

**Fair comparison:** FAISS IndexFlatL2 (deterministic brute force), NOT HNSW (approximate)

---

## Files

- `BENCHMARK_RESULTS.md` - Full results, analysis, and mathematical proof
- `benchmark_scaling.sh` - Complete scaling benchmark (100 → 10k)
- `benchmark_ec2.sh` - Single-scale benchmark
- `setup-ec2.sh` - Launch EC2 instance with dependencies
- `deploy.sh` - Run benchmark on EC2
- `monitor_ec2.sh` - Watch EC2 benchmark progress
- `cleanup.sh` - Terminate EC2 and cleanup resources

---

## Key Findings

**Algorithmic advantage confirmed:** O(512 log n) beats O(n) at agent scale
**Sub-microsecond search:** 0.2-0.5 μs pure algorithm time
**FAISS linear scaling validated:** Perfect O(n) growth as expected
**Bedrock dominates production:** 95%+ of query time is API latency

**The numbers are real. somehow. See [BENCHMARK_RESULTS.md](BENCHMARK_RESULTS.md) for proof.**
