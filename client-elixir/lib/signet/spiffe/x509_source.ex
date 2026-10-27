defmodule Signet.Spiffe.X509Source do
  @moduledoc """
  Holds the workload's current X.509-SVID and trust bundles, kept fresh from
  the Workload API's `FetchX509SVID` stream, the way go-spiffe's `X509Source`
  does.

  Starting the source blocks until the first identity arrives. It retries
  automatically, with no opt-in, if the Workload API reports "no identity
  issued" (a `PermissionDenied` status): SPIRE's controller-manager registers a
  brand-new pod's SPIFFE identity reactively, off the pod's own creation event,
  and that registration takes a few seconds to reach the node-local agent, so a
  freshly created pod's very first fetch can lose that race. The retry makes
  five attempts in all, sleeping 1s, 2s, 4s, and 8s between them, the schedule
  the Go client verified in production. Only that failure is retried: every
  other error (a bad socket path, a malformed response) is returned at once,
  so a real misconfiguration is never masked as a transient blip.

  Once started, the source reads updates from the stream as SPIRE rotates the
  SVID. If the stream ends or fails, the last identity stays current (it is
  valid until its certificate expires) and the stream is reopened with
  exponential backoff, 1s doubling to 30s and reset by the next update, as the
  Go client's bundle watcher does. `current/1` returns the latest
  `Signet.Spiffe.X509Context`. Stop the source with `stop/1` once the
  connections built from it are closed.
  """

  use GenServer

  alias Signet.Spiffe.{WorkloadAPI, X509Context}

  @backoff_ms [1_000, 2_000, 4_000, 8_000]
  @max_attempts length(@backoff_ms) + 1
  @reopen_backoff_min_ms 1_000
  @reopen_backoff_max_ms 30_000

  @typedoc """
  Options for `start_link/1`.

    * `:socket_path` - the Workload API socket, required
    * `:api` - the `Signet.Spiffe.WorkloadAPI` implementation, default the gRPC one
    * `:sleep` - replaces `Process.sleep/1` between retries, for tests
  """
  @type option ::
          {:socket_path, String.t()}
          | {:api, module()}
          | {:sleep, (non_neg_integer() -> :ok)}

  @doc "The retry schedule, in milliseconds, between the five attempts."
  @spec backoff_ms() :: [non_neg_integer()]
  def backoff_ms, do: @backoff_ms

  @doc """
  Starts a source linked to the caller and blocks until it holds an identity,
  or returns why it could not. The link is made only once the source is up, so
  a failed start is an error value to the caller, never an exit signal.
  """
  @spec start_link([option()]) :: {:ok, pid()} | {:error, term()}
  def start_link(opts) do
    with {:ok, pid} <- start(opts) do
      Process.link(pid)
      {:ok, pid}
    end
  end

  @doc "Starts an unlinked source; see `start_link/1`."
  @spec start([option()]) :: {:ok, pid()} | {:error, term()}
  def start(opts) do
    GenServer.start(__MODULE__, opts)
  end

  @doc "The current identity and trust bundles."
  @spec current(GenServer.server()) :: X509Context.t()
  def current(source), do: GenServer.call(source, :current)

  @doc "Stops the source, closing its Workload API stream."
  @spec stop(GenServer.server()) :: :ok
  def stop(source), do: GenServer.stop(source)

  @impl true
  def init(opts) do
    socket_path = Keyword.fetch!(opts, :socket_path)
    api = Keyword.get(opts, :api, WorkloadAPI.GRPC)
    sleep = Keyword.get(opts, :sleep, &Process.sleep/1)
    Process.flag(:trap_exit, true)

    state = %{
      socket_path: socket_path,
      api: api,
      sleep: sleep,
      context: nil,
      reader: nil,
      reopen_backoff_ms: @reopen_backoff_min_ms
    }

    case open_until_identity_issued(state, 1, nil) do
      {:ok, state} -> {:ok, state}
      {:error, reason} -> {:stop, reason}
    end
  end

  @impl true
  def handle_call(:current, _from, state), do: {:reply, state.context, state}

  @impl true
  def handle_info({:x509, reader, response}, %{reader: reader} = state) do
    state = %{state | reopen_backoff_ms: @reopen_backoff_min_ms}

    case X509Context.from_response(response) do
      {:ok, context} -> {:noreply, %{state | context: context}}
      {:error, _reason} -> {:noreply, state}
    end
  end

  # The stream ended or failed after the identity was issued: keep the last
  # identity and reopen after the backoff, without blocking callers meanwhile.
  def handle_info({:stream_ended, reader, _reason}, %{reader: reader} = state) do
    Process.send_after(self(), :reopen, state.reopen_backoff_ms)
    next = min(state.reopen_backoff_ms * 2, @reopen_backoff_max_ms)
    {:noreply, %{state | reader: nil, reopen_backoff_ms: next}}
  end

  def handle_info(:reopen, %{reader: nil} = state) do
    {:noreply, %{state | reader: start_reader(state.api, state.socket_path)}}
  end

  def handle_info({:EXIT, reader, _reason}, %{reader: reader} = state) do
    {:noreply, state}
  end

  def handle_info(_other, state), do: {:noreply, state}

  @impl true
  def terminate(_reason, %{reader: reader}) when is_pid(reader),
    do: Process.exit(reader, :shutdown)

  def terminate(_reason, _state), do: :ok

  # Opens the stream in a reader process and waits for its first message,
  # retrying per @backoff_ms while the Workload API says no identity is issued.
  defp open_until_identity_issued(_state, attempt, last_error) when attempt > @max_attempts do
    {:error, "no identity issued after #{@max_attempts} attempts: #{inspect(last_error)}"}
  end

  defp open_until_identity_issued(state, attempt, _last_error) do
    reader = start_reader(state.api, state.socket_path)

    receive do
      {:x509, ^reader, response} ->
        case X509Context.from_response(response) do
          {:ok, context} ->
            {:ok, %{state | context: context, reader: reader}}

          {:error, reason} ->
            Process.exit(reader, :shutdown)
            {:error, reason}
        end

      {:stream_ended, ^reader, reason} ->
        cond do
          not WorkloadAPI.no_identity_issued?(reason) ->
            {:error, reason}

          attempt < @max_attempts ->
            state.sleep.(Enum.at(@backoff_ms, attempt - 1))
            open_until_identity_issued(state, attempt + 1, reason)

          true ->
            open_until_identity_issued(state, attempt + 1, reason)
        end
    end
  end

  # The reader owns the gRPC stream for its lifetime, forwarding each response
  # and then the reason the stream ended: :closed, or the error that broke it.
  defp start_reader(api, socket_path) do
    parent = self()

    spawn_link(fn ->
      reason =
        case api.fetch_x509_svid(socket_path) do
          {:ok, stream} -> forward(stream, parent)
          {:error, reason} -> reason
        end

      send(parent, {:stream_ended, self(), reason})
    end)
  end

  defp forward(stream, parent) do
    Enum.reduce_while(stream, :closed, fn
      {:ok, response}, _ ->
        send(parent, {:x509, self(), response})
        {:cont, :closed}

      {:error, reason}, _ ->
        {:halt, reason}
    end)
  end
end
