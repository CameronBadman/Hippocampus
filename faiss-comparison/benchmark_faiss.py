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

scale = 100
print(f"Generating {scale} embeddings...")
embeddings = []
start = time.time()
for i in range(scale):
    text = sample_texts[i % len(sample_texts)] + str(i)
    emb = get_embedding(text)
    embeddings.append(emb)
    if (i + 1) % 100 == 0:
        print(f"  Progress: {i+1}/{scale}", end='\r')
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
