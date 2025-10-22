#!/bin/bash

set -e

echo "=================================================="
echo "Hippocampus vs FAISS Scaling Benchmark"
echo "Testing at: 100, 500, 1000, 5000, 10000 nodes"
echo "Region: ap-southeast-2"
echo "=================================================="
echo

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
RED='\033[0;31m'
NC='\033[0m'

# Test scales
SCALES=(100 500 1000 5000 10000)
SEARCH_QUERIES=20

echo -e "${CYAN}This benchmark will test both systems at multiple scales${NC}"
echo -e "${CYAN}to demonstrate logarithmic vs linear scaling.${NC}"
echo
echo -e "${YELLOW}Estimated time: ~30-40 minutes for all scales${NC}"
echo

# Clean up old files
rm -f benchmark_hippo_*.bin

# Define paths
CURRENT_DIR=$(pwd)
PROJECT_ROOT=$(cd .. && pwd)
BINARY_PATH="$PROJECT_ROOT/bin/hippocampus"
LOCAL_BINARY="./hippocampus"

echo -e "${BLUE}Checking Hippocampus CLI...${NC}"
if [ ! -f "$LOCAL_BINARY" ]; then
    echo -e "${YELLOW}⚠ hippocampus binary not found in current directory${NC}"
    
    # Check if it exists in the project bin directory
    if [ -f "$BINARY_PATH" ]; then
        echo -e "${CYAN}Found hippocampus binary in project bin directory. Copying...${NC}"
        cp "$BINARY_PATH" "$LOCAL_BINARY"
        chmod +x "$LOCAL_BINARY"
    else
        echo -e "${YELLOW}Building hippocampus binary...${NC}"
        
        # Navigate to project root
        cd "$PROJECT_ROOT"
        
        # Create bin directory if it doesn't exist
        mkdir -p bin
        
        # Build the binary
        echo -e "${CYAN}Running go build...${NC}"
        CGO_ENABLED=0 go build -o bin/hippocampus src/cmd/cli/main.go
        
        # Check if build was successful
        if [ -f "bin/hippocampus" ]; then
            echo -e "${GREEN}✓ Successfully built Hippocampus binary${NC}"
            # Copy back to the faiss-comparison directory
            cp "bin/hippocampus" "$CURRENT_DIR/hippocampus"
            chmod +x "$CURRENT_DIR/hippocampus"
            # Return to the original directory
            cd "$CURRENT_DIR"
        else
            echo -e "${RED}✗ Failed to build Hippocampus binary${NC}"
            cd "$CURRENT_DIR"
            exit 1
        fi
    fi
fi

# Verify the binary is now in place
if [ ! -f "$LOCAL_BINARY" ]; then
    echo -e "${RED}✗ Could not find or build hippocampus binary${NC}"
    exit 1
fi
echo -e "${GREEN}✓ CLI ready${NC}"
echo

# Check Python environment
echo -e "${BLUE}Checking Python environment...${NC}"
if python -c "import numpy, faiss, boto3" 2>/dev/null; then
    echo -e "${GREEN}✓ Python environment ready (using Nix packages)${NC}"
    PYTHON_CMD="python"
else
    echo -e "${YELLOW}⚠ Nix Python packages not found, falling back to venv${NC}"
    # Setup Python environment for FAISS
    echo -e "${BLUE}Setting up Python environment for FAISS...${NC}"
    if [ ! -d "venv" ]; then
        echo -e "${CYAN}  Creating virtual environment...${NC}"
        python3 -m venv venv
    fi

    echo -e "${CYAN}  Installing dependencies (numpy, faiss-cpu, boto3)...${NC}"
    source venv/bin/activate
    pip install --quiet numpy faiss-cpu boto3 > /dev/null 2>&1
    echo -e "${GREEN}✓ Python environment ready${NC}"
    PYTHON_CMD="./venv/bin/python"
fi
echo

# Results arrays
declare -A hippo_insert_times
declare -A hippo_search_times
declare -A hippo_pure_search_times
declare -A faiss_insert_times
declare -A faiss_search_times
declare -A faiss_pure_search_times

sample_texts=(
    "User prefers dark mode interface"
    "User likes Python programming language"
    "User allergic to shellfish"
    "User enjoys marathon running"
    "User drinks espresso every morning"
    "User lives in Seattle Washington"
    "User works at Amazon Web Services"
    "User has golden retriever named Max"
    "User graduated from MIT computer science"
    "User enjoys landscape photography"
    "User loves Italian food pasta"
    "User wants to visit Japan Tokyo"
    "User reads science fiction novels"
    "User listens to jazz music"
    "User goes to gym three times weekly"
    "User speaks fluent Spanish and French"
    "User plays guitar and piano"
    "User is vegetarian no meat"
    "User allergic to peanuts and latex"
    "User climbs mountains hiking outdoors"
)

