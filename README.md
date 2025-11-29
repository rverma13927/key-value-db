# Go Key-Value Database 🚀

A high-performance, thread-safe, persistent Key-Value store written in Go. This project mimics the architecture of real-world databases like **NutsDB** and **Bitcask**.

## 🌟 Features Implemented

### Phase 1: Thread Safety 🔒
*   **Goal**: Allow multiple readers and one writer concurrently.
*   **Implementation**: Used `sync.RWMutex`.
    *   `RLock()` for `Get` (Multiple readers allowed).
    *   `Lock()` for `Set`/`Delete` (Exclusive access).

### Phase 2: Persistence (Append-Only Log) 💾
*   **Goal**: Data should survive a restart.
*   **Implementation**:
    *   Every `Set` or `Delete` operation is appended to `db.log`.
    *   Format: `SET,bucket,key,value,expiry`.
    *   On startup (`Load()`), the file is replayed to rebuild the in-memory map.

### Phase 3: Time-To-Live (TTL) ⏳
*   **Goal**: Keys should expire automatically.
*   **Implementation**:
    *   Stored `ExpireAt` timestamp in the `Entry` struct.
    *   `Get()` checks `time.Now().After(entry.ExpireAt)`. If true, it returns "Key has expired".

### Phase 4: HTTP API 🌐
*   **Goal**: Access the DB over the web.
*   **Implementation**:
    *   `GET /get?bucket=...&key=...`
    *   `GET /set?bucket=...&key=...&value=...`
    *   Runs on port `8080`.

### Phase 5: Buckets (Namespaces) 🗂️
*   **Goal**: Logical separation of keys (like folders).
*   **Implementation**:
    *   Changed data structure to `map[string]map[string]Entry`.
    *   Outer map key is the **Bucket Name**.

### Phase 6: Log Compaction (Merge) 🧹
*   **Goal**: Prevent `db.log` from growing infinitely.
*   **Implementation**:
    *   `Merge()` creates a temporary file.
    *   Writes only the *active* (non-deleted, non-expired) keys.
    *   Swaps the old log file with the new one.

### Phase 7: Transactions (ACID) ⚛️
*   **Goal**: Atomic updates (All-or-Nothing).
*   **Implementation**:
    *   **Write-Ahead Log (WAL)** approach.
    *   Writes `TX_BEGIN` -> All Changes -> `TX_COMMIT`.
    *   On `Load()`, partial transactions (missing `TX_COMMIT`) are ignored.
    *   `Update(fn)` method ensures rollback on error.

---

## 🛠️ Usage

### 1. Run the HTTP Server
```bash
go run server/server.go
```

### 2. API Examples
*   **Set a Value**:
    ```bash
    curl "http://localhost:8080/set?bucket=users&key=alice&value=100"
    ```
*   **Get a Value**:
    ```bash
    curl "http://localhost:8080/get?bucket=users&key=alice"
    ```

### 3. Run Tests
```bash
go test -v kv/tx_test.go kv/kv.go
```

---

## 📂 Project Structure
*   `kv/kv.go`: Core Database Logic (Structs, Methods, Persistence).
*   `server/server.go`: HTTP Server implementation.
*   `main.go`: CLI playground for testing.
*   `db.log`: The persistent storage file.

## 🔜 Coming Soon
*   **Phase 8**: Data Structures (List, Set).
*   **Phase 9**: Sharding.
