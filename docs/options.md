# DST practice: project options

Goal: learn deterministic simulation testing (DST) by building small systems
where most of the effort goes into the simulator, not the system under test
(SUT). Every project here is chosen so the SUT fits in roughly 80 to 400 lines
of Go and the simulator around it is the interesting part.

Design rule that makes all of this work: the SUT never spawns a goroutine,
never reads the wall clock, and never touches the filesystem or network
directly. It receives a Clock, a Storage, a Network, and a Rand at
construction. Production passes real ones. The simulator passes fakes.

## Core DST patterns

Each project below is tagged with the patterns it exercises.

| ID  | Pattern                          | What it means                                                                                              |
|-----|----------------------------------|------------------------------------------------------------------------------------------------------------|
| P1  | Seeded randomness                | One PRNG drives workload, faults, and schedule. A failure prints `seed=N`. Rerunning the seed reproduces it. |
| P2  | Fake clock                       | No wall clock. The simulator advances time and fires timers. Includes per-node clock skew and drift.        |
| P3  | Fake storage                     | Write, Sync, Rename, Delete behind an interface. Simulates torn writes, reordered writes, lost unsynced data. |
| P4  | Fake network                     | Messages delayed, dropped, duplicated, reordered, partitioned. Per-link control.                            |
| P5  | Reference model and invariants   | A plain map or counter that says what the SUT should contain. Checked after every step or at the end.       |
| P6  | Deterministic scheduler          | No goroutines. Actors are state machines. The simulator picks who steps next from the seeded PRNG.          |
| P7  | Crash and restart                | A process dies at any point, loses volatile state, restarts from storage, and must recover correctly.       |
| P8  | Fault injection schedule         | Fail the Nth op, or fail with probability p, all from the seed. Ramp faults up and down during a run.        |
| P9  | Trace, replay, shrink            | Record the event trace. Replay a seed. Minimize the trace to the shortest one that still fails.             |
| P10 | History checking                 | Record concurrent client ops and check the history is linearizable or serializable against the model.       |
| P11 | Liveness and convergence         | Turn faults off at the end and assert the system settles: replicas agree, leader elected, queue drains.     |

Reusable simulator building blocks you end up with, and which patterns they carry:

- `sim/rand`: seeded PRNG plus helpers like `Chance(p)`, `Pick(slice)`, `Shuffle`. P1
- `sim/clock`: virtual time, timers, per-node skew. P2
- `sim/disk`: in-memory files with crash semantics and injected errors. P3, P7, P8
- `sim/net`: in-memory message bus with delay, drop, dup, partition. P4, P8
- `sim/sched`: actor registry, `Step()` loop, run-until-quiescent. P6, P11
- `sim/trace`: event log, seed printing, replay, shrinker. P9
- `sim/check`: linearizability checker for small histories. P10

## Projects, ordered by scope

Tiers are rough. Within a tier, earlier items are smaller. Each entry lists the
SUT, the invariants to check, patterns covered, and an SUT size guess.

### Tier 0: warmups, one sitting each

1. **Retry with exponential backoff and jitter**
   SUT: a function that retries a fallible op with capped exponential backoff.
   Invariants: retry times match the schedule, jitter stays in bounds, total attempts capped, gives up at deadline.
   Patterns: P1, P2, P5.
   Size: ~40 lines. Purpose: learn the fake clock and nothing else.

2. **Token bucket rate limiter**
   SUT: token bucket with refill rate and burst.
   Invariants: no sliding window of N seconds admits more than the limit; a starved caller eventually gets a token.
   Patterns: P1, P2, P5, P11.
   Size: ~50 lines.

3. **TTL cache with size bound**
   SUT: LRU cache where entries expire.
   Invariants: never returns an expired entry, never exceeds capacity, hit for anything the model says is live.
   Patterns: P1, P2, P5.
   Size: ~80 lines.

4. **Two bank accounts, concurrent transfers**
   SUT: N accounts, M transfer actors written as state machines (read A, read B, write A, write B).
   Invariants: total money conserved, no negative balance if that is a rule.
   Patterns: P1, P5, P6, P9.
   Size: ~80 lines. Purpose: the first hand-written scheduler. You will find the lost-update bug in seconds.

