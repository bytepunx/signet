defmodule Signet.Spiffe.AuthorizeTest do
  use ExUnit.Case, async: true

  alias Signet.Spiffe.Authorize
  alias Signet.TestCerts

  defp peer(names), do: TestCerts.decode(TestCerts.chain(names).cert)

  test "accepts a peer whose SPIFFE ID is in the expected trust domain" do
    cert = peer([{:uniformResourceIdentifier, ~c"spiffe://example.org/ns/default/sa/signet"}])
    assert :ok = Authorize.authorize_peer(cert, "example.org")
    {fun, state} = Authorize.member_of("example.org")
    assert {:valid, "example.org"} = fun.(cert, :valid_peer, state)
  end

  test "ignores DNS and IP names beside the SPIFFE ID" do
    cert =
      peer([
        {:dNSName, ~c"signet.internal"},
        {:uniformResourceIdentifier, ~c"spiffe://example.org/ns/default/sa/signet"},
        {:iPAddress, [10, 0, 0, 1]}
      ])

    assert :ok = Authorize.authorize_peer(cert, "example.org")
  end

  test "rejects a SPIFFE ID from another trust domain, naming both" do
    cert = peer([{:uniformResourceIdentifier, ~c"spiffe://evil.example/ns/default/sa/signet"}])
    assert {:error, reason} = Authorize.authorize_peer(cert, "example.org")
    assert reason =~ ~s(trust domain "evil.example")
    assert reason =~ ~s(expected trust domain "example.org")
    {fun, state} = Authorize.member_of("example.org")
    assert {:fail, ^reason} = fun.(cert, :valid_peer, state)
  end

  test "rejects a certificate with no SPIFFE ID" do
    cert = peer([{:dNSName, ~c"signet.internal"}])
    assert {:error, reason} = Authorize.authorize_peer(cert, "example.org")
    assert reason =~ "did not present a SPIFFE ID"
  end

  test "rejects a certificate with several SPIFFE IDs" do
    cert =
      peer([
        {:uniformResourceIdentifier, ~c"spiffe://example.org/a"},
        {:uniformResourceIdentifier, ~c"spiffe://example.org/b"}
      ])

    assert {:error, reason} = Authorize.authorize_peer(cert, "example.org")
    assert reason =~ "2 SPIFFE IDs"
  end

  test "passes chain validation failures and intermediate certificates through" do
    {fun, state} = Authorize.member_of("example.org")
    assert {:fail, :cert_expired} = fun.(:cert, {:bad_cert, :cert_expired}, state)
    assert {:valid, ^state} = fun.(:cert, :valid, state)
    assert {:unknown, ^state} = fun.(:cert, {:extension, :ext}, state)
  end
end
