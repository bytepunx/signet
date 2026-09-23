# Coordinated rolling restarts

signet provides a distributed lock primitive that process managers use to
serialise service restarts across replicas. When a bundle change notification
arrives, each replica's process manager acquires the restart lock before fetching
the new bundle and restarting the child process. Only one restart proceeds at a
time; all others queue and wait.

## Design

### Why a lock instead of a client-side delay

Without coordination, N replicas receiving the same change notification will all
attempt to restart simultaneously. Staggering with random jitter works but
requires tuning and still allows overlapping restarts under certain conditions.
The restart lock guarantees at most one active restart at any moment, regardless
of fleet size.

### Why bidirectional streaming instead of a poll/reserve loop

The lock is modelled as an open gRPC stream:

- **Held** = stream is open
- **Released** = stream closed (gracefully or due to client disconnect)

This eliminates the need for an explicit release call, handles crashed process
managers automatically (TCP disconnect = release), and lets waiting clients block
on the stream rather than polling. No thundering herd on release.

### Database backing for multi-replica signet

Lock records are stored in CockroachDB with a TTL `expires_at` timestamp. This
ensures:

- Any signet replica can see the current lock state
- Locks expire if the holder crashes without closing the stream (TTL safety net)
- A background sweeper on each signet instance removes expired locks every 2
  seconds and unblocks local waiters

### TTL is the client's responsibility

signet has no server-side default TTL. The process manager must choose a TTL
long enough to cover the full restart window (image pull + service startup +
health check). If the TTL might be exceeded under normal conditions, the process
manager must send heartbeats to extend it.

---

## Message flow

```
process manager                       signet
───────────────                       ──────
AcquireRestartLockRequest             ─→ enter queue
  namespace, service, ttl_seconds
                                      ←─ QUEUE_POSITION{position: 2}
                                      ←─ QUEUE_POSITION{position: 1}  (holder released)
                                      ←─ ACQUIRED{token, expires_at}
[fetch GetServiceBundle]
[start new child process]
[send heartbeats every ttl/4]         ─→ heartbeat=true, ttl_seconds=N
                                      ←─ TTL_EXTENDED{expires_at}
[health check passes]
[drain and stop old child]
close stream                          ─→ lock released, next waiter notified
```

No event is sent until the lock is granted. Position updates are delivered as
the queue shrinks (preceding holders release or disconnect).

---

## Proto reference

```proto
rpc AcquireRestartLock(stream AcquireRestartLockRequest)
    returns (stream AcquireRestartLockResponse);
```

### `AcquireRestartLockRequest`

| Field         | Type   | Required   | Description |
|---------------|--------|------------|-------------|
| `namespace`   | string | First msg  | Lock namespace |
| `service`     | string | First msg  | Lock service name |
| `ttl_seconds` | int32  | First msg  | Lock TTL in seconds. Must be > 0. On heartbeat messages sets the new TTL extension duration; 0 reuses the original. |
| `heartbeat`   | bool   | Subsequent | When true, extends the lock TTL |

### `AcquireRestartLockResponse`

| `message_type`    | Fields set              | Meaning |
|-------------------|-------------------------|---------|
| `QUEUE_POSITION`  | `position`              | Waiting; position is 1-based place in queue. Re-sent on each change. |
| `ACQUIRED`        | `token`, `expires_at`   | Lock granted. Keep stream open to hold. |
| `TTL_EXTENDED`    | `expires_at`            | Heartbeat acknowledged; new expiry confirmed. |

---

## Process manager integration

The intended client is a process manager (supervisor) embedded in the service's
base image. The service itself does not interact with signet directly.

```
┌─────────────────────────────────────────────────────────┐
│  pod                                                    │
│                                                         │
│  ┌──────────────────────────┐   ┌─────────────────────┐ │
│  │  process manager         │   │  service (child)    │ │
│  │                          │   │                     │ │
│  │  WatchServiceBundle ─────┼───┼→ change notification│ │
│  │  AcquireRestartLock      │   │                     │ │
│  │  GetServiceBundle        │   │                     │ │
│  │  start new child process │   │                     │ │
│  │  health check            │   │                     │ │
│  │  handoff + drain old     │   │                     │ │
│  │  close lock stream       │   │                     │ │
│  └──────────────────────────┘   └─────────────────────┘ │
└─────────────────────────────────────────────────────────┘
```