5. **Bounded producer/consumer queue with deadlock detection**
   SUT: bounded queue, producers and consumers as actors that block.
   Invariants: every item produced is consumed exactly once in FIFO order; the scheduler reports when all actors are blocked.
   Patterns: P5, P6, P9, P11.
   Size: ~100 lines.

### Tier 1: single node, crash-safe storage

6. **Atomic config file replace**
   SUT: write to temp file, fsync, rename over the target.
   Invariants: after any crash the file is exactly the old content or exactly the new content, never empty or mixed.
   Patterns: P3, P7, P8, P9.
   Size: ~40 lines. The smallest project that shows why fsync ordering matters. Remove one Sync call and watch it fail.

7. **Crash-safe append-only log**
   SUT: append records with length prefix and checksum, read all back, recover after crash by truncating the partial tail.
   Invariants: every acknowledged record is recovered, records come back in order, unacknowledged tail may be dropped but never corrupts earlier records.
   Patterns: P1, P3, P5, P7, P8, P9.
   Size: ~120 lines.

8. **WAL-only key-value store**
   SUT: the log from item 7 plus an in-memory map rebuilt on startup. Put, Get, Delete. No flush, no compaction.
   Invariants: `kv.Get(k) == model[k]` for all keys after any sequence of ops and crashes.
   Patterns: P1, P3, P5, P7, P8, P9.
   Size: ~150 lines. Recommended first "real" project. This is steps 1 through 7 of the original LSM plan.

9. **Bitcask-style log KV with compaction**
   SUT: item 8 plus multiple log segments and a merge step that rewrites live entries into a new segment then deletes the old ones.
   Invariants: logical contents identical before and after compaction, a crash mid-compaction never loses data, disk usage bounded.
   Patterns: P3, P5, P7, P8, P9.
   Size: ~300 lines. The step right before an LSM. Compaction under crash is where the bugs live.

10. **Snapshot and restore to a fake object store**
    SUT: periodically snapshot the KV from item 8 to a fake S3 (Put, Get, List, conditional Put), truncate the local log, restore from snapshot plus log tail on cold start.
    Invariants: restore reproduces the model, a crash between snapshot upload and log truncation is safe.
    Patterns: P3, P4, P5, P7, P8.
    Size: ~200 lines on top of item 8.

11. **Content-addressed object store with garbage collection**
    SUT: store blobs by hash, trees pointing at blobs, refs pointing at trees. A GC deletes unreachable objects. Loosely git-shaped.
    Invariants: every object reachable from a ref is retrievable after any crash; GC never deletes a reachable object even when a ref update races the GC.
    Patterns: P3, P5, P6, P7, P8.
    Size: ~250 lines. First project with a Cursor "git at any scale" flavor.

### Tier 2: single node, time and concurrency

12. **Job queue with leases and visibility timeouts**
    SUT: enqueue, claim with a lease, ack, lease expiry returns the job to the queue. Workers as actors that crash mid-job.
    Invariants: every job is eventually completed; with idempotency keys, each job's effect happens once; no job is held by two workers at the same time.
    Patterns: P1, P2, P5, P6, P7, P8, P11.
    Size: ~200 lines. Directly relevant to agentic platforms. A tool-call runner is this.

13. **Durable workflow engine, Temporal-lite**
    SUT: a workflow is a sequence of steps with side effects; each step result is appended to an event log; on crash the workflow replays the log and resumes at the first unfinished step.
    Invariants: every side effect happens exactly once (or at-least-once with dedup, your call), every workflow completes, replay is deterministic.
    Patterns: P2, P3, P5, P7, P8, P9.
    Size: ~250 lines. Very relevant to agent orchestration.

14. **Agent tool-call loop with a simulated model**
    SUT: a loop that asks a fake LLM (seeded responses) for the next tool call, runs tools with timeouts, supports cancellation and a token or dollar budget.
    Invariants: no tool execution outlives a cancel, budget never exceeded, every started tool call ends in exactly one of success, timeout, or cancel.
    Patterns: P1, P2, P5, P6, P8.
    Size: ~200 lines.

