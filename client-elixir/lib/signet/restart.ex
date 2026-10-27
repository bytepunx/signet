defmodule Signet.Restart do
  @moduledoc """
  Coordinated restarts against signet's `SecretsService`, mirroring the Go
  client's `WatchBundle`, `AcquireLock`, and `WaitForRestart`.

  A service that pulls its own configuration from signet restarts to pick up a
  change. Three operations coordinate that so at most one replica restarts at
  a time, fleet-wide:

    * `watch_bundle/4` starts a `Signet.Restart.Watcher`: it follows signet's
      bundle-change notifications for a namespace and service, reconnecting
      with 1s-to-30s backoff, and tells its owner about changes with one
      `{:bundle_changed, watcher}` message per burst. `next_change/2` waits
      for that message and acknowledges it, as `<-changes` does in Go;
      `acknowledge/1` does the second half alone, for a `GenServer` owner that
      handles the message in `handle_info/2`.
    * `acquire_lock/5` blocks until this replica holds signet's restart lock,
      queueing behind the current holder, and returns a `Signet.Restart.Lock`
      that heartbeats every TTL/4 until `release/1`. If the lock is lost first,
      the owner gets one `{:lock_lost, lock, reason}` message.
    * `wait_for_restart/6` combines the two: it waits for the next change,
      debounces further changes, then acquires the lock.

  Durations are milliseconds, as Elixir timeouts are; the wire carries the TTL
  in whole seconds. The typical caller shape is the Go one:

      {:ok, lock} =
        Signet.Restart.wait_for_restart(channel, "default", "example", 30_000, 10_000)

      # graceful shutdown: drain in-flight requests, close resources
      :ok = Signet.Restart.release(lock)
      System.stop(0)

  Kubernetes restarts the pod and the new process fetches the bundle fresh;
  nothing here fetches it, and nothing writes secrets to the environment.
  """

  alias Signet.Restart.{Lock, Watcher}

  @typedoc "A bundle watcher, from `watch_bundle/4`."
  @type watcher :: pid()

  @typedoc "A held restart lock, from `acquire_lock/5` or `wait_for_restart/6`."
  @type lock :: pid()

  @doc """
  Starts watching bundle changes for `namespace`/`service` on `channel`; see
  `Signet.Restart.Watcher` for the options and the message protocol.
  """
  @spec watch_bundle(GRPC.Channel.t() | term(), String.t(), String.t(), [Watcher.option()]) ::
          {:ok, watcher()} | {:error, term()}
  def watch_bundle(channel, namespace, service, opts \\ []) do
    Watcher.start_link(channel, namespace, service, opts)
  end

  @doc """
  Waits for the watcher's next `{:bundle_changed, watcher}` message and
  acknowledges it: `:ok`, or `:timeout` after `timeout` milliseconds.
  """
  @spec next_change(watcher(), timeout()) :: :ok | :timeout
  def next_change(watcher, timeout \\ :infinity), do: Watcher.next_change(watcher, timeout)

  @doc "Acknowledges the outstanding change, so the watcher notifies the next burst."
  @spec acknowledge(watcher()) :: :ok
  def acknowledge(watcher), do: Watcher.acknowledge(watcher)

  @doc "Stops a watcher and its stream."
  @spec stop_watch(watcher()) :: :ok
  def stop_watch(watcher), do: Watcher.stop(watcher)

  @doc """
  Blocks until this replica holds the restart lock for `namespace`/`service`;
  `ttl_ms` must cover the time from acquiring to `release/1`. See
  `Signet.Restart.Lock.acquire/5` for the options.
  """
  @spec acquire_lock(GRPC.Channel.t() | term(), String.t(), String.t(), pos_integer(), [
          Lock.option()
        ]) :: {:ok, lock()} | {:error, term()}
  def acquire_lock(channel, namespace, service, ttl_ms, opts \\ []) do
    Lock.acquire(channel, namespace, service, ttl_ms, opts)
  end

  @doc "Releases the lock for the next waiting replica; idempotent."
  @spec release(lock()) :: :ok
  def release(lock), do: Lock.release(lock)

  @doc "The token identifying this acquisition, for audit logs."
  @spec token(lock()) :: String.t()
  def token(lock), do: Lock.token(lock)

  @doc "The lock's current expiry, moved by each acknowledged heartbeat."
  @spec expires_at(lock()) :: DateTime.t() | nil
  def expires_at(lock), do: Lock.expires_at(lock)

  @doc """
  Waits for the next bundle change, then while further changes keep arriving
  within `debounce_ms` of the last waits again (`0` acts on the first change
  at once), then acquires the restart lock with `ttl_ms` and returns it held.

  The watcher is stopped before the lock is acquired, whatever the outcome.
  `opts` are those of `watch_bundle/4` and `acquire_lock/5`; `:owner` applies
  to the lock only, the watcher reports to the caller.
  """
  @spec wait_for_restart(
          GRPC.Channel.t() | term(),
          String.t(),
          String.t(),
          pos_integer(),
          non_neg_integer(),
          keyword()
        ) :: {:ok, lock()} | {:error, term()}
  def wait_for_restart(channel, namespace, service, ttl_ms, debounce_ms, opts \\ [])

  def wait_for_restart(_channel, _namespace, _service, ttl_ms, _debounce_ms, _opts)
      when not (is_integer(ttl_ms) and ttl_ms > 0) do
    {:error, "ttl must be > 0, got #{inspect(ttl_ms)}"}
  end

  def wait_for_restart(_channel, _namespace, _service, _ttl_ms, debounce_ms, _opts)
      when not (is_integer(debounce_ms) and debounce_ms >= 0) do
    {:error, "debounce must be >= 0, got #{inspect(debounce_ms)}"}
  end

  def wait_for_restart(channel, namespace, service, ttl_ms, debounce_ms, opts) do
    watch_opts = Keyword.take(opts, [:streams, :sleep])

    with {:ok, watcher} <- watch_bundle(channel, namespace, service, watch_opts) do
      :ok = next_change(watcher)
      debounce(watcher, debounce_ms)
      stop_and_flush(watcher)

      acquire_lock(
        channel,
        namespace,
        service,
        ttl_ms,
        Keyword.take(opts, [:streams, :owner, :timeout])
      )
    end
  end

  defp debounce(_watcher, 0), do: :ok

  defp debounce(watcher, debounce_ms) do
    case next_change(watcher, debounce_ms) do
      :ok -> debounce(watcher, debounce_ms)
      :timeout -> :ok
    end
  end

  # A change notified between the last next_change and the stop would stay in
  # the caller's mailbox otherwise.
  defp stop_and_flush(watcher) do
    :ok = stop_watch(watcher)

    receive do
      {:bundle_changed, ^watcher} -> :ok
    after
      0 -> :ok
    end
  end
end
