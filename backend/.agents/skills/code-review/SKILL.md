---
name: code-review
description: "Use this skill whenever the user wants to find bugs, logic errors, runtime issues, or performance problems in Go backend code. Triggers include: any mention of 'find bugs', 'debug this', 'something is wrong', 'why is this failing', 'check my code', 'review this', or when the user pastes Go code and asks what's wrong. Also trigger when the user mentions goroutine leaks, deadlocks, race conditions, panics, memory issues, or slow performance in Go. Use this skill for any Go backend code review involving HTTP servers, gRPC, PostgreSQL/sqlc/pgx, Redis, Kafka, WebSockets, or distributed systems code. Do NOT use for frontend, Python, Java, or non-Go codebases."
---

# Go Backend Bug Finder

A skill for systematically identifying logic errors, runtime panics, concurrency bugs, and performance issues in Go backend codebases.

---

## Overview

When this skill is triggered, analyze the provided Go code across these four layers in order:

1. **Go Runtime & Panic risks**
2. **Concurrency & Goroutine safety**
3. **Logic & Algorithm correctness**
4. **Performance & Resource management**

Always tailor analysis depth to what's provided. If only a snippet is given, flag assumptions. If the user describes a symptom (e.g., "it hangs"), prioritize that category.

---

## Analysis Layers

### 1. Runtime & Panic Risks

Check for conditions that cause a panic or unexpected crash at runtime:

- **Nil pointer dereference** — dereferencing a pointer, interface, map, or channel that may be nil
- **Index out of range** — slice/array access without bounds checking
- **Type assertion without ok check** — `v := x.(T)` instead of `v, ok := x.(T)`
- **Closed channel write** — writing to or closing an already-closed channel
- **Map concurrent access** — read/write to a map from multiple goroutines without a mutex
- **Divide by zero** — integer division where denominator can be 0
- **Uninitialized struct fields** — using zero value of a field where non-zero is assumed
- **Defer in loop** — deferred calls inside loops that won't fire until the function returns

```go
// BAD — panic if map is nil
func (s *Store) Get(key string) string {
    return s.cache[key] // panics if s.cache is nil
}

// GOOD
func (s *Store) Get(key string) string {
    if s.cache == nil {
        return ""
    }
    return s.cache[key]
}
```

---

### 2. Concurrency & Goroutine Safety

Go's concurrency model is powerful but has sharp edges:

- **Goroutine leaks** — goroutines started but never terminated (missing done channel, no context cancellation)
- **Race conditions** — shared state accessed from multiple goroutines without synchronization
- **Deadlock** — two goroutines waiting on each other; channel send/recv with no counterpart
- **Channel misuse** — unbuffered channels blocking unexpectedly; wrong direction assumptions
- **WaitGroup misuse** — `wg.Add()` called after `wg.Wait()`, or `wg.Done()` in wrong place
- **Context not propagated** — long-running operations ignoring `ctx.Done()`
- **Mutex copied** — passing `sync.Mutex` by value instead of pointer
- **Once misuse** — `sync.Once` wrapping mutable state that gets stale

```go
// BAD — goroutine leak, no way to stop
func StartWorker() {
    go func() {
        for {
            process()
        }
    }()
}

// GOOD — context-aware goroutine
func StartWorker(ctx context.Context) {
    go func() {
        for {
            select {
            case <-ctx.Done():
                return
            default:
                process()
            }
        }
    }()
}
```

**Tool to suggest:** `go run -race ./...` catches data races at runtime.

---

### 3. Logic & Algorithm Correctness

Language-agnostic logic bugs that produce wrong results:

- **Off-by-one errors** — loop bounds, slice ranges (`s[1:n]` vs `s[1:n+1]`)
- **Wrong operator** — `=` vs `==`, `||` vs `&&`, `&` vs `&&`
- **Shadowed variables** — `:=` inside an if/for block shadowing outer variable
- **Error ignored** — `_, err` or missing `if err != nil` check
- **Wrong error wrapping** — using `errors.New` instead of `fmt.Errorf("%w", err)` losing context
- **Incorrect nil check on interface** — an interface holding a typed nil is not nil
- **Slice mutation surprise** — modifying a slice returned from a function mutates the original
- **Pointer in loop** — capturing loop variable by address, all pointers point to same value

