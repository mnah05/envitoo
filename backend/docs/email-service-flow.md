Here is the master blueprint for your architecture. You can take this directly to your terminal, fire up Neovim, and start laying down the foundation.

We will approach this with structured intensity, breaking the project down into a clear tech tree so you can build, test, and verify each node independently before wiring them all together.

### **Project Blueprint: High-Fidelity Event-Driven Email Dispatch**

**Core Objective:** Build a bulletproof, asynchronous background job system that guarantees exactly-once email delivery while protecting the primary database from external API latencies and localized node failures.

**The Tech Stack:**

* **Language:** Go (Optimized for high-concurrency, lightweight goroutines, and native M1 silicon performance).
* **Primary State:** PostgreSQL (Handles transactional data integrity).
* **Message Broker:** Redis (Handles high-throughput, volatile queue states).
* **Queue Engine:** `asynq` (Go-native Redis queue management).
* **External Provider:** Resend API.

---

### **Execution Tech Tree**

#### **Node 1: The Database Foundation (PostgreSQL)**

* **Action:** Create the `outbox_events` table.
* **Specs:** Needs an `id` (UUID), `event_type` (String), `payload` (JSONB), and `created_at` (Timestamp).
* **Goal:** Establish the durable buffer. This table should remain empty 99% of the time, acting strictly as a fail-safe between the database and Redis.

#### **Node 2: The Application Server Gateway (Go)**

* **Action:** Write the core HTTP handler and the atomic database transaction.
* **Specs:** In a single `tx.Commit()`, insert the primary business data (e.g., the user record) alongside the email payload into the `outbox_events` table.
* **Standardized Return:** Ensure the API returns the exact structure you require for clean client-side parsing:
```json
{
  "data": {
    "status": "queued",
    "action": "email_dispatch_initiated",
    "reference_id": "outbox_uuid_here"
  },
  "error": null
}

```



#### **Node 3: The Outbox Relay (The Drainer)**

* **Action:** Build a standalone goroutine that runs alongside your main server or as a totally separate binary.
* **Specs:** It runs a continuous loop (`time.Ticker`). It `SELECT`s new rows from `outbox_events`, marshals the payload into an `asynq.NewTask`, executes `client.Enqueue()`, and upon receiving no error from Redis, immediately executes a `DELETE` on that Postgres row.
* **Goal:** Decouple the fast database write from the network-dependent queue insertion.

#### **Node 4: The Worker Pool & Execution (Asynq + Resend)**

* **Action:** Set up the `asynq.Server` and define the worker handler for your specific `event_type`.
* **Specs:**
* **Rate Limiting:** Implement a global Redis token bucket (via Lua script) limiting execution to 5 operations per second.
* **Idempotency:** Generate a server-side hash (`SHA256(event_type + outbox_id)`). Use `SETNX` in Redis to ensure the task hasn't already been processed during a previous zombie worker state.
* **Dispatch:** Call the Resend API.


* **Goal:** Consume the queue efficiently, respect third-party rate limits, and ensure network drops never result in duplicate emails.

---

### **Vibe Coding the Prototype**

Because this architecture separates concerns so cleanly, you can prompt your way through prototyping the individual components incredibly fast. You can use one-shot prompts to generate the boilerplate `asynq` worker configurations and the outbox polling loop independently, testing them in isolation before bridging them.

Which node of the tech tree are you unlocking first?
