defmodule Signet.Spiffe.X509SourceTest do
  use ExUnit.Case, async: true

  alias Signet.FakeWorkloadAPI, as: Fake
  alias Signet.Spiffe.{X509Context, X509Source}
  alias Signet.TestCerts
  alias Spiffe.Workload.{X509SVID, X509SVIDResponse}

  defp response(id) do
    %{cert: cert, key: key, cacerts: [root]} =
      TestCerts.chain([{:uniformResourceIdentifier, String.to_charlist(id)}])

    %X509SVIDResponse{
      svids: [
        %X509SVID{
          spiffe_id: id,
          x509_svid: cert,
          x509_svid_key: TestCerts.pkcs8(key),
          bundle: root
        }
      ]
    }
  end

  defp start(script) do
    socket = "unix:///tmp/#{System.unique_integer([:positive])}.sock"
    :ok = Fake.start(socket, script, self())
    test = self()
    sleep = fn ms -> send(test, {:slept, ms}) end
    {socket, X509Source.start_link(socket_path: socket, api: Fake, sleep: sleep)}
  end

  defp slept do
    receive do
      {:slept, ms} -> [ms | slept()]
    after
      0 -> []
    end
  end

  test "holds the first identity without sleeping when it is issued at once" do
    {socket, {:ok, source}} = start([{:ok, [{:ok, response("spiffe://example.org/a")}]}])
    assert %X509Context{spiffe_id: "spiffe://example.org/a"} = X509Source.current(source)
    assert_received {:fetch, ^socket}
    assert slept() == []
    X509Source.stop(source)
  end

  test "retries only no-identity-issued, on the 1s 2s 4s 8s schedule" do
    denied = {:ok, [{:error, Fake.no_identity_issued()}]}

    {_socket, {:ok, source}} =
      start([denied, denied, denied, {:ok, [{:ok, response("spiffe://example.org/b")}]}])

    assert %X509Context{spiffe_id: "spiffe://example.org/b"} = X509Source.current(source)
    assert slept() == [1_000, 2_000, 4_000]
    X509Source.stop(source)
  end

  test "gives up after five attempts naming the last failure" do
    denied = {:ok, [{:error, Fake.no_identity_issued()}]}
    {_socket, result} = start(List.duplicate(denied, 5))
    assert {:error, reason} = result
    assert reason =~ "no identity issued after 5 attempts"
    assert slept() == [1_000, 2_000, 4_000, 8_000]
  end

  test "returns any other failure immediately, unretried" do
    unavailable = GRPC.RPCError.exception(GRPC.Status.unavailable(), "socket missing")

    {_socket, result} =
      start([{:error, unavailable}, {:ok, [{:ok, response("spiffe://example.org/c")}]}])

    assert {:error, ^unavailable} = result
    assert slept() == []
  end

  test "follows rotations delivered on the stream" do
    first = response("spiffe://example.org/v1")
    second = response("spiffe://example.org/v2")
    {_socket, {:ok, source}} = start([{:ok, [{:ok, first}, {:ok, second}]}, {:ok, []}])

    assert eventually(fn -> X509Source.current(source).spiffe_id == "spiffe://example.org/v2" end)
    X509Source.stop(source)
  end

  defp eventually(check, tries \\ 50) do
    cond do
      check.() -> true
      tries == 0 -> false
      true -> Process.sleep(10) && eventually(check, tries - 1)
    end
  end
end
