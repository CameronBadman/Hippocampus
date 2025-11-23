-- complain if script is sourced in psql, rather than via CREATE EXTENSION
\echo Use "CREATE EXTENSION hippocampus" to load this file. \quit

-- Vector type (array of floats)
CREATE TYPE vector;

CREATE OR REPLACE FUNCTION vector_in(cstring)
RETURNS vector
AS 'MODULE_PATHNAME'
LANGUAGE C IMMUTABLE STRICT;

CREATE OR REPLACE FUNCTION vector_out(vector)
RETURNS cstring
AS 'MODULE_PATHNAME'
LANGUAGE C IMMUTABLE STRICT;

CREATE TYPE vector (
    INPUT = vector_in,
    OUTPUT = vector_out,
    STORAGE = EXTENDED
);

-- Hippocampus index creation
CREATE OR REPLACE FUNCTION hippocampus_index_create(
    table_name text,
    column_name text,
    dimensions integer DEFAULT 512
)
RETURNS void
AS 'MODULE_PATHNAME', 'hippocampus_index_create'
LANGUAGE C STRICT;

-- Insert with metadata
CREATE OR REPLACE FUNCTION hippocampus_insert(
    index_name text,
    embedding vector,
    value text,
    metadata jsonb DEFAULT NULL
)
RETURNS void
AS 'MODULE_PATHNAME', 'hippocampus_insert'
LANGUAGE C STRICT;

-- Search function
CREATE OR REPLACE FUNCTION hippocampus_search(
    index_name text,
    query_embedding vector,
    epsilon real DEFAULT 0.3,
    threshold real DEFAULT 0.5,
    top_k integer DEFAULT 5,
    metadata_filter jsonb DEFAULT NULL
)
RETURNS TABLE(value text, distance real, metadata jsonb)
AS 'MODULE_PATHNAME', 'hippocampus_search'
LANGUAGE C STRICT;

-- Distance operator for vectors
CREATE OR REPLACE FUNCTION vector_distance(vector, vector)
RETURNS real
AS 'MODULE_PATHNAME'
LANGUAGE C IMMUTABLE STRICT PARALLEL SAFE;

CREATE OPERATOR <-> (
    LEFTARG = vector,
    RIGHTARG = vector,
    FUNCTION = vector_distance,
    COMMUTATOR = '<->'
);

-- Batch insert
CREATE OR REPLACE FUNCTION hippocampus_batch_insert(
    index_name text,
    embeddings vector[],
    values text[],
    metadata_array jsonb[]
)
RETURNS integer
AS 'MODULE_PATHNAME', 'hippocampus_batch_insert'
LANGUAGE C STRICT;

COMMENT ON TYPE vector IS 'Vector data type for embeddings';
COMMENT ON FUNCTION hippocampus_index_create IS 'Create a Hippocampus vector index';
COMMENT ON FUNCTION hippocampus_search IS 'Search for similar vectors using Hippocampus';
COMMENT ON OPERATOR <-> (vector, vector) IS 'Euclidean distance between vectors';
