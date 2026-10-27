defmodule Signet.WorkloadTest do
  use ExUnit.Case, async: true

  alias Signet.FakeWorkloadAPI, as: Fake
  alias Signet.Spiffe.X509Context
  alias Signet.TestCerts
  alias Signet.Workload
  alias Spiffe.Workload.{X509SVID, X509SVIDResponse}

  @client_id ~c"spiffe://example.org/ns/default/sa/app"
  @server_id ~c"spiffe://example.org/ns/default/sa/signet"

  defp context(trust_domain_root) do
    %{cert: cert, key: key} = TestCerts.chain([{:uniformResourceIdentifier, @client_id}])

    {:ok, ctx} =
      X509Context.from_response(%X509SVIDResponse{
        svids: [
          %X509SVID{
            spiffe_id: to_string(@client_id),
            x509_svid: cert,
            x509_svid_key: TestCerts.pkcs8(key),
            bundle: trust_domain_root
          }
        ]
      })

    ctx
  end

  test "tls_opts presents the SVID, trusts only the bundle, and disables hostname checks" do
    %{cacerts: [root]} = TestCerts.chain([{:uniformResourceIdentifier, @server_id}])
    assert {:ok, opts} = Workload.tls_opts(context(root), "example.org")
    assert [_leaf] = opts[:cert]
    assert {:PrivateKeyInfo, _} = opts[:key]
    assert opts[:cacerts] == [root]
    assert opts[:verify] == :verify_peer
    assert {_fun, "example.org"} = opts[:verify_fun]
    assert opts[:server_name_indication] == :disable
  end

  test "tls_opts fails when the workload has no bundle for the trust domain" do
    %{cacerts: [root]} = TestCerts.chain([])
    assert {:error, reason} = Workload.tls_opts(context(root), "other.org")
    assert reason =~ ~s(no X.509 bundle available for trust domain "other.org")
  end

  test "connect rejects a malformed trust domain before touching the Workload API" do
    assert {:error, reason} =
             Workload.connect("signet:8443", "unix:///nowhere.sock", "Bad Domain")

    assert reason =~ "trust domain"
  end

  test "connect reports a Workload API failure with the socket path" do
    socket = "unix:///tmp/#{System.unique_integer([:positive])}.sock"
    unavailable = GRPC.RPCError.exception(GRPC.Status.unavailable(), "no such socket")
    :ok = Fake.start(socket, [{:error, unavailable}], self())
    assert {:error, reason} = Workload.connect("signet:8443", socket, "example.org", api: Fake)
    assert reason =~ "connect to SPIFFE workload API at #{socket}"
  end

  # A loopback TLS handshake against an Erlang ssl listener presenting a
  # SPIFFE-only server certificate: the exact option set connect/4 hands to
  # the gRPC adapter, exercised end to end without signet or a network.
  describe "handshake" do
    setup do
      server = TestCerts.chain([{:uniformResourceIdentifier, @server_id}])
      [root] = server.cacerts
      ctx = context(root)
      {:ok, [client_root]} = X509Context.bundle_for(ctx, "example.org")
      {:ok, port: listen(server, client_root), ctx: ctx}
    end

    # Accepts one connection and reports whether its handshake completed.
    defp listen(server, client_root) do
      {:ok, listener} =
        :ssl.listen(0,
          cert: server.cert,
          key: server.key,
          cacerts: [client_root],
          verify: :verify_peer,
          fail_if_no_peer_cert: true,
          ip: {127, 0, 0, 1}
        )

      {:ok, {_ip, port}} = :ssl.sockname(listener)

      Task.start_link(fn ->
        {:ok, transport} = :ssl.transport_accept(listener)
        _ = :ssl.handshake(transport, 2_000)
      end)

      port
    end

    test "succeeds against a server in the expected trust domain", %{port: port, ctx: ctx} do
      {:ok, opts} = Workload.tls_opts(ctx, "example.org")
      assert {:ok, socket} = :ssl.connect(~c"127.0.0.1", port, opts, 2_000)
      :ssl.close(socket)
    end

    test "fails against a server in another trust domain", %{port: port, ctx: ctx} do
      {:ok, opts} = Workload.tls_opts(ctx, "example.org")
      opts = Keyword.put(opts, :verify_fun, Signet.Spiffe.Authorize.member_of("other.org"))
      assert {:error, _} = :ssl.connect(~c"127.0.0.1", port, opts, 2_000)
    end
  end
end
