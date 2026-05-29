# Architecture Summary — Persistent Vector Database Lifecycle Design

## Current Layered Architecture

```
API Layer (future)
↓
Database Layer (future)
↓
Collection Layer (current main focus)
↓
Engine/Storage Layers
    ├── WAL
    ├── Index
    ├── Vector
    └── Recovery
```

**Responsibilities**

1. API Layer (future)
   Purpose:

* User interaction boundary
* HTTP/gRPC/CLI handling
* Request validation
* Authentication/routing later

Does NOT:

* manage WAL directly
* replay recovery
* manage vectors internally

*Flow*:

```
user request
→ API
→ database/collection methods
```

2. Database Layer (future)
   Purpose:

* Owns multiple collections
* Collection registry/catalog
* Persistent metadata management

Example:

```
database:
 ├── users
 ├── products
 └── logs
```

Collection identity:

```
(database_name, collection_name)
```

Future responsibilities:

* create/delete collections
* collection metadata persistence
* collection lookup
* WAL directory organization

3. Collection Layer (current focus)
   Purpose:

* Main logical vector store
* Owns:

  * WAL instance
  * Index instance
  * ID mappings
  * Payload mappings
  * Recovery replay

Collection owns:

```
1 collection → 1 WAL
```

Collection runtime state:

```
extToInt
intToExt
payload
idCounter
index
```

Important:
Collection structs are NOT persisted.
Only WAL/data on disk persists.

On restart:

```
NewCollection()
↓
create empty structs
↓
LoadState()
↓
replay WAL
↓
rebuild memory state
```

4. WAL Layer
   Purpose:

* Durable source of truth
* Sequential append-only log
* Crash recovery foundation

Stores:

```
INSERT
DELETE
UPDATE (future)
```

Recovery behavior:

```
scan segments
→ validate headers
→ verify CRC
→ decode records
→ replay
```

5. Recovery Architecture
   Split responsibilities:

WAL:

```
physical recovery
```

Meaning:

* scan files
* validate bytes
* detect corruption
* decode records

Collection:

```
logical replay
```

Meaning:

* rebuild index
* rebuild mappings
* rebuild idCounter
* apply insert/delete semantics

Recovery ownership flow:

```
NewCollection()
↓
LoadState()
↓
wal.Recover()
↓
decoded records
↓
replay into collection state
```

Important Persistence Insight

Go structs are ONLY in-memory representations.

When process exits:

```
ALL structs disappear
```

Persistent truth is:

```
WAL files on disk
```

Recovery reconstructs structs from persisted bytes.

Delete Idempotency Decision

Delete contract:

```
Delete on missing ID must return nil
during recovery semantics
```

Reason:

* compensation WAL records
* replay safety
* crash consistency

Index implementations must honor this.

Compensation WAL Design

Ghost-write issue solved via:

```
WAL INSERT
↓
index failure
↓
WAL DELETE compensation
```

Replay naturally resolves final valid state.

Collection/WAL Directory Design

Future structure:

```
./data/<database>/<collection>/wal/
```

Example:

```
./data/main/users/wal/
./data/main/products/wal/
```

This ensures:

* collection isolation
* unique persistence identity
* multi-database support later

Current Development Phase

Currently in:

```
Lifecycle + Reliability Phase
```

Main goals:

* startup correctness
* recovery correctness
* crash consistency
* persistent durability
* replay integrity

Not yet:

* distributed systems
* networking
* APIs
* approximate indexes
* snapshots/checkpoints