search_queries=(
    "UI preferences"
    "programming languages"
    "food allergies"
    "exercise activities"
    "beverages drinks"
    "location city"
    "work job"
    "pets animals"
    "education degree"
    "hobbies interests"
    "favorite foods"
    "travel destinations"
    "books reading"
    "music preferences"
    "fitness routine"
    "languages spoken"
    "musical instruments"
    "dietary restrictions"
    "allergies medical"
    "outdoor activities"
)

# Run benchmarks for each scale
for SCALE in "${SCALES[@]}"; do
    echo "=================================================="
    echo -e "${CYAN}Testing with $SCALE nodes${NC}"
    echo "=================================================="
    echo

    HIPPO_DB="benchmark_hippo_${SCALE}.bin"

    # Hippocampus Benchmark
    echo -e "${BLUE}[1/2] Hippocampus at $SCALE nodes${NC}"

    # Insert benchmark
    echo -e "${CYAN}  Inserting $SCALE nodes...${NC}"
    insert_start=$(date +%s.%N)

    for i in $(seq 1 $SCALE); do
        idx=$((i % ${#sample_texts[@]}))
        text="${sample_texts[$idx]} variant $i"
        ./hippocampus insert -binary "$HIPPO_DB" \
            -region ap-southeast-2 \
            -key "memory_$i" \
            -text "$text" > /dev/null 2>&1

        if [ $((i % 100)) -eq 0 ]; then
            echo -ne "    Progress: $i/$SCALE\r"
        fi
    done

    insert_end=$(date +%s.%N)
    insert_time=$(echo "$insert_end - $insert_start" | bc)
    avg_insert=$(echo "scale=1; $insert_time * 1000 / $SCALE" | bc)
    hippo_insert_times[$SCALE]=$avg_insert
    echo -e "  ${GREEN}✓ Insert: ${avg_insert}ms avg (total: ${insert_time}s)${NC}"

    # Search benchmark with timing breakdown
    echo -e "${CYAN}  Searching $SEARCH_QUERIES queries...${NC}"
    search_times=()

    # Capture timing from one sample search
    sample_output=$(./hippocampus search -binary "$HIPPO_DB" \
        -region ap-southeast-2 \
        -text "test query" \
        -epsilon 0.3 \
        -threshold 0.5 \
        -top-k 5 2>&1)

    # Extract timing data
    if echo "$sample_output" | grep -q "TIMING:"; then
        timing_line=$(echo "$sample_output" | grep "TIMING:")
        hippo_embed=$(echo "$timing_line" | sed -n 's/.*EMBED:\([0-9.]*\).*/\1/p')
        hippo_load=$(echo "$timing_line" | sed -n 's/.*LOAD:\([0-9.]*\).*/\1/p')
        hippo_pure_search=$(echo "$timing_line" | sed -n 's/.*SEARCH:\([0-9.]*\).*/\1/p')
        echo -e "  ${CYAN}Timing breakdown: Embed=${hippo_embed}ms, Load=${hippo_load}ms, Search=${hippo_pure_search}ms${NC}"
    fi

    # Run full search benchmark
    for i in $(seq 1 $SEARCH_QUERIES); do
        query="${search_queries[$((i-1))]}"
        search_start=$(date +%s.%N)
        ./hippocampus search -binary "$HIPPO_DB" \
            -region ap-southeast-2 \
            -text "$query" \
            -epsilon 0.3 \
            -threshold 0.5 \
            -top-k 5 > /dev/null 2>&1
        search_end=$(date +%s.%N)
        search_time=$(echo "$search_end - $search_start" | bc)
        search_times+=($search_time)
    done

    total=0
    for t in "${search_times[@]}"; do
        total=$(echo "$total + $t" | bc)
    done
    avg_search=$(echo "scale=1; $total * 1000 / $SEARCH_QUERIES" | bc)
    hippo_search_times[$SCALE]=$avg_search
    hippo_pure_search_times[$SCALE]=$hippo_pure_search
    echo -e "  ${GREEN}✓ Search: ${avg_search}ms avg${NC}"
    echo

    # FAISS Benchmark
    echo -e "${BLUE}[2/2] FAISS at $SCALE nodes${NC}"

    cat > benchmark_faiss_temp.py << PYEOF
import time
import numpy as np
import json
import sys
import faiss
import boto3

bedrock = boto3.client('bedrock-runtime', region_name='ap-southeast-2')

def get_embedding(text):
    response = bedrock.invoke_model(
        modelId='amazon.titan-embed-text-v2:0',
        body=json.dumps({
            "inputText": text,
            "dimensions": 512,
            "normalize": True
        })
    )
    result = json.loads(response['body'].read())
    return np.array(result['embedding'], dtype=np.float32)

sample_texts = [
    "User prefers dark mode interface variant ",
    "User likes Python programming language variant ",
    "User allergic to shellfish variant ",
    "User enjoys marathon running variant ",
    "User drinks espresso every morning variant ",
    "User lives in Seattle Washington variant ",
    "User works at Amazon Web Services variant ",
    "User has golden retriever named Max variant ",
    "User graduated from MIT computer science variant ",
    "User enjoys landscape photography variant ",
]

scale = $SCALE
print(f"Generating {scale} embeddings...")
embeddings = []
start = time.time()
for i in range(scale):
    text = sample_texts[i % len(sample_texts)] + str(i)
    emb = get_embedding(text)
    embeddings.append(emb)
    if (i + 1) % 100 == 0:
        print(f"  Progress: {i+1}/{scale}", end='\\r')
embeddings = np.array(embeddings)
embed_time = time.time() - start
avg_embed = (embed_time / scale) * 1000
print(f"✓ Embeddings: {avg_embed:.1f}ms avg (total: {embed_time:.1f}s)")

print("Building FAISS index...")
dimension = 512
index = faiss.IndexFlatL2(dimension)
index.add(embeddings)
print("✓ Index built")

search_queries = [
    "UI preferences",
    "programming languages",
    "food allergies",
    "exercise activities",
    "beverages drinks",
    "location city",
    "work job",
    "pets animals",
    "education degree",
    "hobbies interests",
    "favorite foods",
    "travel destinations",
    "books reading",
    "music preferences",
    "fitness routine",
    "languages spoken",
    "musical instruments",
    "dietary restrictions",
    "allergies medical",
    "outdoor activities"
]

print(f"Running {len(search_queries)} searches...")
search_times = []
embedding_times = []
pure_search_times = []

for query in search_queries:
    # Time the embedding generation
    embed_start = time.time()
    query_emb = get_embedding(query)
    embed_end = time.time()
    embedding_times.append(embed_end - embed_start)

    # Time the pure FAISS search
    query_emb = np.array([query_emb])
    search_start = time.time()
    distances, indices = index.search(query_emb, 5)
    search_end = time.time()
    pure_search_times.append(search_end - search_start)

    # Total time
    search_times.append(embed_end - embed_start + search_end - search_start)

avg_search = np.mean(search_times) * 1000
avg_embed_time = np.mean(embedding_times) * 1000
avg_pure_search = np.mean(pure_search_times) * 1000

print(f"✓ Search: {avg_search:.1f}ms avg")
print(f"  - Embedding: {avg_embed_time:.1f}ms avg")
print(f"  - Pure FAISS search: {avg_pure_search:.3f}ms avg")

# Output for parsing
print(f"FAISS_INSERT:{avg_embed}")
print(f"FAISS_SEARCH:{avg_search}")
print(f"FAISS_PURE_SEARCH:{avg_pure_search}")
PYEOF

    # Run Python script and parse output
    echo -e "${CYAN}Running FAISS benchmark with: $PYTHON_CMD${NC}"
    if command -v python &> /dev/null; then
        output=$($PYTHON_CMD benchmark_faiss_temp.py 2>&1)
        echo "$output" | grep -v "FAISS_"

        faiss_insert=$(echo "$output" | grep "FAISS_INSERT:" | cut -d: -f2)
        faiss_search=$(echo "$output" | grep "FAISS_SEARCH:" | cut -d: -f2)
        faiss_pure_search=$(echo "$output" | grep "FAISS_PURE_SEARCH:" | cut -d: -f2)

        faiss_insert_times[$SCALE]=$faiss_insert
        faiss_search_times[$SCALE]=$faiss_search
        faiss_pure_search_times[$SCALE]=$faiss_pure_search
    else
        echo -e "${RED}Python not found, skipping FAISS benchmark${NC}"
    fi

    rm -f benchmark_faiss_temp.py
    echo
done

# Generate comparison table
echo
echo "=================================================="
echo -e "${GREEN}SCALING COMPARISON RESULTS${NC}"
echo "=================================================="
echo
echo "INSERT PERFORMANCE (ms per operation):"
echo "
