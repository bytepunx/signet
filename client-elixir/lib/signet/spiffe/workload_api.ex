defmodule Signet.Spiffe.WorkloadAPI do
  @moduledoc """
  The slice of the SPIFFE Workload API this client depends on, as a behaviour
  so `Signet.Spiffe.X509Source` can be tested against a fake.

  The default implementation, `Signet.Spiffe.WorkloadAPI.GRPC`, calls
  `SpiffeWorkloadAPI.FetchX509SVID` over the agent's Unix socket with the
  `workload.spiffe.io: true` header the standard requires.
  """

  @typedoc """
  The responses of one `FetchX509SVID` stream, in order: each element is an
  `X509SVIDResponse`, or the error that ended the stream.
  """
  @type stream :: Enumerable.t({:ok, Spiffe.Workload.X509SVIDResponse.t()} | {:error, term()})

  @doc """
  Opens a `FetchX509SVID` stream against the Workload API at `socket_path`
  (`unix:///run/spire/sockets/agent.sock`, or the bare path). The stream is
  consumed by the calling process.
  """
  @callback fetch_x509_svid(socket_path :: String.t()) :: {:ok, stream()} | {:error, term()}

  @doc "Reports whether `error` is the Workload API's \"no identity issued\" failure: a `PermissionDenied` status."
  @spec no_identity_issued?(term()) :: boolean()
  def no_identity_issued?(%GRPC.RPCError{status: status}),
    do: status == GRPC.Status.permission_denied()

  def no_identity_issued?(_), do: false

  @doc "The gRPC target for a Workload API socket, accepting `unix://` and bare paths."
  @spec target(String.t()) :: String.t()
  def target("unix://" <> path), do: "unix://" <> path
  def target("unix:" <> path), do: "unix://" <> path
  def target("/" <> _ = path), do: "unix://" <> path
  def target(other), do: other
end

defmodule Signet.Spiffe.WorkloadAPI.GRPC do
  @moduledoc "The `Signet.Spiffe.WorkloadAPI` over gRPC on the agent's Unix socket."

  @behaviour Signet.Spiffe.WorkloadAPI

  alias Spiffe.Workload.SpiffeWorkloadAPI.Stub
  alias Spiffe.Workload.X509SVIDRequest

  @impl true
  def fetch_x509_svid(socket_path) do
    target = Signet.Spiffe.WorkloadAPI.target(socket_path)

    with {:ok, channel} <- GRPC.Stub.connect(target, adapter_opts: [retry: 0]) do
      case Stub.fetch_x509_svid(channel, %X509SVIDRequest{},
             metadata: %{"workload.spiffe.io" => "true"},
             timeout: :infinity
           ) do
        {:ok, stream} ->
          {:ok, Stream.concat(stream, disconnect_after(channel))}

        {:error, reason} ->
          GRPC.Stub.disconnect(channel)
          {:error, reason}
      end
    end
  end

  # Releases the channel once the stream is exhausted; yields nothing.
  defp disconnect_after(channel) do
    Stream.resource(fn -> channel end, fn ch -> {:halt, ch} end, &GRPC.Stub.disconnect/1)
  end
end
