defmodule Signet.Restart.Streams do
  @moduledoc """
  The two `SecretsService` streams `Signet.Restart` depends on, as a behaviour
  so the bundle watcher and the restart lock can be tested against a fake.

  The default implementation, `Signet.Restart.Streams.GRPC`, calls
  `WatchServiceBundle` and the bidirectional `AcquireRestartLock` through
  `Signet.V1.SecretsService.Stub` on a channel `Signet.Workload.connect/4`
  opened.
  """

  alias Signet.V1.{AcquireRestartLockRequest, AcquireRestartLockResponse}
  alias Signet.V1.{WatchServiceBundleRequest, WatchServiceBundleResponse}

  @typedoc """
  The responses of one `WatchServiceBundle` stream, in order: each element is
  a `WatchServiceBundleResponse`, or the error that ended the stream.
  """
  @type watch_stream :: Enumerable.t({:ok, WatchServiceBundleResponse.t()} | {:error, term()})

  @typedoc "An open `AcquireRestartLock` stream; opaque to the caller."
  @type lock_stream :: term()

  @typedoc """
  The responses of one `AcquireRestartLock` stream, in order: each element is
  an `AcquireRestartLockResponse`, or the error that ended the stream.
  """
  @type lock_responses ::
          Enumerable.t({:ok, AcquireRestartLockResponse.t()} | {:error, term()})

  @doc """
  Opens a `WatchServiceBundle` stream for `request` on `channel`. The stream is
  consumed by the calling process.
  """
  @callback watch(GRPC.Channel.t(), WatchServiceBundleRequest.t()) ::
              {:ok, watch_stream()} | {:error, term()}

  @doc "Opens an `AcquireRestartLock` stream on `channel`; nothing is sent yet."
  @callback open_lock(GRPC.Channel.t()) :: {:ok, lock_stream()} | {:error, term()}

  @doc "Sends one request on an open lock stream: the initial request, or a heartbeat."
  @callback send_lock(lock_stream(), AcquireRestartLockRequest.t()) :: :ok | {:error, term()}

  @doc """
  The server's side of an open lock stream, consumed by the calling process.
  It ends when the server closes the stream or the stream fails.
  """
  @callback lock_responses(lock_stream()) :: {:ok, lock_responses()} | {:error, term()}

  @doc """
  Half-closes the request side of a lock stream, which releases the lock the
  stream holds. The response side ends when the server observes the close.
  """
  @callback close_lock(lock_stream()) :: :ok
end

defmodule Signet.Restart.Streams.GRPC do
  @moduledoc "The `Signet.Restart.Streams` over gRPC, through `Signet.V1.SecretsService.Stub`."

  @behaviour Signet.Restart.Streams

  alias Signet.V1.SecretsService.Stub

  @impl true
  def watch(channel, request) do
    Stub.watch_service_bundle(channel, request, timeout: :infinity)
  end

  # acquire_restart_lock/2 is a request-streaming call: it sends the headers
  # and returns the %GRPC.Client.Stream{} to send requests on, raising if the
  # connection process is gone rather than returning an error.
  @impl true
  def open_lock(channel) do
    {:ok, Stub.acquire_restart_lock(channel, timeout: :infinity)}
  rescue
    error -> {:error, error}
  end

  # send_request/3 returns the stream and raises when the transport refuses
  # the frame; the caller only needs to know whether the request went out.
  @impl true
  def send_lock(stream, request) do
    _ = GRPC.Stub.send_request(stream, request)
    :ok
  rescue
    error -> {:error, error}
  end

  @impl true
  def lock_responses(stream) do
    GRPC.Stub.recv(stream, timeout: :infinity)
  end

  @impl true
  def close_lock(stream) do
    _ = GRPC.Stub.end_stream(stream)
    :ok
  end
end