15. **Sandbox lifecycle manager, e2b-shaped**
    SUT: a controller that creates, execs into, pauses, and kills sandboxes through a flaky backend API, persisting its own state; the controller can crash.
    Invariants: no leaked sandbox (backend has one the controller forgot), no sandbox billed twice, no exec sent to a dead sandbox, idle sandboxes killed within the timeout.
    Patterns: P2, P3, P4, P5, P7, P8, P11.
    Size: ~300 lines.

16. **Multi-tenant fair scheduler**
    SUT: many tenants submit tasks against a fixed pool of slots; the scheduler enforces per-tenant quotas and fairness.
    Invariants: no tenant exceeds quota, no tenant starves while it has work and the pool has capacity, slots never oversubscribed.
    Patterns: P1, P2, P5, P6, P11.
    Size: ~150 lines.

### Tier 3: two or three nodes, unreliable network

17. **Retrying RPC client with exactly-once effects**
    SUT: client sends requests with idempotency tokens over a lossy network; server dedups.
    Invariants: each logical request's effect applied once, client eventually learns the result, dedup table bounded if you add expiry.
    Patterns: P1, P2, P4, P5, P8, P11.
    Size: ~120 lines. Cleanest introduction to the fake network.

18. **Webhook delivery with at-least-once and receiver dedup**
    SUT: a sender with a durable outbox and retries, a receiver that dedups by event id. Sender crashes between send and mark-delivered.
    Invariants: every event delivered, receiver applies each event once, outbox drains when the network heals.
    Patterns: P2, P3, P4, P5, P7, P8, P11.
    Size: ~150 lines.

19. **Heartbeat failure detector**
    SUT: nodes send heartbeats; each node marks peers alive or suspected based on timeouts. Optionally phi-accrual.
    Invariants: a truly crashed node is suspected within a bound; a live node on a healthy link is never suspected; no flapping beyond a bound under jitter.
    Patterns: P2, P4, P7, P8, P11.
    Size: ~100 lines. Add per-node clock drift and watch a naive implementation break.

20. **Distributed lock with leases and clock skew**
    SUT: one lock server, N clients acquire with a lease, renew, release. Clients have skewed clocks. Client and server can crash.
    Invariants: never two holders at once (safety), a lock held by a crashed client becomes free within lease duration (liveness).
    Patterns: P2, P4, P5, P7, P8, P11.
    Size: ~150 lines. The classic "fencing token" lesson.

21. **Rendezvous or consistent hashing with membership churn**
    SUT: pure placement function plus a membership view; the simulator adds and removes nodes.
    Invariants: every key has exactly one owner per view, membership change of one node moves at most about 1/N of keys, all nodes with the same view agree on every key's owner.
    Patterns: P1, P5.
    Size: ~80 lines. Pure and small, but the simulator drives churn. Cursor uses rendezvous hashing for repo placement.

22. **Primary-backup replication with synchronous ack**
    SUT: client, primary, backup. Primary writes locally, forwards to backup, acks the client after the backup acks. On primary crash the backup is promoted.
    Invariants: every acknowledged write survives primary crash; backup never exposes a write the primary has not acknowledged (or you document the anomaly).
    Patterns: P3, P4, P5, P7, P8, P10.
    Size: ~200 lines. First project where a linearizability check pays off.

23. **Two-phase commit with participant crashes**
    SUT: coordinator, two or three participants, prepare/commit/abort with durable logs on every node.
    Invariants: atomicity (all commit or all abort), no participant commits after another aborts; observe and document the blocking case when the coordinator dies after prepare.
    Patterns: P3, P4, P5, P7, P8, P11.
    Size: ~250 lines.

### Tier 4: replicated logs and convergence

24. **CAS-on-object-store log, Cursor "Continuity" core**
    SUT: a fake S3 with conditional Put (If-Match on ETag). Several stateless writers append entries to a log by reading the head, then CAS-ing the next slot. Readers catch up by conditional Get.
    Invariants: exactly one writer wins each slot, no acknowledged entry is lost, every reader eventually sees the same prefix, a writer crash between Put-object and CAS-head leaves garbage but never corruption.
    Patterns: P1, P4, P5, P7, P8, P10, P11.
    Size: ~200 lines. This is the git-at-any-scale architecture with the git parts removed: the object store is the consensus mechanism.

