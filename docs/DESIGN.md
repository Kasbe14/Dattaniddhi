
## Vector Database – High-Level Design Document (v1)   

## 1. Problem Statement   

> This project is a vector database system designed to store, index, and search high-dimensional vector embeddings efficiently.
> 
> The system accepts raw data (such as text, images, audio, or video) through an embedding layer, converts that data into vector embeddings, and supports similarity search over stored vectors using well-defined similarity metrics.
>
>The system returns identifiers and similarity scores for matching vectors. Retrieval of the original raw data (e.g., images, text, audio) is performed by the caller using these identifiers.
>
>The system focuses exclusively on vector-based retrieval, not data generation, and is designed with strong invariants, clear ownership boundaries, and extensibility toward advanced indexing and optimization techniques.

## 2. Non-Goals   

The system explicitly does not:   
- Generate new data   
- Perform generative AI tasks   
- Modify raw user data    
- Interpret semantic meaning of embeddings   

The following are explicitly out of scope for the initial design:   
- Text generation or LLM inference   
- Training embedding models   
- Approximate nearest neighbor optimizations (HNSW, IVF, PQ)   
- Persistence to disk   
- Distributed indexing   
- Query filtering or metadata search   

## 3. Core Data Flow   

> Raw Data (text, image, audio, video, etc.)   
>    ↓   
> Embedder   
>    ↓   
> Vector (immutable)   
>    ↓   
> Index   
>    ↓   
> SearchResult (vector ID + similarity score)   

## 4. Component Responsibilities, Core Concepts    

### 4.1 Raw Data   
Raw data represents the original user-provided input to the system.   
*Examples*:   
- Text (strings, documents)   
- Images (files, byte streams)   
- Other modalities in the future (audio, video, etc.)   
Raw data is never stored or indexed directly. It must first pass through an *Embedder*.   
   
### 4.2 Embedder   
An Embedder is a logical component responsible for converting raw data into vectors.   
Embedder is stateless and does not generate IDs.
It only produces embeddings and embedding metadata.

**Key properties**:   
- Each Embedder supports exactly one input modality (e.g., text, image).   
- Each Embedder produces vectors of a fixed, invariant dimension.   
- Each Embedder wraps an external embedding model (local or remote).   
- The system does not implement or train embedding models itself.   
- Vectors must never exist without an originating Embedder.   
- Consumers of vectors trust the Embedder’s guarantees.   
>Vectors are meaningless without the Embedder that produced them.   

### 4.3 Ingestion   
Ingestion Layer owns IDs generation and mapping to the storage reference

### 4.4 Vector   
A Vector is the numerical representation of raw data produced by an Embedder.   
**Properties**:   
- Fixed-length numeric array (dimension defined by the Embedder)   
- Immutable after creation   
- Always associated with:   
     - An originating Embedder   
     - A unique vector ID   
Vectors are **not user-supplied** and **cannot be created independently**.   

### 4.5 Index   
An **Index** is a storage and retrieval structure for vectors.   
**Design constraints**:   
- An Index is bound to exactly one Embedder identity   
- Index compatibility is determined by the Embedder, not just vector dimension   
- Vectors from different modalities or embedders must never be mixed in the same index   
 **Responsibilities**:   
- Store vectors   
- Perform similarity search (kNN,HNSW etc.)   
- Return vector IDs ranked by similarity score   
 
### 4.6 Similarity & Distance    
Similarity computation is defined implicitly by the Embedder and Index combination.   
*Examples*:   
- Cosine similarity   
- Dot product   
- Euclidean distance   
These metrics are implementation details, not part of the core API yet.   
They may be formalized later as the system evolves.   

### 4.7. Search Result   
A **Search Result** represents the outcome of querying an Index.
For the current scope:
- Contains:
   - Vector ID
   - Similarity score
- Does not expose index references or internal metadata
Future versions may extend this structure.

### 4.8. Retrieval of Original Data (Goal)
The long-term goal of the system is to allow retrieval of original raw data corresponding to vectors.
*Conceptually*:
- Vector ID → raw data reference
- Similar vectors → similar raw data (e.g., images, text)
This behavior **is out of scope for v1**, but explicitly part of the system’s direction.

## 5. Ownerships & Invariants   

### 1. Index Variant   
The index guarantees:   
All stored vectors have the same dimension   
The dimension is locked after the first insertion   
Search results are sorted by similarity score (descending)   
Search returns at most k results   
The index does not guarantee:   
At least one result   
Non-empty results for empty indexes   

### 2. Search Semantics   
Inputs   
Query vector (non-nil)   
Integer k, where k > 0   
Outputs   
A slice of SearchResult   
Each result contains:   
Vector ID   

### 3. Similarity Score Behavior   
If k > index size, return all vectors   
If index is empty, return empty result set   
If dimensions mismatch, return an error   
   
### 4. Search vs Retrieval   
Search identifies which vectors are relevant.   
Retrieval (outside this system) uses vector IDs to fetch raw data.   
   
### 5.  Error Handling Philosophy   
Invalid inputs fail fast with explicit errors   
Valid inputs on empty data return empty results, not errors   
Errors indicate contract violations, not absence of data   
   
### 6. ID Ownership   
Vector IDs are owned by the Embedder.   
Users MUST NOT:   
Manually create vectors   
Manually assign vector IDs   
Insert arbitrary vectors into an index   

## 6. Current Scope (MVP)   
### Implemented:   
- Vector abstraction   
- Linear index   
- Add / Get / Delete   
- Dimension locking   
- Linear similarity search   
### Planned:   
- Embedder interface   
- Optimized search (top-k heap,HNSW,IVF)   
- Metadata filtering   
- Persistent storage   

## Status   
This document defines the current architectural contract.   
Code must conform to this document.