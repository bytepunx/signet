# Pulls the service bundle once, then waits for signet to report a change and
# for this replica to hold the restart lock before exiting, the Elixir
# counterpart of the Go client's examples/restart-on-change. From client-elixir/:
#
#   mix run examples/restart_on_change.exs -- --addr signet.internal:8443 \
#     --socket unix:///run/spire/sockets/agent.sock --trust-domain example.org \
#     --namespace default --service example --lock-ttl 30000 --debounce 10000
#
# Or in Docker, from the repository root:
#
#   scripts/test-client-elixir.sh mix run examples/restart_on_change.exs -- --addr ...
#
# The pattern:
#  1. Fetch the bundle once at startup and configure the app in memory.
#  2. Serve traffic normally.
#  3. Block on wait_for_restart, which returns only once signet reports a
#     change AND this replica has acquired the restart lock, so at most one
#     replica restarts at a time fleet-wide.
#  4. Do your own graceful shutdown (drain in-flight requests, close resources).
#  5. Release the lock, then exit 0. Kubernetes starts a fresh process, which
#     repeats from step 1 with the new configuration.

{opts, _, _} =
  OptionParser.parse(System.argv(),
    strict: [
      addr: :string,
      socket: :string,
      trust_domain: :string,
      namespace: :string,
      service: :string,
      lock_ttl: :integer,
      debounce: :integer
    ]
  )

addr = opts[:addr] || "localhost:8443"
socket = opts[:socket] || "unix:///run/spire/sockets/agent.sock"
trust_domain = opts[:trust_domain] || "example.org"
namespace = opts[:namespace] || "default"
service = opts[:service] || "example"
# The lock TTL must cover your graceful shutdown; the debounce absorbs rapid
# successive edits. Both are milliseconds.
lock_ttl = opts[:lock_ttl] || 30_000
debounce = opts[:debounce] || 10_000

fail = fn step, reason ->
  IO.puts(:stderr, "#{step}: #{inspect(reason)}")
  System.halt(1)
end

{channel, source} =
  case Signet.Workload.connect(addr, socket, trust_domain) do
    {:ok, channel, source} -> {channel, source}
    {:error, reason} -> fail.("dial", reason)
  end

# 1. Fetch once at startup; configure the app in memory. Never write this to
# the environment or disk.
request = %Signet.V1.GetServiceBundleRequest{namespace: namespace, service: service}

case Signet.V1.SecretsService.Stub.get_service_bundle(channel, request, timeout: 10_000) do
  {:ok, bundle} -> IO.puts("configured config_version=#{bundle.config_version}")
  {:error, reason} -> fail.("GetServiceBundle", reason)
end

# 2. Serve traffic normally (omitted; this is a minimal example).

# 3. Block until a change is reported and this replica holds the lock.
lock =
  case Signet.Restart.wait_for_restart(channel, namespace, service, lock_ttl, debounce) do
    {:ok, lock} -> lock
    {:error, reason} -> fail.("wait_for_restart", reason)
  end

IO.puts(
  "restart lock acquired token=#{Signet.Restart.token(lock)} " <>
    "expires_at=#{Signet.Restart.expires_at(lock)}"
)

# 4. Graceful shutdown: drain in-flight requests, close resources (omitted).

# 5. Release the lock for the next waiting replica, then exit 0. Kubernetes
# restarts the pod; the new process fetches the bundle fresh.
:ok = Signet.Restart.release(lock)
Signet.Workload.disconnect(channel, source)
System.halt(0)
