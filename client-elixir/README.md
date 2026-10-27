# signet_client (Elixir)

Elixir client for [signet](https://github.com/bytepunx/signet), generated from
[bytepunx/signet-proto](https://github.com/bytepunx/signet-proto) and pinned
independently of the server's own release cycle (see `buf.gen.yaml`). It mirrors the
[Go client](../client-go)'s connection layer idiomatically: `{:ok, value}` and
`{:error, reason}` tuples instead of `(value, error)` pairs, a `GenServer` where Go uses a
goroutine, and the generated `GRPC.Stub` modules where Go has typed clients.

## Install

```elixir
def deps do
  [{:signet_client, "~> 0.1"}]
end
```

The package is not yet published to Hex; depend on it by path or git until it is.

## Building and testing

Nothing is installed on the host. From the repository root:

```sh
make test                              # every component's behaviour tests
scripts/test-client-elixir.sh          # this client alone
scripts/test-client-elixir.sh mix format
```

The script builds the toolchain image from `Dockerfile` (Elixir, Erlang/OTP, buf, and
`protoc-gen-elixir`, all pinned), then runs stub generation, `mix deps.get`, `mix compile
--warnings-as-errors`, and `mix test` inside it with this folder mounted at `/app`, so
`deps/`, `_build/`, and the generated `lib/gen/` persist on the host between runs,
gitignored. Any other `mix` or `buf` command can be passed as arguments.

## Regenerating stubs

```sh
scripts/test-client-elixir.sh buf generate
scripts/test-client-elixir.sh buf generate --template buf.gen.spiffe.yaml
```

Buf's schema registry hosts no remote plugin for Elixir or Erlang, so the BEAM clients
cannot be wired up as the Go, Python, Rust, TypeScript, and C# clients are (a `remote:`
plugin in `buf.gen.yaml`). `buf.gen.yaml` names `protoc-gen-elixir`, the escript the
`protobuf` hex package ships, as a local plugin, run inside the Docker toolchain image. An
Erlang client would take the same shape, with
[`gpb`](https://github.com/tomas-abrahamsson/gpb)'s `protoc-gen-gpb` for messages and
either [`grpcbox`](https://github.com/tsloughter/grpcbox) or this client's `grpc` package
(whose stubs are plain modules an Erlang application can call) for the service layer; none
is scaffolded. The
first template pulls `signet/v1` and `admin/v1` from `buf.build/bytepunx/signet-proto`
into `lib/gen/`; the second generates the SPIRE Workload API client from the vendored
`proto/spiffe/workload/workload.proto` under the `Spiffe.Workload` namespace. Edit the
module reference in `buf.gen.yaml` to pin a different schema version.

## Usage

### Workload access (SecretsService, SPIFFE mTLS)

```elixir
{:ok, channel, source} =
  Signet.Workload.connect("signet.internal:8443",
    "unix:///run/spire/sockets/agent.sock", "example.org")

{:ok, resp} =
  Signet.V1.SecretsService.Stub.get_secret(channel, %Signet.V1.GetSecretRequest{
    namespace: "default", service: "example", name: "api-key"
  })

Signet.Workload.disconnect(channel, source)
```

`connect/3` retries automatically, with no opt-in, if it loses the race between a
freshly created pod's SPIFFE identity registration and this call: SPIRE's
controller-manager registers a new pod's identity reactively, off the pod's own creation
event, and that registration takes a few seconds to reach the node-local SPIRE agent this
dials. On "no identity issued" (a `PermissionDenied` status, and only that failure;
anything else is returned at once) it makes five attempts in all, sleeping 1s, 2s, 4s, and
8s between them, the schedule the Go client verified in production. See
[bytepunx/signet-clients#33](https://github.com/bytepunx/signet-clients/issues/33).

The returned `source` is a `Signet.Spiffe.X509Source`: a process that holds the SVID and
trust bundles and keeps following the Workload API stream as SPIRE rotates them,
reopening it with backoff if it ends. A connection presents the SVID current when it was
opened, as a Go `tls.Config` does; a new connection picks up the latest. Stop the source
with the channel.

The channel also serves the subset of `GitOpsService` signet accepts from a workload's
own identity (`SyncBundle`, `PatchServiceConfig`, `GetSOPSPublicKey`) through
`Admin.V1.GitOpsService.Stub`, without an admin token.

See `examples/getsecret.exs` for a complete runnable example.

### SPIFFE support

Unlike the Python client, this client authorises the server the way the Go client's
`tlsconfig.AuthorizeMemberOf` does. Erlang's `ssl` lets a `verify_fun` inspect the peer
certificate after chain validation, and lets `server_name_indication: :disable` turn off
hostname verification, which an X.509-SVID (a certificate with only a SPIFFE URI SAN, no
DNS or IP name) would otherwise always fail. `Signet.Spiffe.Authorize.member_of/1` is that
`verify_fun`: the leaf must carry exactly one SPIFFE ID, and its trust domain must be the
one passed to `connect/3`. The trust bundle for that trust domain, from the SVID's own
bundle or a federated one, is the only set of roots the connection trusts.

The Workload API client is generated from the SPIFFE standard's `workload.proto`; no
third-party SPIFFE library exists for the BEAM. `Signet.Spiffe.WorkloadAPI` is the slice
this client depends on, as a behaviour, so `Signet.Spiffe.X509Source` is tested against a
fake with a scripted sequence of answers. The retry schedule, the authoriser, the DER
splitting of chains and bundles, and a loopback TLS handshake with the exact option set
`connect/3` uses are all covered without a network or a SPIRE agent.

### Operator access (AdminService/GitOpsService, bearer token)

```elixir
{:ok, channel} = Signet.Admin.connect("localhost:8444", token)
{:ok, status} = Admin.V1.AdminService.Stub.status(channel, %Admin.V1.StatusRequest{})
GRPC.Stub.disconnect(channel)
```

`Signet.Admin.connect/3` picks transport security from the address, as the Go client's
`DialAdmin` does: loopback addresses (`localhost`, `127.0.0.1`, `::1`, the documented
`kubectl port-forward` workflow) default to plaintext, and every other address is upgraded
to TLS automatically, verified against the operating system's trust store unless
`ca_pem: File.read!(path)` pins a CA bundle. Two options override that heuristic in
opposite directions: `force_tls: true` requests TLS even for a loopback address, and
`plaintext: true` forces insecure transport even for a non-loopback address, which is what
dialing a real in-cluster admin listener by its cluster-DNS name needs
([bytepunx/signet#19](https://github.com/bytepunx/signet/issues/19)): that listener is
plaintext behind the bearer token, not TLS-terminated, and without the override the
heuristic would pick TLS and the handshake would fail at once. Per-RPC bearer-token
authentication is unaffected either way. `force_tls` and `plaintext` are mutually
exclusive with each other, and `plaintext` with a non-empty `ca_pem`; either combination,
and an empty token, is an error before any connection is opened. See
[bytepunx/signet-clients#32](https://github.com/bytepunx/signet-clients/issues/32).

The bearer token is attached as a channel header, so every RPC on the channel carries
`authorization: Bearer <token>` over plaintext and TLS alike.

### Coordinated restarts

`Signet.Restart` lets a service pull its own configuration directly, in memory, and
coordinate a fleet-wide serialized restart when it changes, as the Go client's
`WatchBundle`, `AcquireLock`, and `WaitForRestart` do: no process host, and no secrets
written to the OS environment.

```elixir
{:ok, channel, source} =
  Signet.Workload.connect("signet.internal:8443",
    "unix:///run/spire/sockets/agent.sock", "example.org")

# Fetch once at startup, configure the app in memory ... serve traffic ...

# Block until signet reports a change AND this replica holds the restart lock
# (at most one replica restarts at a time, fleet-wide).
# Lock TTL 30s: must cover your graceful shutdown. Debounce 10s: absorbs rapid
# successive changes.
{:ok, lock} = Signet.Restart.wait_for_restart(channel, "default", "example", 30_000, 10_000)

# Do your own graceful shutdown: drain in-flight requests, close resources.

:ok = Signet.Restart.release(lock) # the next waiting replica acquires
Signet.Workload.disconnect(channel, source)
System.stop(0) # Kubernetes restarts the pod; the new process fetches fresh config
```

The three operations map one to one onto the Go client's:

- `Signet.Restart.watch_bundle/4` is `WatchBundle`: a `Signet.Restart.Watcher` process
  follows `WatchServiceBundle` for the namespace and service, reopening the stream with
  exponential backoff, 1s doubling to 30s and reset by any event received, when it ends or
  fails. Where Go hands back a channel of capacity one, the watcher sends its owner one
  `{:bundle_changed, watcher}` message per burst of changes: the next is sent only once the
  owner has acknowledged the last with `Signet.Restart.next_change/2` (receive then
  acknowledge, like `<-changes`) or `Signet.Restart.acknowledge/1` from a `GenServer`'s
  `handle_info/2`. The message says at least one change happened, never how many.
- `Signet.Restart.acquire_lock/5` is `AcquireLock`: it opens the bidirectional
  `AcquireRestartLock` stream, enters the queue, and blocks until `ACQUIRED`. The lock is a
  `Signet.Restart.Lock` process that heartbeats every TTL/4 (signet's convention: four
  consecutive missed heartbeats exhaust the TTL) and exposes `Signet.Restart.token/1` and
  `Signet.Restart.expires_at/1`, moved by each acknowledged heartbeat. The TTL must cover
  the whole time from acquiring to releasing; signet has no server-side default.
  `Signet.Restart.release/1` half-closes the stream, which hands the lock to the next
  waiting replica, and is idempotent. If the stream fails, the server closes it, or a
  heartbeat cannot be sent before you release, the owner gets one
  `{:lock_lost, lock, reason}` message, Go's `Lock.Lost()`: treat it as "another replica
  may now restart concurrently"; there is nothing to undo.
- `Signet.Restart.wait_for_restart/6` is `WaitForRestart`: it waits for the next change,
  keeps waiting while further changes arrive within the debounce, stops the watcher, and
  acquires the lock. It does not fetch the updated bundle: the next process instance fetches
  it during its own startup.

Durations are milliseconds, as Elixir timeouts are; the wire carries the TTL in whole
seconds, at least one.

The two streams sit behind the `Signet.Restart.Streams` behaviour, so the watcher and
the lock are tested against `Signet.FakeRestartStreams`, a scripted fake whose lock
stream the test feeds after the fact: coalescing, the backoff schedule, queueing,
heartbeat timing, expiry updates, loss, and idempotent release are all covered without a
network or a signet server.

See `examples/restart_on_change.exs` for a complete runnable example, and
[signet's restart-lock docs](https://github.com/bytepunx/signet/blob/main/docs/restart-lock.md)
for the underlying protocol.

### SOPS encryption

```elixir
{:ok, %{public_key: public_key}} =
  Admin.V1.GitOpsService.Stub.get_sops_public_key(channel, %Admin.V1.GetSOPSPublicKeyRequest{})

{:ok, yaml} = Signet.SopsEncrypt.encrypt_for_secret(public_key, "s3cr3t")
```

`Signet.SopsEncrypt.encrypt_for_secret/2` is the Go client's `EncryptForSecret`: it
produces the SOPS-encrypted YAML document a workload submits as a secret file's content
through `GitOpsService.SyncBundle`. `SyncBundle` stores content as-is and never encrypts
on the server's behalf, so a plaintext value sent through it fails to decrypt and is never
stored; this call is the way to produce a value signet persists. The document has a single
top-level `value` leaf, encrypted with AES-256-GCM under a fresh data key, and that data
key wrapped to one age recipient, signet's own public key from `GetSOPSPublicKey`. It is
the same shape `signet secret set` writes. The public key is validated before anything is
encrypted; an empty or malformed one is `{:error, reason}`.

age is implemented on OTP `:crypto` (X25519, HKDF-SHA256, ChaCha20-Poly1305), since no
age package exists on hex; see
[`../design/tech/age-elixir.md`](../design/tech/age-elixir.md). The behaviour tests check
the document's shape and decrypt it with the test suite's own age decryptor;
`make integration` (`scripts/integration-client-elixir.sh`) runs
`tests/integration/sops_encrypt_test.exs` in the same Docker toolchain image with the real
`sops` binary installed and has it decrypt the document, and refuse to with the wrong
identity.

## Dependencies

| Package | Why |
|---------|-----|
| `grpc` 1.0 | gRPC client: unary and streaming calls, interceptors, TLS, Unix sockets |
| `gun` 2.4 | The HTTP/2 transport `grpc` drives by default, and the adapter whose Unix-socket support the Workload API dial needs |
| `protobuf` 0.17 | Runtime for the generated stubs and the Google well-known types signet-proto imports; also the source of `protoc-gen-elixir` |

`mix hex.audit` reports an advisory on `gun` and `cowlib` (CVE-2026-43966, HTTP response
splitting when structured header values are built from attacker-controlled bytes). This
client sends fixed ASCII header values only, and `grpc` pins `gun ~> 2.4.0`; the entry in
[`../design/tech/grpc-elixir.md`](../design/tech/grpc-elixir.md) tracks it.
