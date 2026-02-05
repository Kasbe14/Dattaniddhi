# Vector Database – Design Document (v1)

## 1. Problem Statement

> This project implements the **core engine of a vector database**. The system stores vector embeddings, enforces strong invariants, and supports similarity search over those vectors. It is designed to sit **between embedding generation and raw data retrieval systems**.
>
> The system **does not generate data** and **does not retrieve original payloads** (images, text, audio). Instead, it indexes vectors and returns **references (IDs) and similarity scores**, which downstream systems can use to fetch original data from external storage (e.g., S3, DBs).
>
---

## 2. High-Level Architecture

```
Raw Data
   ↓
Embedder (external models / APIs)
   ↓
Ingestion Layer
   ↓
Vector Database (this project)
   ↓
Retrieval System (external)
   ↓
Original Data
```

This repository implements the **Vector Database** and prepares the foundation for the **Ingestion Layer**.

---

## 3. Core Concepts

### 3.1 Vector

A `Vector` is an **immutable, fully-formed data structure** representing an embedding.

**Invariants:**

* A vector MUST have:

  * ID
  * Normalized values
  * Dimension
  * DataType
  * SimilarityMetric
* Vectors are normalized at construction time
* A vector can never exist without embedder context

**Implications:**

* No partial or "empty" vectors are allowed
* Similarity functions rely on normalization invariants

---

### 3.2 Index

An `Index` is a storage and search structure for vectors.

**Responsibilities:**

* Store vectors
* Enforce index-level invariants
* Perform similarity search

**Non-responsibilities:**

* ID generation
* Vector normalization
* Payload storage

---

### 3.3 Index Configuration

Each index has an internal, immutable configuration:

```go
IndexConfig {
    Dimension        int
    DataType         DataType
    SimilarityMetric SimilarityMetric
}
```

**IndexConfig invariants:**

* Locked on first successful insert
* All subsequent vectors must match exactly
* Violations result in errors

---

## 4. Search Semantics

### 4.1 Search Input

* Query vector
* Integer `k` (number of results)

### 4.2 Search Output

* Slice of `SearchResult`

  * Vector ID
  * Similarity score

### 4.3 Guarantees

* Results sorted by descending similarity score
* If `0 < k <= index size`: return `k` results
* If `k > index size`: return all results
* Empty index → empty result, no error

### 4.4 Non-Guarantees

* At least one result
* Data generation
* Payload retrieval

---

## 5. Similarity Metrics

* Vectors are normalized at construction
* Cosine similarity reduces to dot product

**Invariant:**

* All similarity functions assume normalized vectors

---

## 6. Concurrency Model

* Index implementations use `sync.RWMutex`
* Read-heavy workloads are supported
* Mutation operations are serialized

---

## 7. Current Scope (MVP)

Included:

* Vector construction and validation
* Linear index implementation
* Strict invariant enforcement
* Search correctness
* Unit tests for all behaviors

Excluded (for now):

* Ingestion orchestration
* Embedder execution
* Persistence
* Payload storage

---

## 8. Upcoming Phase

### Phase 5: Ingestion Layer

Planned responsibilities:

* Accept raw data
* Call embedder
* Generate vector IDs
* Construct vectors
* Route vectors to appropriate index
* Reject invalid flows early

---

## 9. Design Philosophy

* Invariants enforced at construction time
* No "half-valid" objects
* Fail fast, fail explicitly
* Simple core, extensible edges

---

## 10. Status

* Vector: ✅ Complete
* Index (Linear): ✅ Complete
* Search: ✅ Complete
* Tests: ✅ Complete
* Ingestion Layer: ⏳ Pending
* Persistence: ⏳ Pending