25. **Stateless servers with local warm cache verified against object store**
    SUT: item 24 plus servers that keep a local copy and verify freshness against the store's ETag before serving reads. Local cache can be wiped by a crash.
    Invariants: reads never return data older than the last acknowledged write (fully consistent reads), servers recover from an empty cache, bandwidth to the store stays bounded when healthy.
    Patterns: P3, P4, P5, P7, P8, P10.
    Size: ~150 lines on top of item 24. "Always correct when degraded, always fast when healthy."

26. **State-based CRDTs with anti-entropy gossip**
    SUT: G-Counter, then OR-Set, then LWW-Register. Replicas gossip full state on a timer over a partitioned network.
    Invariants: merge is commutative, associative, idempotent (property test); all replicas converge once partitions heal; converged value equals the model.
    Patterns: P1, P2, P4, P5, P8, P11.
    Size: ~200 lines.

27. **Gossip membership, SWIM-lite**
    SUT: nodes ping random peers, indirect ping via a third node on timeout, disseminate alive/suspect/dead.
    Invariants: a crashed node is declared dead everywhere within a bound; a live node is never declared dead on a healthy network; membership converges.
    Patterns: P2, P4, P7, P8, P11.
    Size: ~200 lines.

28. **Partitioned log with consumer groups, Kafka-lite**
    SUT: append-only partitions, consumers commit offsets, consumer crash triggers rebalance.
    Invariants: every message consumed at least once, with idempotent consumers exactly once, offsets never go backwards, no two live consumers own a partition at the same time.
    Patterns: P2, P3, P4, P5, P7, P8, P11.
    Size: ~300 lines.

### Tier 5: consensus, still small

29. **Leader election with leases on a shared store**
    SUT: N candidates race to write a lease record with CAS into the fake object store from item 24, renew it, step down on expiry. Skewed clocks.
    Invariants: at most one node believes it is leader at any instant given the skew bound; a leader is elected within a bound after the previous one dies.
    Patterns: P2, P4, P5, P7, P8, P11.
    Size: ~120 lines.

30. **Single-decree Paxos**
    SUT: proposers, acceptors, learners deciding one value. Roughly 150 lines of algorithm.
    Invariants: only one value is ever chosen; a chosen value was proposed by someone; with faults off, a value is eventually chosen.
    Patterns: P4, P5, P6, P7, P8, P9, P11.
    Size: ~150 lines. The canonical DST target. Shrinking a failing trace here is where P9 earns its keep.

31. **Raft leader election, then log replication**
    SUT: terms, votes, heartbeats first. Add AppendEntries and commit index second. Snapshots third if you still want more.
    Invariants: election safety (one leader per term), log matching, leader completeness, state machine safety; liveness after faults stop.
    Patterns: P2, P3, P4, P5, P7, P8, P9, P10, P11.
    Size: ~250 lines for election, ~500 with replication. The biggest thing on this list. Do it only after 24 and 30.

32. **Linearizability checker as its own project**
    SUT: the checker. Feed it histories from items 22, 24, or 31 and known-bad hand-written histories.
    Invariants: accepts every history the model says is valid, rejects the hand-written bad ones, runs in reasonable time on histories of a few hundred ops.
    Patterns: P5, P9, P10.
    Size: ~200 lines. Tooling as the project. Makes every later project stronger.

## Suggested paths

Storage path: 4, 6, 7, 8, 9, then grow 9 into the original LSM plan.

Agentic platform path: 1, 4, 12, 13, 14, 15, then 24 for durable shared state.

Distributed systems path: 4, 8, 17, 20, 22, 24, 25, 30, 31.

Cursor "git at any scale" path: 8, 11, 21, 24, 25, then add gossip from 27 for replica catch-up hints.

Whatever the path, do 4 and 8 first. Item 4 forces you to write the scheduler.
Item 8 forces you to write the fake disk and the reference model. Everything
else reuses those.

## Coverage matrix

