defmodule Signet.FakeRestartStreams do
  @moduledoc """
  A `Signet.Restart.Streams` whose answers a test scripts in advance and that
  reports every call to the test.

  `start/2` returns the term tests pass where a `GRPC.Channel` would go: a
  reference under which an Agent holds the script and the test's pid, so the
  test and the watcher's or lock's reader processes share it.

  The watch side is scripted one answer per `watch/2` call, `{:ok, enumerable}`
  or `{:error, reason}`; once the script is exhausted a stream that stays open
  and silent is returned, as a real one is between changes. The lock side
  answers `open_lock/1` from the `:open_lock` script (`:open` or
  `{:error, reason}`, `:open` once the script is exhausted) and, when open,
  starts a `Signet.FakeRestartStreams.LockStream` the test feeds with `push/2`
  after the fact, so the reader blocks on it as it would on a live stream.
  """

  @behaviour Signet.Restart.Streams

  alias Signet.FakeRestartStreams.LockStream
  alias Signet.V1.{AcquireRestartLockResponse, WatchServiceBundleResponse}

  @doc """
  Starts a fake for one test and returns the channel term to hand to the code
  under test. `script` takes `:watch`, a list of `watch/2` answers, and
  `:open_lock`, a list of `open_lock/1` answers; both default to empty.
  """
  @spec start(keyword(), pid()) :: reference()
  def start(script, test_pid) do
    channel = make_ref()

    state = %{
      watch: Keyword.get(script, :watch, []),
      open_lock: Keyword.get(script, :open_lock, []),
      test: test_pid
    }

    {:ok, _} = Agent.start_link(fn -> state end, name: name(channel))
    channel
  end

  @impl true
  def watch(channel, request) do
    Agent.get_and_update(name(channel), fn
      %{watch: [answer | rest], test: test} = state ->
        send(test, {:watch, channel, request})
        {answer, %{state | watch: rest}}

      %{test: test} = state ->
        send(test, {:watch, channel, request})
        {{:ok, silent()}, state}
    end)
  end

  @impl true
  def open_lock(channel) do
    answer =
      Agent.get_and_update(name(channel), fn
        %{open_lock: [answer | rest]} = state -> {answer, %{state | open_lock: rest}}
        state -> {:open, state}
      end)

    case answer do
      :open ->
        test = Agent.get(name(channel), & &1.test)
        {:ok, stream} = LockStream.start_link(test)
        send(test, {:lock_opened, channel, stream})
        {:ok, stream}

      {:error, _} = error ->
        error
    end
  end

  @impl true
  def send_lock(stream, request), do: LockStream.send_request(stream, request)

  @impl true
  def lock_responses(stream), do: {:ok, LockStream.responses(stream)}

  @impl true
  def close_lock(stream), do: LockStream.close(stream)

  @doc "Feeds the lock stream's response side: `{:ok, response}` or `{:error, reason}`."
  @spec push(pid(), {:ok, AcquireRestartLockResponse.t()} | {:error, term()}) :: :ok
  def push(stream, item), do: LockStream.push(stream, item)

  @doc "The requests sent on the lock stream so far, oldest first."
  @spec sent(pid()) :: [Signet.V1.AcquireRestartLockRequest.t()]
  def sent(stream), do: LockStream.sent(stream)

  @doc "Whether `close_lock/1` was called on the lock stream."
  @spec closed?(pid()) :: boolean()
  def closed?(stream), do: LockStream.closed?(stream)

  @doc "A bundle-changed notification."
  @spec changed() :: WatchServiceBundleResponse.t()
  def changed, do: %WatchServiceBundleResponse{event_type: :EVENT_TYPE_CHANGED}

  @doc "The server's ACQUIRED message."
  @spec acquired(String.t(), DateTime.t()) :: AcquireRestartLockResponse.t()
  def acquired(token, %DateTime{} = expires_at) do
    %AcquireRestartLockResponse{
      message_type: :MESSAGE_TYPE_ACQUIRED,
      token: token,
      expires_at: timestamp(expires_at)
    }
  end

  @doc "The server's QUEUE_POSITION message."
  @spec queue_position(pos_integer()) :: AcquireRestartLockResponse.t()
  def queue_position(position) do
    %AcquireRestartLockResponse{message_type: :MESSAGE_TYPE_QUEUE_POSITION, position: position}
  end

  @doc "The server's TTL_EXTENDED message."
  @spec ttl_extended(DateTime.t()) :: AcquireRestartLockResponse.t()
  def ttl_extended(%DateTime{} = expires_at) do
    %AcquireRestartLockResponse{
      message_type: :MESSAGE_TYPE_TTL_EXTENDED,
      expires_at: timestamp(expires_at)
    }
  end

  defp timestamp(%DateTime{} = at) do
    micros = DateTime.to_unix(at, :microsecond)

    %Google.Protobuf.Timestamp{
      seconds: div(micros, 1_000_000),
      nanos: rem(micros, 1_000_000) * 1_000
    }
  end

  # A stream that stays open and delivers nothing, as a real one does while
  # nothing changes.
  defp silent, do: Stream.repeatedly(fn -> Process.sleep(:infinity) end)

  defp name(channel), do: {:global, {__MODULE__, channel}}
