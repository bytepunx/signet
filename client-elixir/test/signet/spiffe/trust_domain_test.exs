defmodule Signet.Spiffe.TrustDomainTest do
  use ExUnit.Case, async: true

  alias Signet.Spiffe.TrustDomain

  test "accepts valid names and SPIFFE IDs" do
    assert {:ok, "example.org"} = TrustDomain.parse("example.org")

    assert {:ok, "cluster.local"} =
             TrustDomain.parse("spiffe://cluster.local/ns/default/sa/signet")

    assert {:ok, "cluster.local"} = TrustDomain.parse("spiffe://cluster.local")
  end

  test "rejects empty, uppercase, and overlong names" do
    assert {:error, "trust domain is missing"} = TrustDomain.parse("")
    assert {:error, _} = TrustDomain.parse("Example.org")
    assert {:error, _} = TrustDomain.parse("exam ple")
    assert {:error, _} = TrustDomain.parse(String.duplicate("a", 256))
  end

  test "of_spiffe_id rejects non-SPIFFE URIs" do
    assert {:error, _} = TrustDomain.of_spiffe_id("https://example.org")
    assert {:ok, "example.org"} = TrustDomain.of_spiffe_id("spiffe://example.org/x")
  end
end
