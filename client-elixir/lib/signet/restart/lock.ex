defmodule Signet.Restart.Lock do
  @moduledoc """
  Holds signet's restart lock for one namespace and service, the way the Go
  client's `AcquireLock` and `Lock` do.

  `acquire/5` opens an `AcquireRestartLock` stream, enters the queue, and
  blocks until the server answers `ACQUIRED`, queueing behind the current
  holder. The lock is held for as long as the stream stays open: `release/1`
  half-closes it, which hands the lock to the next waiting replica.

  The TTL must cover the whole time between acquiring the lock and releasing
  it; signet has no server-side default. While the lock is held, heartbeats go
  out every TTL/4, signet's documented convention: four consecutive missed
  heartbeats exhaust the TTL. Each `TTL_EXTENDED` answer moves `expires_at/1`.

  If the stream fails or the server closes it before `release/1`, or a
  heartbeat cannot be sent, the lock is lost: the owner gets one
  `{:lock_lost, lock, reason}` message, and never one after a release. There
  is nothing to undo; another replica may now restart concurrently, so log it.
  """

  use GenServer

  alias Signet.Restart.Streams
  alias Signet.V1.AcquireRestartLockRequest

  @typedoc """
  Options for `acquire/5`.

    * `:streams` - the `Signet.Restart.Streams` implementation, default the gRPC one
    * `:owner` - the process told when the lock is lost, default the caller
    * `:timeout` - how long to wait for `ACQUIRED`, in milliseconds, default `:infinity`
  """
  @type option :: {:streams, module()} | {:owner, pid()} | {:timeout, timeout()}

  @doc """
  Acquires the restart lock for `namespace`/`service` on `channel`, blocking
  until it is held or the attempt fails, and returns the lock process, linked
  to the caller. `ttl_ms` is the lock TTL in milliseconds; the wire carries
  whole seconds, at least one, and heartbeats go out every `ttl_ms / 4`.

  A failed attempt closes the stream and is an error value, never an exit:
  `{:error, :timeout}` when `ACQUIRED` did not come within `:timeout`, the
  stream's error, or `:closed` when the server ended the stream first.
  """
  @spec acquire(GRPC.Channel.t() | term(), String.t(), String.t(), pos_integer(), [option()]) ::
          {:ok, pid()} | {:error, term()}
  def acquire(channel, namespace, service, ttl_ms, opts \\ [])

  def acquire(_channel, _namespace, _service, ttl_ms, _opts)
      when not (is_integer(ttl_ms) and ttl_ms > 0) do
    {:error, "ttl must be > 0, got #{inspect(ttl_ms)}"}
  end

  def acquire(channel, namespace, service, ttl_ms, opts) do
    opts = Keyword.put_new(opts, :owner, self())
    {:ok, pid} = GenServer.start(__MODULE__, {channel, namespace, service, ttl_ms, opts})

    case GenServer.call(pid, :acquire, :infinity) do
      :ok ->
        Process.link(pid)
        {:ok, pid}

      {:error, reason} ->
        GenServer.stop(pid)
        {:error, reason}
    end
  end

  @doc "The token identifying this acquisition, for audit logs."
  @spec token(GenServer.server()) :: String.t()
  def token(lock), do: GenServer.call(lock, :token)

  @doc "The lock's current expiry, moved by each acknowledged heartbeat."
  @spec expires_at(GenServer.server()) :: DateTime.t() | nil
  def expires_at(lock), do: GenServer.call(lock, :expires_at)

  @doc "Whether `release/1` has been called."
  @spec released?(GenServer.server()) :: boolean()
  def released?(lock), do: GenServer.call(lock, :released?)

  @doc """
  Releases the lock by closing the stream, for the next waiting replica. Call
  it once your own graceful shutdown is complete. Idempotent; the process stays
  up so `token/1` and `expires_at/1` keep answering.
  """
  @spec release(GenServer.server()) :: :ok
  def release(lock), do: GenServer.call(lock, :release)

  @doc "Releases the lock and stops the process."
  @spec stop(GenServer.server()) :: :ok
  def stop(lock) do
    :ok = release(lock)
    GenServer.stop(lock)
  end

  @impl true
  def init({channel, namespace, service, ttl_ms, opts}) do
    Process.flag(:trap_exit, true)

    state = %{
      streams: Keyword.get(opts, :streams, Streams.GRPC),
      owner: Keyword.fetch!(opts, :owner),
      timeout: Keyword.get(opts, :timeout, :infinity),
      channel: channel,
      request: %AcquireRestartLockRequest{
        namespace: namespace,
        service: service,
        ttl_seconds: max(div(ttl_ms, 1_000), 1)
      },
      interval_ms: max(div(ttl_ms, 4), 1),
      stream: nil,
      reader: nil,
      token: nil,
      expires_at: nil,
      released: false,
      lost: false,
      timer: nil
    }

    {:ok, state}
  end

  @impl true
  def handle_call(:acquire, _from, state) do
    case open_and_await(state) do
      {:ok, state} -> {:reply, :ok, schedule_heartbeat(state)}
      {:error, reason, state} -> {:reply, {:error, reason}, state}
    end
  end

  def handle_call(:token, _from, state), do: {:reply, state.token, state}
  def handle_call(:expires_at, _from, state), do: {:reply, state.expires_at, state}
  def handle_call(:released?, _from, state), do: {:reply, state.released, state}

  def handle_call(:release, _from, %{released: true} = state), do: {:reply, :ok, state}

  def handle_call(:release, _from, state) do
    if state.timer, do: Process.cancel_timer(state.timer)
    :ok = state.streams.close_lock(state.stream)
    {:reply, :ok, %{state | released: true, timer: nil}}
  end

  @impl true
  def handle_info({:lock_resp, reader, response}, %{reader: reader} = state) do
    {:noreply, apply_response(state, response)}
  end

  # The stream ended while we still held the lock: the server closed it, or
  # it failed. After a release this is the close we asked for.
  def handle_info({:stream_ended, reader, reason}, %{reader: reader} = state) do
    {:noreply, report_lost(%{state | reader: nil}, reason)}
  end

  def handle_info(:heartbeat, %{released: true} = state), do: {:noreply, state}

  def handle_info(:heartbeat, state) do
    case state.streams.send_lock(state.stream, %AcquireRestartLockRequest{heartbeat: true}) do
      :ok -> {:noreply, schedule_heartbeat(state)}
      {:error, reason} -> {:noreply, report_lost(%{state | timer: nil}, reason)}
    end
  end

  def handle_info({:EXIT, reader, _reason}, %{reader: reader} = state), do: {:noreply, state}

  def handle_info(_other, state), do: {:noreply, state}

  @impl true
  def terminate(_reason, %{reader: reader}) when is_pid(reader),
    do: Process.exit(reader, :shutdown)

  def terminate(_reason, _state), do: :ok

  # Opens the stream, enters the queue, and waits for ACQUIRED; on any failure
  # the stream is closed and the reader shut down before the error is returned.
  defp open_and_await(state) do
    with {:ok, stream} <- state.streams.open_lock(state.channel),
         :ok <- send_initial(state, stream) do
      reader = start_reader(state.streams, stream)
      state = %{state | stream: stream, reader: reader}

      case await_acquired(state, deadline(state.timeout)) do
        {:ok, state} ->
          {:ok, state}

        {:error, reason} ->
          :ok = state.streams.close_lock(stream)
          Process.exit(reader, :shutdown)
          {:error, reason, %{state | reader: nil}}
      end
    else
      {:error, reason} -> {:error, reason, state}
    end
  end

  defp send_initial(state, stream) do
    case state.streams.send_lock(stream, state.request) do
      :ok ->
        :ok

      {:error, reason} ->
        :ok = state.streams.close_lock(stream)
        {:error, reason}
    end
  end

  defp await_acquired(%{reader: reader} = state, deadline) do
    receive do
      {:lock_resp, ^reader, %{message_type: :MESSAGE_TYPE_ACQUIRED} = response} ->
        {:ok, apply_response(state, response)}

      {:lock_resp, ^reader, _queue_position_or_other} ->
        await_acquired(state, deadline)

      {:stream_ended, ^reader, reason} ->
        {:error, reason}
    after
      remaining(deadline) -> {:error, :timeout}
    end
  end

  defp deadline(:infinity), do: :infinity
  defp deadline(timeout_ms), do: System.monotonic_time(:millisecond) + timeout_ms

  defp remaining(:infinity), do: :infinity
  defp remaining(deadline), do: max(deadline - System.monotonic_time(:millisecond), 0)

  defp apply_response(state, %{message_type: :MESSAGE_TYPE_ACQUIRED} = response) do
    %{state | token: response.token, expires_at: to_datetime(response.expires_at, state)}
  end

  defp apply_response(state, %{message_type: :MESSAGE_TYPE_TTL_EXTENDED} = response) do
    %{state | expires_at: to_datetime(response.expires_at, state)}
  end

  defp apply_response(state, _queue_position_or_unspecified), do: state

  defp to_datetime(nil, state), do: state.expires_at

  defp to_datetime(%{seconds: seconds, nanos: nanos}, _state) do
    DateTime.from_unix!(seconds * 1_000_000 + div(nanos, 1_000), :microsecond)
  end

  defp schedule_heartbeat(state) do
    %{state | timer: Process.send_after(self(), :heartbeat, state.interval_ms)}
  end

  defp report_lost(%{released: true} = state, _reason), do: state
  defp report_lost(%{lost: true} = state, _reason), do: state

  defp report_lost(state, reason) do
    send(state.owner, {:lock_lost, self(), reason})
    %{state | lost: true}
  end

  # The reader owns the response side for its lifetime, forwarding each
  # response and then the reason the stream ended: :closed, or its error.
  defp start_reader(streams, stream) do
    parent = self()

    spawn_link(fn ->
      reason =
        case streams.lock_responses(stream) do
          {:ok, responses} -> forward(responses, parent)
          {:error, reason} -> reason
        end

      send(parent, {:stream_ended, self(), reason})
    end)
  end

  defp forward(responses, parent) do
    Enum.reduce_while(responses, :closed, fn
      {:ok, response}, _ ->
        send(parent, {:lock_resp, self(), response})
        {:cont, :closed}

      {:error, reason}, _ ->
        {:halt, reason}
    end)
  end
end