end

defmodule Signet.FakeRestartStreams.LockStream do
  @moduledoc """
  One fake `AcquireRestartLock` stream: a queue of responses the test pushes,
  pulled one at a time by the reader consuming `responses/1`, and a record of
  every request sent. Closing it ends the response side once the queue drains,
  as the server does when a client half-closes.
  """

  use GenServer

  def start_link(test_pid), do: GenServer.start_link(__MODULE__, test_pid)

  def push(stream, item), do: GenServer.call(stream, {:push, item})

  def send_request(stream, request), do: GenServer.call(stream, {:send, request})

  def close(stream), do: GenServer.call(stream, :close)

  def sent(stream), do: GenServer.call(stream, :sent)

  def closed?(stream), do: GenServer.call(stream, :closed?)

  @doc "The response side as an enumerable; each pull blocks until an item or the close."
  def responses(stream) do
    Stream.resource(
      fn -> stream end,
      fn stream ->
        case GenServer.call(stream, :pull, :infinity) do
          {:item, item} -> {[item], stream}
          :halt -> {:halt, stream}
        end
      end,
      fn _ -> :ok end
    )
  end

  @impl true
  def init(test_pid) do
    {:ok, %{test: test_pid, queue: :queue.new(), waiting: nil, sent: [], closed: false}}
  end

  @impl true
  def handle_call({:push, item}, _from, %{waiting: nil} = state) do
    {:reply, :ok, %{state | queue: :queue.in(item, state.queue)}}
  end

  def handle_call({:push, item}, _from, %{waiting: waiting} = state) do
    GenServer.reply(waiting, {:item, item})
    {:reply, :ok, %{state | waiting: nil}}
  end

  def handle_call(:pull, from, state) do
    case :queue.out(state.queue) do
      {{:value, item}, queue} -> {:reply, {:item, item}, %{state | queue: queue}}
      {:empty, _} when state.closed -> {:reply, :halt, state}
      {:empty, _} -> {:noreply, %{state | waiting: from}}
    end
  end

  def handle_call({:send, _request}, _from, %{closed: true} = state) do
    {:reply, {:error, :closed}, state}
  end

  def handle_call({:send, request}, _from, state) do
    send(state.test, {:lock_sent, self(), request})
    {:reply, :ok, %{state | sent: [request | state.sent]}}
  end

  def handle_call(:close, _from, state) do
    if state.waiting, do: GenServer.reply(state.waiting, :halt)
    {:reply, :ok, %{state | closed: true, waiting: nil}}
  end

  def handle_call(:sent, _from, state), do: {:reply, Enum.reverse(state.sent), state}

  def handle_call(:closed?, _from, state), do: {:reply, state.closed, state}
end
