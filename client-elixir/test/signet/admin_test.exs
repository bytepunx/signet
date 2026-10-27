defmodule Signet.AdminTest do
  use ExUnit.Case, async: true

  alias Signet.Admin

  # A self-signed certificate is enough to exercise PEM parsing; nothing is
  # verified against it. Generated once per test module from a throwaway key.
  defp ca_pem do
    key = :public_key.generate_key({:rsa, 2048, 65537})
    %{cert: der} = :public_key.pkix_test_root_cert(~c"signet-test-ca", key: key)
    :public_key.pem_encode([{:Certificate, der, :not_encrypted}])
  end

  describe "loopback_host?/1" do
    test "matches the Go client's table" do
      assert Admin.loopback_host?("localhost")
      assert Admin.loopback_host?("127.0.0.1")
      assert Admin.loopback_host?("::1")
      refute Admin.loopback_host?("10.0.0.5")
      refute Admin.loopback_host?("signet.internal")
    end
  end

  describe "host/1" do
    test "splits host:port and bracketed IPv6" do
      assert Admin.host("localhost:8444") == "localhost"
      assert Admin.host("[::1]:8444") == "::1"
      assert Admin.host("::1") == "::1"
      assert Admin.host("signet.internal") == "signet.internal"
    end
  end

  describe "transport/2" do
    test "loopback defaults to plaintext" do
      assert {:ok, :plaintext} = Admin.transport("localhost:8444")
      assert {:ok, :plaintext} = Admin.transport("[::1]:8444")
    end

    test "non-loopback requires TLS with the system trust store and hostname verification" do
      assert {:ok, {:tls, opts}} = Admin.transport("signet.internal:8444")
      assert opts[:verify] == :verify_peer
      assert opts[:server_name_indication] == ~c"signet.internal"
      assert Keyword.has_key?(opts, :customize_hostname_check)
      assert is_list(opts[:cacerts])
    end

    test "an IP literal gets TLS without SNI" do
      assert {:ok, {:tls, opts}} = Admin.transport("10.0.0.5:8444")
      refute Keyword.has_key?(opts, :server_name_indication)
    end

    test "force_tls upgrades a loopback address" do
      assert {:ok, {:tls, _}} = Admin.transport("localhost:8444", force_tls: true)
    end

    test "ca_pem pins the trust store and implies TLS" do
      assert {:ok, {:tls, opts}} = Admin.transport("localhost:8444", ca_pem: ca_pem())
      assert [der] = opts[:cacerts]
      assert is_binary(der)
    end

    test "an invalid CA bundle is rejected" do
      assert {:error, "no PEM certificates found in provided CA bundle"} =
               Admin.transport("signet.internal:8444", ca_pem: "not a cert")
    end

    test "plaintext overrides a non-loopback address" do
      assert {:ok, :plaintext} = Admin.transport("signet.internal:8444", plaintext: true)
      assert {:ok, :plaintext} = Admin.transport("localhost:8444", plaintext: true)
    end

    test "plaintext and force_tls are mutually exclusive" do
      assert {:error, "force_tls and plaintext are mutually exclusive"} =
               Admin.transport("signet.internal:8444", plaintext: true, force_tls: true)
    end

    test "plaintext and ca_pem are mutually exclusive" do
      assert {:error, "plaintext and ca_pem are mutually exclusive"} =
               Admin.transport("signet.internal:8444", plaintext: true, ca_pem: "some pem")
    end
  end

  describe "connect/3" do
    test "rejects an empty token before connecting" do
      assert {:error, "token must not be empty"} = Admin.connect("localhost:8444", "  ")
    end

    test "rejects conflicting options before connecting" do
      assert {:error, "force_tls and plaintext are mutually exclusive"} =
               Admin.connect("localhost:8444", "token", plaintext: true, force_tls: true)
    end

    test "reports the adapter's failure when nothing listens" do
      # Port 1 (tcpmux) is never served on a developer or CI loopback; the
      # retries are turned off so the refusal surfaces at once.
      assert {:error, _reason} =
               Admin.connect("127.0.0.1:1", "token", adapter_opts: [retry: 0, await_timeout: 1_000])
    end
  end
end
