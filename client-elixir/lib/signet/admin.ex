defmodule Signet.Admin do
  @moduledoc """
  Connects to signet's operator-facing admin listener (`AdminService` and
  `GitOpsService`) with a bearer token, mirroring the Go client's `DialAdmin`.

  Transport security is chosen from the address:

    * Loopback addresses (`localhost`, `127.0.0.1`, `::1`, the documented
      `kubectl port-forward` workflow) default to plaintext.
    * Every other address is upgraded to TLS automatically, verified against
      the operating system's trust store, or against `:ca_pem` when given.

  Two options override that heuristic in opposite directions. `force_tls: true`
  requests TLS even for a loopback address. `plaintext: true` forces insecure
  transport even for a non-loopback address, which is what dialing an
  in-cluster admin listener by its cluster-DNS name needs: that listener is
  plaintext behind the bearer token, not TLS-terminated, and without the
  override the heuristic would pick TLS and the handshake would fail at once.
  Per-RPC bearer-token authentication is unaffected either way.

  `force_tls` and `plaintext` are mutually exclusive with each other, and
  `plaintext` is mutually exclusive with a non-empty `:ca_pem`; `connect/3`
  returns an error for either combination, and for an empty token, before
  opening any connection.

      {:ok, channel} = Signet.Admin.connect("localhost:8444", token)
      {:ok, status} = Admin.V1.AdminService.Stub.status(channel, %Admin.V1.StatusRequest{})
      GRPC.Stub.disconnect(channel)
  """

  @typedoc """
  Options for `connect/3` and `transport/2`.

    * `:ca_pem` - a PEM bundle whose certificates replace the system trust store
    * `:force_tls` - use TLS even for a loopback address
    * `:plaintext` - use plaintext even for a non-loopback address
    * `:adapter_opts` - passed through to the gRPC adapter (connection retries,
      `await_timeout`), for callers that need to tune connection establishment
  """
  @type option ::
          {:ca_pem, binary()}
          | {:force_tls, boolean()}
          | {:plaintext, boolean()}
          | {:adapter_opts, keyword()}

  @typedoc "The transport `transport/2` chose: plaintext, or TLS with Erlang `ssl` options."
  @type transport :: :plaintext | {:tls, keyword()}

  @doc """
  Opens a gRPC channel to signet's admin listener at `addr` (`host:port`),
  attaching `authorization: Bearer <token>` to every RPC.

  Returns `{:ok, channel}` for use with `Admin.V1.AdminService.Stub` and
  `Admin.V1.GitOpsService.Stub`, or `{:error, reason}`: a string for an
  invalid argument combination, otherwise the adapter's connection failure.
  """
  @spec connect(String.t(), String.t(), [option()]) :: {:ok, GRPC.Channel.t()} | {:error, term()}
  def connect(addr, token, opts \\ []) when is_binary(addr) and is_binary(token) do
    token = String.trim(token)

    with :ok <- check_token(token),
         {:ok, transport} <- transport(addr, opts) do
      GRPC.Stub.connect(addr, connect_opts(transport, token, opts))
    end
  end

  @doc """
  Chooses the transport for `addr` from the heuristic and overrides described
  in the module documentation, without connecting.

  Returns `{:ok, :plaintext}`, `{:ok, {:tls, ssl_opts}}` with the Erlang `ssl`
  options the connection will use, or `{:error, message}`.
  """
  @spec transport(String.t(), [option()]) :: {:ok, transport()} | {:error, String.t()}
  def transport(addr, opts \\ []) when is_binary(addr) do
    ca_pem = Keyword.get(opts, :ca_pem, "")
    force_tls = Keyword.get(opts, :force_tls, false)
    plaintext = Keyword.get(opts, :plaintext, false)
    host = host(addr)

    cond do
      plaintext and force_tls ->
        {:error, "force_tls and plaintext are mutually exclusive"}

      plaintext and ca_pem != "" ->
        {:error, "plaintext and ca_pem are mutually exclusive"}

      plaintext ->
        {:ok, :plaintext}

      force_tls or ca_pem != "" or not loopback_host?(host) ->
        with {:ok, cacerts} <- cacerts(ca_pem) do
          {:ok, {:tls, tls_opts(host, cacerts)}}
        end

      true ->
        {:ok, :plaintext}
    end
  end

  @doc "Reports whether `host` is a loopback name or address, as the Go client's `isLoopbackHost` does."
  @spec loopback_host?(String.t()) :: boolean()
  def loopback_host?("localhost"), do: true

  def loopback_host?(host) do
    case :inet.parse_address(String.to_charlist(host)) do
      {:ok, {127, _, _, _}} -> true
      {:ok, {0, 0, 0, 0, 0, 0, 0, 1}} -> true
      _ -> false
    end
  end

  @doc """
  The host part of a `host:port` address. A bracketed IPv6 literal loses its
  brackets; an address with no port, or a bare IPv6 literal, is returned whole.
  """
  @spec host(String.t()) :: String.t()
  def host("[" <> rest) do
    case String.split(rest, "]", parts: 2) do
      [host, _port] -> host
      [host] -> host
    end
  end

  def host(addr) do
    case String.split(addr, ":") do
      [host, _port] -> host
      _ -> addr
    end
  end

  defp check_token(""), do: {:error, "token must not be empty"}
  defp check_token(_token), do: :ok

  defp connect_opts(transport, token, opts) do
    base = [
      headers: [{"authorization", "Bearer " <> token}],
      adapter_opts: Keyword.get(opts, :adapter_opts, [])
    ]

    case transport do
      :plaintext -> base
      {:tls, ssl_opts} -> [{:cred, GRPC.Credential.new(ssl: ssl_opts)} | base]
    end
  end

  # With no CA bundle, the system trust store applies. :public_key.cacerts_get/0
  # raises when the OS has none; that is a deployment fault worth surfacing as is.
  defp cacerts(""), do: {:ok, :public_key.cacerts_get()}

  defp cacerts(pem) do
    case for {:Certificate, der, :not_encrypted} <- :public_key.pem_decode(pem), do: der do
      [] -> {:error, "no PEM certificates found in provided CA bundle"}
      ders -> {:ok, ders}
    end
  end

  defp tls_opts(host, cacerts) do
    [
      verify: :verify_peer,
      cacerts: cacerts,
      versions: [:"tlsv1.2", :"tlsv1.3"],
      depth: 4
    ] ++ hostname_opts(host)
  end

  # SNI and hostname verification apply to DNS names only; for an IP literal
  # the peer certificate is matched on its IP SAN by ssl itself.
  defp hostname_opts(host) do
    case :inet.parse_address(String.to_charlist(host)) do
      {:ok, _ip} ->
        []

      {:error, _} ->
        [
          server_name_indication: String.to_charlist(host),
          customize_hostname_check: [match_fun: :public_key.pkix_verify_hostname_match_fun(:https)]
        ]
    end
  end
end
