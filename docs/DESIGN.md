# Dattanidhi – System Design Document (v2.0)

> **Disclaimer:** Dattanidhi is an educational project. It is a work-in-progress designed as a hands-on exploration of low-level systems engineering, concurrency, and database durability. While built with "production-grade" discipline, it is meant for learning rather than enterprise deployment.

## 1. Problem Statement

This project implements the **core engine of a standalone vector database**. The system stores vector embeddings, enforces strong schema invariants, ensures data durability, and supports similarity search. It is designed to sit between embedding generation and raw data retrieval systems.

The system **does not generate data** and **does not retrieve original payloads** (images, text, audio). Instead, it indexes vectors and returns references (IDs) and similarity scores, which downstream systems use to fetch original data from external storage.

---

## 2. High-Level Architecture

Dattanidhi enforces a strict separation between **Policy** (user-facing identity and semantics) and **Mechanism** (mathematical search and byte-level storage).

    External Embedder / API
              ↓
    [ Policy: Collection Layer ] ← (Validates schema, translates UUIDs to internal IDs)
              ↓
    [ Mechanism: Index Layer ]   ← (Mathematical vector search via RWMutex)
              ↓
    [ Persistence: WAL Layer ]   ← (Binary Write-Ahead Log for crash recovery)

---

## 3. Core Concepts

### 3.1 Vector (`internal/vector`)
A `Vector` is an **immutable, fully-formed data structure** representing an embedding.
* **Invariants:** Must have a positive dimension, no `NaN`/`Inf` values, and is automatically normalized to a unit vector at construction.
* **Immutability:** The internal slice cannot be modified after creation, preventing external memory corruption.

### 3.2 Index (`internal/index`)
The `Index` is the raw search mechanism. It knows nothing about external UUIDs or AI models.
* **Schema-Driven:** Governed by an immutable `IndexConfig` (Dimension, Metric, Type).
* **Internal Addressing:** Operates strictly on zero-indexed integer IDs (`VecId`).

### 3.3 Collection (`internal/collection`)
The `Collection` acts as the Database Identity Layer.
* **Responsibilities:** Generates standard `UUIDv7/String` IDs, translates them to internal integer IDs, and manages the semantic `CollectionConfig` (DataType, ModelName).
* **Rollbacks:** Ensures atomic operations. If an index insertion fails, ID mappings are instantly reverted to prevent database corruption.

---

## 4. Concurrency Model

* **Index Level:** Implementations (like `LinearIndex`) utilize `sync.RWMutex`. This allows high-throughput concurrent read-heavy search operations while safely serializing `Add` and `Delete` mutations.
* **Collection Level:** Employs a separate `sync.RWMutex` to protect the ID translation maps (`extToInt`, `intToExt`) and the payload store from race conditions during concurrent API requests.

---

## 5. Persistence Architecture: Write-Ahead Log (WIP)

To guarantee durability, Dattanidhi uses a custom binary Write-Ahead Log (`internal/wal`).

* **Segment Files:** The log is split into `.waldrky` segment files. Each begins with a 16-byte header containing magic bytes (`SANGITA`) and a Segment ID.
* **Binary Encoding:** Operations are serialized into a strict binary format. A record includes a 32-byte header (Version, LSN, OpType) followed by the payload (Vector bits, UUIDs).
* **Integrity:** Every complete record wrapper is sealed with an IEEE CRC32 checksum to detect disk corruption.
* **Sync Policies:** Tunable durability via `SyncEverySec`, `SyncAlways`, or `SyncOS`.

---

## 6. Current Scope & Execution Status

**Included (MVP):**
* ✅ Vector construction and mathematical validation.
* ✅ Linear index implementation with concurrency controls.
* ✅ Collection layer for identity and schema enforcement.

**Active Development:**
* 🔄 **Phase 5 (Persistence):** Building the binary WAL, segment cycling, and crash recovery logic.

**Upcoming Phases:**
* ⏳ **Phase 3:** Pluggable persistent payload storage (replacing current in-memory map).
* ⏳ **Phase 4:** API-facing Ingestion pipeline.
* ⏳ **Phase 7:** Advanced Index structures (HNSW, IVF, PQ).