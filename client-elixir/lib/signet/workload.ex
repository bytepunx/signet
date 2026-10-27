defmodule Signet.Workload do
  @moduledoc """
  Connects to signet's workload listener (`SecretsService`) over SPIFFE mTLS,
  mirroring the Go client's `DialWorkload`.

      {:ok, channel, source} =
        Signet.Workload.connect("signet.internal:8443",
          "unix:///run/spire/sockets/agent.sock", "example.org")

      {:ok, resp} =
        Signet.V1.SecretsService.Stub.get_secret(channel,
          %Signet.V1.GetSecretRequest{namespace: "default", service: "example", name: "api-key"})

      Signet.Workload.disconnect(channel, source)

  The X.509-SVID and trust bundle come from the SPIRE Workload API at the
  socket path, through a `Signet.Spiffe.X509Source` that retries the
  identity-registration race automatically (see that module) and keeps
  following rotations. The connection presents the SVID current when it is
  opened, and verifies the server's certificate against the trust bundle of
  `trust_domain`, then checks that the server's SPIFFE ID is a member of that
  trust domain (`Signet.Spiffe.Authorize`). No hostname verification takes
  place, and no server name is sent: an X.509-SVID carries only a SPIFFE URI.

  The channel also serves the subset of `GitOpsService` signet accepts from a
  workload's own identity (`SyncBundle`, `PatchServiceConfig`,
  `GetSOPSPublicKey`), through `Admin.V1.GitOpsService.Stub`, without an admin
  token.
  """

  alias Signet.Spiffe.{Authorize, TrustDomain, X509Context, X509Source}

  @typedoc """
  Options for `connect/4`.

    * `:api` - the `Signet.Spiffe.WorkloadAPI` implementation, for tests
    * `:adapter_opts` - passed through to the gRPC adapter
  """
  @type option :: {:api, module()} | {:adapter_opts, keyword()}

  @doc """
  Opens a gRPC channel to signet at `addr` with an SVID from the Workload API
  at `socket_path`, authorising the server as a member of `trust_domain`.

  Returns `{:ok, channel, source}`; pass both to `disconnect/2` when done, so
  the source's Workload API stream is released with the channel.
  """
  @spec connect(String.t(), String.t(), String.t(), [option()]) ::
          {:ok, GRPC.Channel.t(), pid()} | {:error, term()}
  def connect(addr, socket_path, trust_domain, opts \\ []) do
    with {:ok, trust_domain} <- TrustDomain.parse(trust_domain),
         {:ok, source} <- start_source(socket_path, opts),
         {:ok, ssl_opts} <- tls_opts(X509Source.current(source), trust_domain),
         {:ok, channel} <- dial(addr, ssl_opts, opts, source) do
      {:ok, channel, source}
    end
  end

  @doc "Closes the channel and stops the source."
  @spec disconnect(GRPC.Channel.t(), pid()) :: :ok
  def disconnect(channel, source) do
    GRPC.Stub.disconnect(channel)
    X509Source.stop(source)
    :ok
  end

  @doc """
  The Erlang `ssl` options for mutual TLS from `context` to a server in
  `trust_domain`: the SVID as the client certificate, the trust domain's bundle
  as the only trusted roots, peer verification with `Signet.Spiffe.Authorize`,
  and no server name indication or hostname check.
  """
  @spec tls_opts(X509Context.t(), TrustDomain.t()) :: {:ok, keyword()} | {:error, String.t()}
  def tls_opts(%X509Context{} = context, trust_domain) do
    case X509Context.bundle_for(context, trust_domain) do
      {:ok, bundle} ->
        {:ok,
         [
           cert: context.cert_chain,
           key: context.key,
           cacerts: bundle,
           verify: :verify_peer,
           verify_fun: Authorize.member_of(trust_domain),
           server_name_indication: :disable,
           versions: [:"tlsv1.2", :"tlsv1.3"]
         ]}

      :error ->
        {:error,
         "no X.509 bundle available for trust domain #{inspect(trust_domain)} (is the Workload API federated with it?)"}
    end
  end

  defp start_source(socket_path, opts) do
    source_opts = [socket_path: socket_path] ++ Keyword.take(opts, [:api])

    case X509Source.start_link(source_opts) do
      {:ok, source} ->
        {:ok, source}

      {:error, reason} ->
        {:error, "connect to SPIFFE workload API at #{socket_path}: #{inspect(reason)}"}
    end
  end

  defp dial(addr, ssl_opts, opts, source) do
    connect_opts = [
      cred: GRPC.Credential.new(ssl: ssl_opts),
      adapter_opts: Keyword.get(opts, :adapter_opts, [])
    ]

    case GRPC.Stub.connect(addr, connect_opts) do
      {:ok, channel} ->
        {:ok, channel}

      {:error, reason} ->
        X509Source.stop(source)
        {:error, reason}
    end
  end
end
