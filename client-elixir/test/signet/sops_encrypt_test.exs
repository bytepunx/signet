defmodule Signet.SopsEncryptTest do
  use ExUnit.Case, async: true

  alias Signet.AgeTestSupport, as: Support
  alias Signet.SopsEncrypt

  @begin "-----BEGIN AGE ENCRYPTED FILE-----"
  @end_marker "-----END AGE ENCRYPTED FILE-----"
  @enc ~r/^ENC\[AES256_GCM,data:[A-Za-z0-9+\/=]+,iv:[A-Za-z0-9+\/=]+,tag:[A-Za-z0-9+\/=]+,type:str\]$/

  # Picks the fields out of the fixed document without a YAML library: the
  # scalar lines by key, the age block as the lines indented by 12 spaces.
  defp parse(yaml) do
    lines = String.split(yaml, "\n")

    scalar = fn key ->
      Enum.find_value(lines, fn line ->
        case String.split(unindent(line), ": ", parts: 2) do
          [^key, value] -> value
          _ -> nil
        end
      end)
    end

    enc =
      lines
      |> Enum.filter(&String.starts_with?(&1, String.duplicate(" ", 12)))
      |> Enum.map_join("\n", &String.slice(&1, 12..-1//1))

    %{
      value: scalar.("value"),
      recipient: scalar.("recipient"),
      enc: enc <> "\n",
      lastmodified: scalar.("lastmodified"),
      mac: scalar.("mac"),
      version: scalar.("version")
    }
  end

  defp unindent(line), do: String.replace(line, ~r/^[ -]+/, "")

  defp decrypt_leaf(enc, data_key, aad) do
    [_, data, iv, tag] =
      Regex.run(~r/^ENC\[AES256_GCM,data:(.*),iv:(.*),tag:(.*),type:str\]$/, enc)

    iv = Base.decode64!(iv)
    assert byte_size(iv) == 32

    :crypto.crypto_one_time_aead(
      :aes_256_gcm,
      data_key,
      iv,
      Base.decode64!(data),
      aad,
      Base.decode64!(tag),
      false
    )
  end

  defp round_trip(value) do
    identity = Support.generate_identity()
    assert {:ok, yaml} = SopsEncrypt.encrypt_for_secret(identity.recipient, value)
    doc = parse(yaml)
    assert {:ok, data_key} = Support.decrypt(doc.enc, identity.private)
    assert byte_size(data_key) == 32
    assert decrypt_leaf(doc.value, data_key, "value:") == value

    assert decrypt_leaf(doc.mac, data_key, String.trim(doc.lastmodified, ~s("))) ==
             Base.encode16(:crypto.hash(:sha512, value))

    yaml
  end

  describe "encrypt_for_secret/2 document shape" do
    setup do
      identity = Support.generate_identity()
      {:ok, yaml} = SopsEncrypt.encrypt_for_secret(identity.recipient, "hello-world")
      %{identity: identity, yaml: yaml, doc: parse(yaml)}
    end

    test "has the Go client's keys in order and ends with a newline", %{yaml: yaml} do
      assert String.starts_with?(yaml, "value: ENC[AES256_GCM,")
      assert String.ends_with?(yaml, "\n")

      keys =
        yaml
        |> String.split("\n", trim: true)
        |> Enum.reject(&String.starts_with?(&1, String.duplicate(" ", 12)))
        |> Enum.map(&(&1 |> unindent() |> String.split(":", parts: 2) |> hd()))

      assert keys == ~w(value sops age recipient enc lastmodified mac version)
    end

    test "value and mac are sops AES256_GCM leaves", %{doc: doc} do
      assert doc.value =~ @enc
      assert doc.mac =~ @enc
    end

    test "names the recipient and embeds the armored age file", %{doc: doc, identity: identity} do
      assert doc.recipient == identity.recipient
      assert String.starts_with?(doc.enc, @begin <> "\n")
      assert String.ends_with?(doc.enc, "\n" <> @end_marker <> "\n")
    end

    test "lastmodified is a quoted RFC 3339 UTC time of now", %{doc: doc} do
      assert <<?", stamp::binary-size(20), ?">> = doc.lastmodified
      assert {:ok, at, 0} = DateTime.from_iso8601(stamp)
      assert abs(DateTime.diff(DateTime.utc_now(), at)) < 60
    end

    test "records the sops file version", %{doc: doc} do
      assert doc.version == "3.13.1"
    end
  end

  describe "encrypt_for_secret/2 decryptability" do
    test "the leaf and the MAC decrypt with the data key wrapped to the recipient" do
      round_trip("hello-world")
    end

    test "an empty value round-trips" do
      round_trip("")
    end

    test "bytes with newlines and non-ASCII round-trip" do
      round_trip("line one\nline two: value\n\xff\x00 héllo")
    end

    test "every call draws a fresh data key and fresh nonces" do
      identity = Support.generate_identity()
      {:ok, a} = SopsEncrypt.encrypt_for_secret(identity.recipient, "same")
      {:ok, b} = SopsEncrypt.encrypt_for_secret(identity.recipient, "same")
      assert parse(a).value != parse(b).value
      assert parse(a).mac != parse(b).mac
      assert parse(a).enc != parse(b).enc
    end
  end

  describe "encrypt_for_secret/2 errors" do
    test "rejects an empty public key" do
      assert {:error, reason} = SopsEncrypt.encrypt_for_secret("", "hello-world")
      assert reason =~ "recipient"
    end

    test "rejects a malformed public key" do
      assert {:error, reason} = SopsEncrypt.encrypt_for_secret("not-a-real-age-key", "x")
      assert reason =~ "recipient"
    end

    test "rejects an identity passed as the recipient" do
      identity = Support.generate_identity()
      assert {:error, reason} = SopsEncrypt.encrypt_for_secret(identity.identity, "x")
      assert reason =~ "recipient"
    end
  end
end
