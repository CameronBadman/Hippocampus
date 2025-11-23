-- Hippocampus PostgreSQL Extension Example
-- Semantic search for a simple document database

-- 1. Create extension
CREATE EXTENSION IF NOT EXISTS hippocampus;

-- 2. Create table for documents
CREATE TABLE IF NOT EXISTS documents (
    id SERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    embedding vector(512),  -- Hippocampus vector type
    metadata JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);

-- 3. Create Hippocampus index
SELECT hippocampus_index_create('documents', 'embedding', 512);

-- 4. Insert sample documents
-- Note: In real usage, embeddings would come from an embedding model (e.g., OpenAI, Cohere, Ollama)
-- These are dummy vectors for demonstration

INSERT INTO documents (title, content, embedding, metadata) VALUES
(
    'Introduction to Machine Learning',
    'Machine learning is a subset of artificial intelligence...',
    (SELECT ('[' || string_agg(random()::text, ',') || ']')::vector
     FROM generate_series(1, 512)),
    '{"category": "AI", "difficulty": "beginner", "author": "Alice"}'::jsonb
),
(
    'Advanced Neural Networks',
    'Deep learning models with multiple hidden layers...',
    (SELECT ('[' || string_agg(random()::text, ',') || ']')::vector
     FROM generate_series(1, 512)),
    '{"category": "AI", "difficulty": "advanced", "author": "Bob"}'::jsonb
),
(
    'Database Design Principles',
    'Normalization, indexing, and query optimization...',
    (SELECT ('[' || string_agg(random()::text, ',') || ']')::vector
     FROM generate_series(1, 512)),
    '{"category": "databases", "difficulty": "intermediate", "author": "Charlie"}'::jsonb
),
(
    'Python for Data Science',
    'NumPy, Pandas, and Scikit-learn basics...',
    (SELECT ('[' || string_agg(random()::text, ',') || ']')::vector
     FROM generate_series(1, 512)),
    '{"category": "programming", "difficulty": "beginner", "author": "Diana"}'::jsonb
),
(
    'Distributed Systems',
    'Consistency, availability, and partition tolerance...',
    (SELECT ('[' || string_agg(random()::text, ',') || ']')::vector
     FROM generate_series(1, 512)),
    '{"category": "systems", "difficulty": "advanced", "author": "Eve"}'::jsonb
);

-- 5. Simple distance query (traditional approach)
-- Find documents similar to a query vector
WITH query AS (
    SELECT ('[' || string_agg(random()::text, ',') || ']')::vector AS embedding
    FROM generate_series(1, 512)
)
SELECT
    d.title,
    d.embedding <-> q.embedding AS distance,
    d.metadata->>'category' AS category
FROM documents d, query q
ORDER BY d.embedding <-> q.embedding
LIMIT 5;

-- 6. Hippocampus semantic search (fast!)
-- Search for similar documents with metadata filtering
WITH query AS (
    SELECT ('[' || string_agg(random()::text, ',') || ']')::vector AS embedding
    FROM generate_series(1, 512)
)
SELECT
    title,
    distance,
    metadata
FROM query q,
     LATERAL hippocampus_search(
         'documents_embedding_idx',
         q.embedding,
         0.3,    -- epsilon
         0.5,    -- threshold
         5,      -- top_k
         NULL    -- no metadata filter
     ) AS results(value TEXT, distance REAL, metadata JSONB)
ORDER BY distance;

-- 7. Search with metadata filter
-- Only search within "AI" category, beginner difficulty
WITH query AS (
    SELECT ('[' || string_agg(random()::text, ',') || ']')::vector AS embedding
    FROM generate_series(1, 512)
)
SELECT
    title,
    distance,
    metadata->>'difficulty' AS difficulty
FROM query q,
     LATERAL hippocampus_search(
         'documents_embedding_idx',
         q.embedding,
         0.3,
         0.5,
         3,
         '{"category": "AI", "difficulty": "beginner"}'::jsonb  -- Filter
     ) AS results(value TEXT, distance REAL, metadata JSONB)
ORDER BY distance;

-- 8. Batch insert example
-- Insert 1000 documents efficiently
SELECT hippocampus_batch_insert(
    'documents_embedding_idx',
    (SELECT array_agg(('[' || string_agg(random()::text, ',') || ']')::vector)
     FROM generate_series(1, 1000),
          LATERAL (SELECT string_agg(random()::text, ',')
                   FROM generate_series(1, 512)) AS vec_gen(vec)),
    (SELECT array_agg('Document ' || gs::text)
     FROM generate_series(1, 1000) AS gs),
    (SELECT array_agg(('{"batch": true, "index": ' || gs::text || '}')::jsonb)
     FROM generate_series(1, 1000) AS gs)
);

-- 9. Time-range search
-- Find documents created in the last hour
WITH query AS (
    SELECT ('[' || string_agg(random()::text, ',') || ']')::vector AS embedding
    FROM generate_series(1, 512)
)
SELECT
    title,
    created_at,
    distance
FROM documents d, query q
WHERE d.created_at > NOW() - INTERVAL '1 hour'
ORDER BY d.embedding <-> q.embedding
LIMIT 10;

-- 10. Aggregate statistics
SELECT
    metadata->>'category' AS category,
    COUNT(*) AS document_count,
    AVG(array_length(ARRAY(SELECT json_array_elements_text((embedding)::json)), 1)) AS avg_dimensions
FROM documents
GROUP BY metadata->>'category'
ORDER BY document_count DESC;

-- 11. Performance comparison
-- Compare Hippocampus vs. sequential scan
EXPLAIN ANALYZE
WITH query AS (
    SELECT ('[' || string_agg(random()::text, ',') || ']')::vector AS embedding
    FROM generate_series(1, 512)
)
SELECT title, d.embedding <-> q.embedding AS distance
FROM documents d, query q
ORDER BY distance
LIMIT 5;

-- vs

EXPLAIN ANALYZE
WITH query AS (
    SELECT ('[' || string_agg(random()::text, ',') || ']')::vector AS embedding
    FROM generate_series(1, 512)
)
SELECT title, distance
FROM query q,
     LATERAL hippocampus_search(
         'documents_embedding_idx',
         q.embedding,
         0.3, 0.5, 5, NULL
     ) AS results(value TEXT, distance REAL, metadata JSONB);

-- 12. Clean up (optional)
-- DROP TABLE documents;
-- DROP EXTENSION hippocampus;

\echo 'Example completed! Check the results above.'
