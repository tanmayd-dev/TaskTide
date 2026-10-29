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

## 2. TaskTide's Approach: At-Least-Once Delivery with Duplicate Suppression

TaskTide guarantees that every submitted job is executed **at least once**; "exactly once" execution is **best-effort**, backed by duplicate suppression at the coordinator rather than by transport-level guarantees. The mechanism combines database persistence, transactionally claimed leases, DB-driven lease extensions, heartbeats, and idempotent submissions:

1. **Persistent State**: Jobs are stored in a durable PostgreSQL database. A job is never in-memory only.
2. **Atomic Transaction-Bound Claims**: When a worker polls, the coordinator claims pending jobs inside a single transaction using `FOR UPDATE SKIP LOCKED`: the job's status, its lease row, and an execution record are committed together, so a thundering herd of concurrent pollers can never issue the same job twice.
3. **DB-Driven Lease Extensions**: While a job executes, the worker extends its lease through the `ExtendLease` RPC, which rewrites the lease's `expires_at` in the database. A healthy worker (lease renewed every 10 s against a 30 s lease window) is therefore never reclaimed mid-execution anymore; renewal only fails when the worker has actually lost the lease.
4. **Worker Heartbeats**: Active workers send periodic heartbeats (every 5 seconds) to refresh their active status.
5. **Failure Recovery That Accounts for Retries**: The coordinator's recovery loop reclaims jobs from dead workers (heartbeat older than 15 s) and from expired leases, then routes them through one code path: while the retry budget remains (`RetryCount < MaxRetries`), the job's retry count is incremented and it is rescheduled as `pending` after the next backoff window. A job whose budget is exhausted is moved to `dead_letter`, a record is added to the DLQ, and (when the job belongs to a workflow) its task execution is marked failed so downstream tasks never fan out.
6. **Saturated Exponential Backoff With Jitter**: Retries are scheduled with an exponential delay of 2 s base at attempt 0, doubling per retry until the shift cap (10) and capped hard at 300 s, with random jitter up to half the backoff so failed workers do not redeliver in lockstep.
7. **Idempotent Submissions**: Clients may submit with an `idempotency_key`; concurrent resubmissions of the same key resolve to a single job row (see Section 3).

This combination guarantees that no job is lost (liveness), even if workers crash or experience network dropouts. However, exactly-once *execution* is still best-effort: duplicate suppression holds under contention (claims are transactional, submissions are deduplicated, reclaimers skip leases a worker has renewed), but a worker that completes a job in the narrow instant its lease expires and is reclaimed by another worker can still cause a duplicate execution. Closing that final window requires application-level idempotency on the worker side (Section 3).

---

## 3. Achieving Effectively Exactly-Once via Idempotency

To achieve the safety of exactly-once execution, TaskTide shifts the guarantee from the transport layer to the **application layer** using **Idempotency Keys** and an explicit trust boundary:

### Coordinator Idempotency
When submitting a job, clients can supply an `idempotency_key`. The coordinator stores this key with a `UNIQUE` constraint in the database and inserts via `ON CONFLICT (idempotency_key) DO NOTHING`:
- If a client resubmits a job due to a timeout or lost confirmation, the insert resolves as a no-op; the coordinator falls back to resolving the existing job by `idempotency_key` and returns the original job ID and status instead of executing the payload again. This makes concurrent duplicate submits race-safe: N concurrent submissions of the same key all return the same single job row.
- The submission's retry budget is clamped server-side (values outside `[0, 20]` are replaced by 3), so a resubmitting client cannot arbitrarily inflate the retry budget of an already-submitted job.

### Worker Idempotency
Workers should design their job handlers to be idempotent:
- **Write Deduplication**: Before writing side-effects (e.g. inserting records, charging a credit card), workers check if the execution ID has already completed the operation.
- **Atomic Operations**: Using database transactions or conditional updates (`UPDATE ... WHERE status = 'pending'`) to ensure that state transitions occur exactly once.

### Trust Boundary
It must be stated explicitly how the safety properties above are scoped: all coordinator RPCs (`SubmitJob`, `PollJobs`, `CompleteJob`, `ExtendLease`, ...) run as **plaintext gRPC without TLS**. TaskTide treats the coordinator's RPC surface as internal and considers the *container/private network in which the services run to be the trusted boundary* — the services are bound to container networks, and the published gRPC port exists only so host-side tooling (CLI, E2E harness) can dial in. Inside that boundary there is no authentication or TLS, by design.

If the coordinator's gRPC port is ever exposed beyond the container network, requests are only as trustworthy as the network they transit: **mTLS (or an equivalent TLS-terminating front) becomes mandatory** before allowing external access. Without it, any party that can reach the port can poll, claim, and complete jobs.

By combining **at-least-once delivery** with **idempotency keys**, TaskTide delivers the exact reliability characteristics of an exactly-once queue without violating the theoretical limits of distributed computing.