### Restart sequence

```go
// 1. Receive change notification (from WatchServiceBundle on a separate stream).
// 2. Acquire the restart lock — blocks until this replica's turn.
lockStream, _ := client.AcquireRestartLock(ctx)

// Send the initial request.
lockStream.Send(&signetv1.AcquireRestartLockRequest{
    Namespace:  "payments",
    Service:    "api",
    TtlSeconds: 120, // must cover full restart window
})

// Drain queue position updates until ACQUIRED.
for {
    msg, err := lockStream.Recv()
    if err != nil { panic(err) }
    switch msg.MessageType {
    case signetv1.AcquireRestartLockResponse_MESSAGE_TYPE_QUEUE_POSITION:
        log.Printf("restart lock: queue position %d", msg.Position)
        continue
    case signetv1.AcquireRestartLockResponse_MESSAGE_TYPE_ACQUIRED:
        log.Printf("restart lock acquired (token=%s, expires=%s)", msg.Token, msg.ExpiresAt.AsTime())
    }
    break
}

// 3. Start heartbeat goroutine (interval = ttl / 4).
go func() {
    for range time.Tick(30 * time.Second) {
        lockStream.Send(&signetv1.AcquireRestartLockRequest{Heartbeat: true})
        msg, _ := lockStream.Recv() // MESSAGE_TYPE_TTL_EXTENDED
        log.Printf("lock TTL extended to %s", msg.ExpiresAt.AsTime())
    }
}()

// 4. Fetch fresh bundle.
bundle, _ := client.GetServiceBundle(ctx, &signetv1.GetServiceBundleRequest{
    Namespace: "payments",
    Service:   "api",
})

// 5. Start new child process with updated configuration.
newChild := startChild(bundle)

// 6. Wait for health check.
newChild.WaitHealthy(ctx)

// 7. Hand off: drain existing traffic from old child, then stop it.
oldChild.Drain()
oldChild.Stop()

// 8. Close the lock stream — releases the lock for the next waiting replica.
lockStream.CloseSend()
```

### Heartbeat cadence

Set the heartbeat interval to `ttl_seconds / 4`. This means 4 consecutive missed
heartbeats exhaust the TTL. The process manager should keep heartbeating until
the lock is explicitly released (stream closed).

On each heartbeat, `ttl_seconds` may be omitted (0) to reuse the original TTL,
or set to a new value to extend by a different duration.

### TTL selection

| Scenario | Suggested TTL |
|----------|---------------|
| Stateless service, fast start | 30–60s |
| Service with DB migrations | 180–300s |
| Service pulling large images | 120–180s |

If unsure, err toward a longer TTL — the lock is released immediately when the
stream closes, so a generous TTL only affects crash recovery time, not normal
operation.

---

## Access control

`AcquireRestartLock` follows the same convention-first policy as other signet
RPCs. A workload whose SPIFFE ID encodes `ns/<namespace>/sa/<service>` can
acquire the lock for that namespace/service without an explicit policy entry.

---

## Operational notes

### Lock stuck after signet restart

If signet restarts while a holder has the lock, the DB record persists and the
TTL sweeper on the new signet instance will clean it up within `expires_at` time.
Ensure TTLs are not set excessively long (hours) to limit this recovery window.

### Viewing current lock holders

```sql
SELECT namespace, service, token, acquired_at, expires_at,
       expires_at - now() AS ttl_remaining
FROM restart_locks
ORDER BY acquired_at;
```

### Forcing a lock release (emergency)

```sql
DELETE FROM restart_locks WHERE namespace = 'payments' AND service = 'api';
```

This is safe — the next process manager to call `AcquireRestartLock` will acquire
normally. The previous holder's stream will error on its next heartbeat attempt
(token no longer matches) and the process manager should treat this as a lost-lock
condition and re-acquire.