```go
// BAD — all goroutines share the same i
for i := 0; i < 5; i++ {
    go func() {
        fmt.Println(i) // always prints 5
    }()
}

// GOOD — pass i as argument
for i := 0; i < 5; i++ {
    go func(i int) {
        fmt.Println(i)
    }(i)
}
```

```go
// BAD — interface nil trap
var p *MyStruct = nil
var i MyInterface = p
fmt.Println(i == nil) // prints false!
```

---

### 4. Performance & Resource Management

- **Connection/resource leaks** — `*sql.Rows`, `http.Response.Body`, files not closed; use `defer`
- **N+1 query problem** — database queries inside a loop; suggest bulk query or JOIN
- **Inefficient string concatenation** — using `+` in a loop; suggest `strings.Builder`
- **Unnecessary allocations** — creating slices/maps without pre-allocated capacity
- **Repeated JSON marshal/unmarshal** — in hot paths; suggest caching or streaming
- **Missing context timeout** — DB/HTTP calls without deadline; can hang indefinitely
- **Blocking call in goroutine pool** — saturating a worker pool with slow I/O

```go
// BAD — response body leak
resp, err := http.Get(url)
if err != nil {
    return err
}
// forgot to defer resp.Body.Close()

// GOOD
resp, err := http.Get(url)
if err != nil {
    return err
}
defer resp.Body.Close()
```

---

## Domain-Specific Checks

### PostgreSQL / sqlc / pgx

- `pgx` connections not returned to pool (missing `defer conn.Release()`)
- `sql.Rows.Close()` not deferred
- Missing transaction rollback on error path
- `pgx.ErrNoRows` not handled distinctly from other errors
- Bulk inserts done in a loop instead of using `pgx.CopyFrom` or `UNNEST`

### Redis

- Missing TTL on `SET` calls — keys accumulate forever
- `KEYS *` in production — blocks Redis; use `SCAN` instead
- Race condition between `GET` and `SET` without using Lua script or `SET NX`
- Not handling `redis.Nil` error on cache miss

### HTTP / Middleware

- Missing `http.TimeoutHandler` or request context timeout
- Writing to `http.ResponseWriter` after handler returns
- Not calling `next.ServeHTTP` in middleware chain
- Ignoring `r.Context().Done()` in long-running handlers

### Kafka (franz-go / sarama)

- Missing `consumer.CommitMessages` after processing
- Not handling rebalance events
- Producing without checking delivery errors
- Single consumer for high-throughput topic — suggest consumer group

### WebSocket

- Not reading messages in a loop (connection goes stale)
- Missing `SetReadDeadline` / `SetWriteDeadline`
- Concurrent writes to `*websocket.Conn` without mutex

---

## Output Format

For each bug found, report in this structure:

```
## Bug N — [Category]: [Short title]

**Location:** func/file name, line if available
**Severity:** Critical | High | Medium | Low
**Problem:** What is wrong and why it is a bug
**Trigger:** When/how this bug manifests
**Fix:**
[corrected code snippet]
```

**Severity guide:**

- **Critical** — causes panic, data loss, or silent wrong result
- **High** — goroutine leak, resource leak, race condition
- **Medium** — logic error producing wrong output in edge cases
- **Low** — performance issue, style, missing error context

---

## Checklist (use before finalizing output)

- [ ] Checked all error return paths
- [ ] Checked all goroutines for leak/cancellation
- [ ] Checked all defers (especially in loops)
- [ ] Checked all interface nil comparisons
- [ ] Checked loop variable capture in goroutines
- [ ] Checked all resource close/release calls
- [ ] Checked context propagation to I/O calls
- [ ] Checked for missing mutex on shared state

---

## Usage Tips

- If the user only describes a symptom (e.g., "my service hangs under load"), focus on **Concurrency** first.
- If the user says "wrong output" or "incorrect result", focus on **Logic** first.
- If the user says "memory keeps growing", focus on **Resource Management** and **Goroutine leaks**.
- Always suggest `go vet ./...` and `go run -race ./...` as a first step when the full codebase is available.
