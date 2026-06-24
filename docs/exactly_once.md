# Exactly-Once Semantics in Distributed Job Queues

This document discusses the theoretical impossibility of achieving guaranteed "exactly-once" delivery in a distributed system, and details how TaskTide achieves **effectively exactly-once** execution at the application level.

---

## 1. The Theoretical Impossibility

In distributed systems, the consensus and communication protocols are bound by fundamental limits:
- **The Two Generals Problem**: Demonstrates that it is impossible for two nodes to reach absolute agreement over an unreliable link. A sender cannot distinguish between a packet that was lost, a receiver that crashed, or a confirmation packet that was lost on its way back.
- **FLP Impossibility Result**: In an asynchronous network, no deterministic consensus protocol can guarantee liveness and safety in the presence of even a single unannounced process crash.

Because of these limits, a job queue coordinator cannot guarantee that a job is delivered to a worker and executed *exactly once* under all network partition and failure conditions:
1. If the coordinator sends a job to a worker, and the network connection drops before receiving an acknowledgment:
   - The coordinator must choose between safety (assuming the worker received and ran it, thus risk not running it if the worker didn't) or liveness (resending the job, thus risking running it twice).
2. If the worker executes the job successfully, but crashes or loses connection right before sending the completion acknowledgment:
   - The coordinator will eventually reclaim the job lease due to expiration and dispatch it to another worker, leading to double execution.

Therefore, **pure exactly-once delivery is a physical impossibility over unreliable networks.**

---

## 2. TaskTide's Approach: At-Least-Once Delivery

TaskTide guarantees that every submitted job is executed **at least once** by employing a combination of database persistence, lease-based locking, visibility timeouts, and heartbeats:

1. **Persistent State**: Jobs are stored in a durable PostgreSQL database. A job is never in-memory only.
2. **Lease-Based Locking**: When a worker pulls a job, it acquires a lease with a **visibility timeout** (e.g., 30 seconds). During this time, no other worker can see or poll this job.
3. **Worker Heartbeats**: Active workers send periodic heartbeats (every 5 seconds) to refresh their active status.
4. **Failure Recovery**:
   - If a worker crashes, its heartbeat expires. The coordinator detects the dead worker, releases all its leases, and resets the corresponding jobs to `pending`.
   - If a worker becomes unresponsive or experiences a long GC pause, its lease expires. The coordinator reclaims the job and places it back in the queue.

This combination guarantees that no job is lost (liveness), even if workers crash or experience network dropouts. However, it permits duplicate execution under partitions.

---

## 3. Achieving Effectively Exactly-Once via Idempotency

To achieve the safety of exactly-once execution, TaskTide shifts the guarantee from the transport layer to the **application layer** using **Idempotency Keys**:

### Coordinator Deduplication
When submitting a job, clients can supply an `idempotency_key`. The coordinator stores this key with a `UNIQUE` index constraint in the database.
- If a client resubmits a job due to a timeout or lost confirmation, the coordinator catches the duplicate key error, retrieves the existing job status, and returns it to the client without executing the job payload again.

### Worker Idempotency
Workers should design their job handlers to be idempotent:
- **Write Deduplication**: Before writing side-effects (e.g. inserting records, charging a credit card), workers check if the execution ID has already completed the operation.
- **Atomic Operations**: Using database transactions or conditional updates (`UPDATE ... WHERE status = 'pending'`) to ensure that state transitions occur exactly once.

By combining **at-least-once delivery** with **idempotency keys**, TaskTide delivers the exact reliability characteristics of an exactly-once queue without violating the theoretical limits of distributed computing.
