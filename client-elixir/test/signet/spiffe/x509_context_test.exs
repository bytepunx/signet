defmodule Signet.Spiffe.X509ContextTest do
  use ExUnit.Case, async: true

  alias Signet.Spiffe.X509Context
  alias Signet.TestCerts
  alias Spiffe.Workload.{X509SVID, X509SVIDResponse}

  defp response(names, federated \\ %{}) do
    %{cert: cert, key: key, cacerts: [root]} = TestCerts.chain(names)

    %X509SVIDResponse{
      svids: [
        %X509SVID{
          spiffe_id: "spiffe://example.org/ns/default/sa/app",
          x509_svid: cert <> root,
          x509_svid_key: TestCerts.pkcs8(key),
          bundle: root
        }
      ],
      federated_bundles: federated
    }
  end

  test "decodes the default SVID, splitting the chain and keying bundles by trust domain" do
    resp = response([{:uniformResourceIdentifier, ~c"spiffe://example.org/ns/default/sa/app"}])
    assert {:ok, %X509Context{} = ctx} = X509Context.from_response(resp)
    assert ctx.spiffe_id == "spiffe://example.org/ns/default/sa/app"
    assert ctx.trust_domain == "example.org"
    assert [_leaf, _root] = ctx.cert_chain
    assert {:PrivateKeyInfo, der} = ctx.key
    assert is_binary(der)
    assert {:ok, [_root]} = X509Context.bundle_for(ctx, "example.org")
    assert :error = X509Context.bundle_for(ctx, "other.org")
  end

  test "keeps federated bundles under their own trust domains" do
    %{cacerts: [other_root]} =
      TestCerts.chain([{:uniformResourceIdentifier, ~c"spiffe://other.org/x"}])

    resp =
      response([{:uniformResourceIdentifier, ~c"spiffe://example.org/x"}], %{
        "spiffe://other.org" => other_root
      })

    assert {:ok, ctx} = X509Context.from_response(resp)
    assert {:ok, [^other_root]} = X509Context.bundle_for(ctx, "other.org")
  end

  test "rejects a response with no SVIDs and malformed DER" do
    assert {:error, "Workload API returned no SVIDs"} =
             X509Context.from_response(%X509SVIDResponse{svids: []})

    assert {:error, "malformed DER in x"} = X509Context.split_der(<<0x30, 0x05, 1, 2>>, "x")
    assert {:error, "malformed DER in x"} = X509Context.split_der("junk", "x")
    assert {:error, "empty DER"} = X509Context.split_der("", "x")
  end

  test "splits long-form DER lengths" do
    body = :binary.copy(<<0>>, 300)
    cert = <<0x30, 0x82, 300::16>> <> body
    assert {:ok, [^cert, ^cert]} = X509Context.split_der(cert <> cert, "x")
  end
end
