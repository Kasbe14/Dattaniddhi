# Dattanidhi (दत्तनिधि)

[![Go Report Card](https://goreportcard.com/badge/github/Kasbe14/dattanidhi)](https://goreportcard.com/report/github/Kasbe14/dattanidhi)
![License](https://img.shields.io/badge/license-MIT-blue.svg)

> **दत्त (Datta):** Given / Data
> **निधि (Nidhi):** Treasure / Repository

**Dattanidhi** is an educational, production-grade Vector Database engine written in Go. 

The project serves as a deep dive into low-level systems engineering. It bypasses off-the-shelf libraries to explore the fundamental mechanics of database construction: strict architectural boundaries, concurrency safety, and disk-level durability.

## 🛠 Core Engineering Focus

* **Clean Architecture:** A strict boundary between *Policy* (The `Collection` layer managing external UUIDs and semantics) and *Mechanism* (The `Index` layer handling pure mathematical search).
* **Concurrency-Safe Indexing:** High-performance linear indexing utilizing `sync.RWMutex` to support concurrent read-heavy workloads safely.
* **Custom Binary WAL (WIP):** A from-scratch Write-Ahead Log implementation featuring IEEE CRC32 checksums, zero-padded segment files, and raw binary encoding for crash recovery.
* **Invariant Enforcement:** Strict validation at object construction. Vectors are normalized at birth, and schemas are locked to prevent "half-valid" states.

## 🚀 Project Status: Active Development

Dattanidhi is currently in the middle of a major architectural pivot towards **Phase 5: Persistence**. 

* **Complete:** Vector mathematics, Linear Indexing, Collection Identity mapping.
* **In Progress:** The Write-Ahead Log (WAL) and disk-flush synchronization policies.
* **Roadmap:** HNSW Indexing, IVF, Snapshotting, and gRPC/HTTP API layers.

*(Note: This is an educational project meant for learning systems architecture, not for enterprise production environments.)*