|  # | Project                              | P1 | P2 | P3 | P4 | P5 | P6 | P7 | P8 | P9 | P10 | P11 |
|----|--------------------------------------|----|----|----|----|----|----|----|----|----|-----|-----|
|  1 | Retry with backoff                   | x  | x  |    |    | x  |    |    |    |    |     |     |
|  2 | Token bucket                         | x  | x  |    |    | x  |    |    |    |    |     | x   |
|  3 | TTL cache                            | x  | x  |    |    | x  |    |    |    |    |     |     |
|  4 | Bank transfers                       | x  |    |    |    | x  | x  |    |    | x  |     |     |
|  5 | Bounded queue, deadlock detect       |    |    |    |    | x  | x  |    |    | x  |     | x   |
|  6 | Atomic file replace                  |    |    | x  |    |    |    | x  | x  | x  |     |     |
|  7 | Crash-safe log                       | x  |    | x  |    | x  |    | x  | x  | x  |     |     |
|  8 | WAL-only KV                          | x  |    | x  |    | x  |    | x  | x  | x  |     |     |
|  9 | Bitcask KV with compaction           |    |    | x  |    | x  |    | x  | x  | x  |     |     |
| 10 | Snapshot to object store             |    |    | x  | x  | x  |    | x  | x  |    |     |     |
| 11 | Content-addressed store with GC      |    |    | x  |    | x  | x  | x  | x  |    |     |     |
| 12 | Job queue with leases                | x  | x  |    |    | x  | x  | x  | x  |    |     | x   |
| 13 | Durable workflow engine              |    | x  | x  |    | x  |    | x  | x  | x  |     |     |
| 14 | Agent tool-call loop                 | x  | x  |    |    | x  | x  |    | x  |    |     |     |
| 15 | Sandbox lifecycle manager            |    | x  | x  | x  | x  |    | x  | x  |    |     | x   |
| 16 | Multi-tenant fair scheduler          | x  | x  |    |    | x  | x  |    |    |    |     | x   |
| 17 | Exactly-once RPC client              | x  | x  |    | x  | x  |    |    | x  |    |     | x   |
| 18 | Webhook delivery with outbox         |    | x  | x  | x  | x  |    | x  | x  |    |     | x   |
| 19 | Heartbeat failure detector           |    | x  |    | x  |    |    | x  | x  |    |     | x   |
| 20 | Distributed lock with leases         |    | x  |    | x  | x  |    | x  | x  |    |     | x   |
| 21 | Rendezvous hashing with churn        | x  |    |    |    | x  |    |    |    |    |     |     |
| 22 | Primary-backup replication           |    |    | x  | x  | x  |    | x  | x  |    | x   |     |
| 23 | Two-phase commit                     |    |    | x  | x  | x  |    | x  | x  |    |     | x   |
| 24 | CAS log on object store              | x  |    |    | x  | x  |    | x  | x  |    | x   | x   |
| 25 | Warm cache verified by ETag          |    |    | x  | x  | x  |    | x  | x  |    | x   |     |
| 26 | CRDTs with gossip                    | x  | x  |    | x  | x  |    |    | x  |    |     | x   |
| 27 | SWIM-lite membership                 |    | x  |    | x  |    |    | x  | x  |    |     | x   |
| 28 | Kafka-lite partitioned log           |    | x  | x  | x  | x  |    | x  | x  |    |     | x   |
| 29 | Leader election via leases           |    | x  |    | x  | x  |    | x  | x  |    |     | x   |
| 30 | Single-decree Paxos                  |    |    |    | x  | x  | x  | x  | x  | x  |     | x   |
| 31 | Raft                                 |    | x  | x  | x  | x  |    | x  | x  | x  | x   | x   |
| 32 | Linearizability checker              |    |    |    |    | x  |    |    |    | x  | x   |     |

## Skipped on purpose

- Full LSM with leveled compaction: mostly database work. Grow item 9 into it once the simulator exists.
- Real Docker or Firecracker sandbox backends: item 15 uses a fake backend for a reason.
- Multi-Paxos, Raft with membership changes, Spanner-style clocks: too much algorithm per unit of simulator learning.
- Anything with real goroutines in the SUT: breaks P6 and makes every failure non-reproducible.
