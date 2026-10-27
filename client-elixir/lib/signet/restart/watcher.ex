defmodule Signet.Restart.Watcher do
  @moduledoc """
  Watches signet's bundle-change notifications for one namespace and service,
  the way the Go client's `WatchBundle` does.

  A reader process keeps a `WatchServiceBundle` stream open, reopening it with
  exponential backoff, 1s doubling to 30s and reset by any event received, when
  it ends or fails. Every `EVENT_TYPE_CHANGED` event reaches this process, which
  tells its owner.

  Go hands the owner a channel of capacity one, so rapid changes coalesce into
  a single pending signal. Here the owner gets one `{:bundle_changed, watcher}`
  message per burst: it is sent when a change arrives and none is outstanding,
  and the next is sent only once the owner has acknowledged the last with
  `acknowledge/1`. `next_change/2` does both at once, receive then acknowledge,
  as `<-changes` does in Go; a `GenServer` owner handles the message in
  `handle_info/2` and acknowledges there. The message says that at least one
  change happened since the owner last looked, never how many.
  """

  use GenServer

  alias Signet.Restart.Streams
  alias Signet.V1.WatchServiceBundleRequest

  @backoff_min_ms 1_000
  @backoff_max_ms 30_000

  @typedoc """
  Options for `start_link/4`.

    * `:streams` - the `Signet.Restart.Streams` implementation, default the gRPC one
    * `:owner` - the process to notify, default the caller
    * `:sleep` - replaces `Process.sleep/1` between reopen attempts, for tests
  """
  @type option ::
          {:streams, module()}
          | {:owner, pid()}
          | {:sleep, (non_neg_integer() -> :ok)}

  @doc "The reopen backoff bounds, in milliseconds: `{1_000, 30_000}`."
  @spec backoff_ms() :: {pos_integer(), pos_integer()}
  def backoff_ms, do: {@backoff_min_ms, @backoff_max_ms}

  @doc "The backoff that follows `current`: doubled, capped at 30s."
  @spec next_backoff(pos_integer()) :: pos_integer()
  def next_backoff(current), do: min(current * 2, @backoff_max_ms)

  @doc """
  Starts a watcher for `namespace`/`service` on `channel`, linked to the
  caller, notifying the owner of changes as the module doc describes.
  """
  @spec start_link(GRPC.Channel.t() | term(), String.t(), String.t(), [option()]) ::
          GenServer.on_start()
  def start_link(channel, namespace, service, opts \\ []) do
    opts = Keyword.put_new(opts, :owner, self())
    GenServer.start_link(__MODULE__, {channel, namespace, service, opts})
  end

  @doc """
  Marks the outstanding change as seen, so the next change is notified again.
  """
  @spec acknowledge(GenServer.server()) :: :ok
  def acknowledge(watcher), do: GenServer.call(watcher, :acknowledge)

  @doc """
  Waits for the next `{:bundle_changed, watcher}` message in the caller's
  mailbox and acknowledges it, returning `:ok`, or `:timeout` after `timeout`
  milliseconds with none. The caller must be the watcher's owner.
  """
  @spec next_change(pid(), timeout()) :: :ok | :timeout
  def next_change(watcher, timeout \\ :infinity) do
    receive do
      {:bundle_changed, ^watcher} -> acknowledge(watcher)
    after
      timeout -> :timeout
    end
  end

  @doc "Stops the watcher and closes its stream."
  @spec stop(GenServer.server()) :: :ok
  def stop(watcher), do: GenServer.stop(watcher)

  @impl true
  def init({channel, namespace, service, opts}) do
    Process.flag(:trap_exit, true)

    reader = %{
      streams: Keyword.get(opts, :streams, Streams.GRPC),
      sleep: Keyword.get(opts, :sleep, &Process.sleep/1),
      channel: channel,
      request: %WatchServiceBundleRequest{namespace: namespace, service: service},
      watcher: self()
    }

    state = %{
      owner: Keyword.fetch!(opts, :owner),
      outstanding: false,
      reader: spawn_link(fn -> watch_loop(reader, @backoff_min_ms) end)
    }

    {:ok, state}
  end

  @impl true
  def handle_call(:acknowledge, _from, state), do: {:reply, :ok, %{state | outstanding: false}}

  @impl true
  def handle_info({:changed, reader}, %{reader: reader, outstanding: false} = state) do
    send(state.owner, {:bundle_changed, self()})
    {:noreply, %{state | outstanding: true}}
  end

  # A change while one is outstanding: coalesced into the pending message.
  def handle_info({:changed, reader}, %{reader: reader} = state), do: {:noreply, state}

  # The reader never returns on its own; its exit is a bug in the stream, and
  # the owner learns of it through the link rather than from a silent watcher.
  def handle_info({:EXIT, reader, reason}, %{reader: reader} = state) do
    {:stop, {:reader_exited, reason}, %{state | reader: nil}}
  end

  def handle_info(_other, state), do: {:noreply, state}

  @impl true
  def terminate(_reason, %{reader: reader}) when is_pid(reader),
    do: Process.exit(reader, :shutdown)

  def terminate(_reason, _state), do: :ok

  # The Go watchLoop: open, forward every CHANGED event, and on any end sleep
  # the backoff and reopen. Any event received resets the backoff.
  defp watch_loop(reader, backoff) do
    backoff =
      case reader.streams.watch(reader.channel, reader.request) do
        {:ok, stream} -> consume(stream, reader, backoff)
        {:error, _reason} -> backoff
      end

    reader.sleep.(backoff)
    watch_loop(reader, next_backoff(backoff))
  end

  defp consume(stream, reader, backoff) do
    Enum.reduce_while(stream, backoff, fn
      {:ok, response}, _backoff ->
        if response.event_type == :EVENT_TYPE_CHANGED do
          send(reader.watcher, {:changed, self()})
        end

        {:cont, @backoff_min_ms}

      {:error, _reason}, backoff ->
        {:halt, backoff}
    end)
  end
end
