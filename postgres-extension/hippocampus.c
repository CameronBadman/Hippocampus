/*
 * Hippocampus PostgreSQL Extension
 * Fast vector similarity search for AI agents
 */

#include "postgres.h"
#include "fmgr.h"
#include "utils/array.h"
#include "utils/builtins.h"
#include "utils/jsonb.h"
#include "catalog/pg_type.h"
#include "executor/spi.h"
#include <math.h>

PG_MODULE_MAGIC;

/* Vector type structure */
typedef struct Vector {
    int32 dim;          /* Number of dimensions */
    float4 *data;       /* Vector data */
} Vector;

/* Function declarations */
PG_FUNCTION_INFO_V1(vector_in);
PG_FUNCTION_INFO_V1(vector_out);
PG_FUNCTION_INFO_V1(vector_distance);
PG_FUNCTION_INFO_V1(hippocampus_index_create);
PG_FUNCTION_INFO_V1(hippocampus_insert);
PG_FUNCTION_INFO_V1(hippocampus_search);
PG_FUNCTION_INFO_V1(hippocampus_batch_insert);

/*
 * vector_in - Parse text representation of vector
 * Input: '[0.1, 0.2, 0.3]'
 */
Datum
vector_in(PG_FUNCTION_ARGS)
{
    char *str = PG_GETARG_CSTRING(0);
    Vector *result;
    int dim = 0;
    char *token, *ptr;
    float4 *data;
    int i = 0;

    /* Count dimensions by counting commas + 1 */
    for (ptr = str; *ptr; ptr++) {
        if (*ptr == ',') dim++;
    }
    dim++; /* One more element than commas */

    /* Allocate vector */
    result = (Vector *) palloc(VARHDRSZ + sizeof(int32) + dim * sizeof(float4));
    SET_VARSIZE(result, VARHDRSZ + sizeof(int32) + dim * sizeof(float4));
    result->dim = dim;
    data = result->data;

    /* Parse values */
    ptr = str;
    while (*ptr == '[' || *ptr == ' ') ptr++; /* Skip opening bracket and spaces */

    token = strtok(ptr, ",]");
    while (token != NULL && i < dim) {
        data[i++] = (float4) atof(token);
        token = strtok(NULL, ",]");
    }

    PG_RETURN_POINTER(result);
}

/*
 * vector_out - Convert vector to text representation
 * Output: '[0.1, 0.2, 0.3]'
 */
Datum
vector_out(PG_FUNCTION_ARGS)
{
    Vector *vec = (Vector *) PG_GETARG_POINTER(0);
    StringInfoData buf;
    int i;

    initStringInfo(&buf);
    appendStringInfoChar(&buf, '[');

    for (i = 0; i < vec->dim; i++) {
        if (i > 0)
            appendStringInfoString(&buf, ", ");
        appendStringInfo(&buf, "%f", vec->data[i]);
    }

    appendStringInfoChar(&buf, ']');

    PG_RETURN_CSTRING(buf.data);
}

/*
 * vector_distance - Calculate Euclidean distance between two vectors
 */
Datum
vector_distance(PG_FUNCTION_ARGS)
{
    Vector *a = (Vector *) PG_GETARG_POINTER(0);
    Vector *b = (Vector *) PG_GETARG_POINTER(1);
    float4 distance = 0.0;
    int i;

    if (a->dim != b->dim)
        ereport(ERROR,
                (errcode(ERRCODE_DATA_EXCEPTION),
                 errmsg("vector dimensions must match")));

    for (i = 0; i < a->dim; i++) {
        float4 diff = a->data[i] - b->data[i];
        distance += diff * diff;
    }

    PG_RETURN_FLOAT4(sqrt(distance));
}

/*
 * hippocampus_index_create - Create a Hippocampus index
 * Calls the Go library to create the database file
 */
Datum
hippocampus_index_create(PG_FUNCTION_ARGS)
{
    text *table_name = PG_GETARG_TEXT_PP(0);
    text *column_name = PG_GETARG_TEXT_PP(1);
    int32 dimensions = PG_GETARG_INT32(2);

    char *table_str = text_to_cstring(table_name);
    char *column_str = text_to_cstring(column_name);

    /* TODO: Call Go library via CGO to create index */
    elog(NOTICE, "Creating Hippocampus index on %s.%s with %d dimensions",
         table_str, column_str, dimensions);

    PG_RETURN_VOID();
}

/*
 * hippocampus_insert - Insert a vector into the index
 */
Datum
hippocampus_insert(PG_FUNCTION_ARGS)
{
    text *index_name = PG_GETARG_TEXT_PP(0);
    Vector *embedding = (Vector *) PG_GETARG_POINTER(1);
    text *value = PG_GETARG_TEXT_PP(2);
    Jsonb *metadata = NULL;

    if (!PG_ARGISNULL(3))
        metadata = PG_GETARG_JSONB_P(3);

    char *index_str = text_to_cstring(index_name);
    char *value_str = text_to_cstring(value);

    /* TODO: Call Go library to insert */
    elog(NOTICE, "Inserting into index %s: %s (dims: %d)",
         index_str, value_str, embedding->dim);

    PG_RETURN_VOID();
}

/*
 * hippocampus_search - Search for similar vectors
 */
Datum
hippocampus_search(PG_FUNCTION_ARGS)
{
    text *index_name = PG_GETARG_TEXT_PP(0);
    Vector *query = (Vector *) PG_GETARG_POINTER(1);
    float4 epsilon = PG_GETARG_FLOAT4(2);
    float4 threshold = PG_GETARG_FLOAT4(3);
    int32 top_k = PG_GETARG_INT32(4);
    Jsonb *metadata_filter = NULL;

    if (!PG_ARGISNULL(5))
        metadata_filter = PG_GETARG_JSONB_P(5);

    /* TODO: Call Go library to search and return results */
    elog(NOTICE, "Searching index with epsilon=%f, threshold=%f, top_k=%d",
         epsilon, threshold, top_k);

    /* This would normally return a set of tuples */
    PG_RETURN_NULL();
}

/*
 * hippocampus_batch_insert - Batch insert vectors
 */
Datum
hippocampus_batch_insert(PG_FUNCTION_ARGS)
{
    text *index_name = PG_GETARG_TEXT_PP(0);
    ArrayType *embeddings = PG_GETARG_ARRAYTYPE_P(1);
    ArrayType *values = PG_GETARG_ARRAYTYPE_P(2);
    ArrayType *metadata_array = PG_GETARG_ARRAYTYPE_P(3);

    int nelems = ArrayGetNItems(ARR_NDIM(embeddings), ARR_DIMS(embeddings));

    elog(NOTICE, "Batch inserting %d vectors", nelems);

    /* TODO: Call Go library for batch insert */

    PG_RETURN_INT32(nelems);
}
