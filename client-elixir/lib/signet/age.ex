defmodule Signet.Age do
  @moduledoc """
  Encrypts to an age X25519 recipient, producing the ASCII-armored file
  `filippo.io/age`'s `age.Encrypt` and `armor.NewWriter` produce.

  This is the slice of the age v1 format (`https://age-encryption.org/v1`)
  that wrapping a SOPS data key needs: one X25519 recipient stanza, the
  header MAC, and the STREAM payload, all on OTP `:crypto` (X25519,
  HKDF-SHA256 over HMAC, ChaCha20-Poly1305). No hex package implements
  age; see `design/tech/age-elixir.md`.

      {:ok, armored} = Signet.Age.encrypt("age1...", data_key)
  """

  alias Signet.Age.Bech32

  @version_line "age-encryption.org/v1"
  @x25519_info "age-encryption.org/v1/X25519"
  @file_key_size 16
  @nonce_size 16
  @chunk_size 65_536

  @doc "Parses an `age1...` recipient into its 32-byte X25519 public key."
  @spec parse_recipient(String.t()) :: {:ok, <<_::256>>} | {:error, String.t()}
  def parse_recipient(""), do: {:error, "parse age recipient: recipient must not be empty"}

  def parse_recipient(recipient) when is_binary(recipient) do
    case Bech32.decode(recipient) do
      {:ok, {"age", <<key::binary-size(32)>>}} ->
        {:ok, key}

      {:ok, {"age", key}} ->
        {:error,
         "parse age recipient #{inspect(recipient)}: key is #{byte_size(key)} bytes, want 32"}

      {:ok, {hrp, _}} ->
        {:error,
         "parse age recipient #{inspect(recipient)}: unknown recipient type #{inspect(hrp)}"}

      :error ->
        {:error, "parse age recipient #{inspect(recipient)}: not a bech32 string"}
    end
  end

  @doc "Encrypts `plaintext` to `recipient` as an ASCII-armored age file."
  @spec encrypt(String.t(), binary()) :: {:ok, String.t()} | {:error, String.t()}
  def encrypt(recipient, plaintext) when is_binary(recipient) and is_binary(plaintext) do
    with {:ok, public} <- parse_recipient(recipient),
         {:ok, stanza, file_key} <- wrap_file_key(public) do
      header = @version_line <> "\n" <> stanza <> "---"
      mac = :crypto.mac(:hmac, :sha256, hkdf(file_key, "", "header"), header)
      nonce = :crypto.strong_rand_bytes(@nonce_size)
      payload = encrypt_chunks(hkdf(file_key, nonce, "payload"), plaintext, 0)
      {:ok, armor(header <> " " <> b64(mac) <> "\n" <> nonce <> payload)}
    end
  end

  # The X25519 recipient stanza: an ephemeral share and the file key wrapped
  # under HKDF(shared secret, ephemeral || recipient) with a zero nonce.
  defp wrap_file_key(public) do
    file_key = :crypto.strong_rand_bytes(@file_key_size)
    {eph_pub, eph_priv} = :crypto.generate_key(:ecdh, :x25519)

    case :crypto.compute_key(:ecdh, public, eph_priv, :x25519) do
      <<0::256>> ->
        {:error, "wrap age file key: recipient is a low-order point"}

      shared ->
        wrap_key = hkdf(shared, eph_pub <> public, @x25519_info)

        {cipher, tag} =
          :crypto.crypto_one_time_aead(
            :chacha20_poly1305,
            wrap_key,
            <<0::96>>,
            file_key,
            "",
            true
          )

        stanza = "-> X25519 " <> b64(eph_pub) <> "\n" <> b64(cipher <> tag) <> "\n"
        {:ok, stanza, file_key}
    end
  end

  # STREAM: 64 KiB chunks under ChaCha20-Poly1305, the nonce an 11-byte
  # big-endian counter and a final-chunk flag. The last chunk is whatever
  # remains, a full chunk when the plaintext is a multiple of the size, and
  # empty only for an empty plaintext.
  defp encrypt_chunks(key, plaintext, counter) when byte_size(plaintext) <= @chunk_size do
    seal(key, counter, 1, plaintext)
  end

  defp encrypt_chunks(key, <<chunk::binary-size(@chunk_size), rest::binary>>, counter) do
    seal(key, counter, 0, chunk) <> encrypt_chunks(key, rest, counter + 1)
  end

  defp seal(key, counter, last, chunk) do
    nonce = <<counter::88, last::8>>
    {cipher, tag} = :crypto.crypto_one_time_aead(:chacha20_poly1305, key, nonce, chunk, "", true)
    cipher <> tag
  end

  # HKDF-SHA256 (RFC 5869), one 32-byte block, as age uses it everywhere.
  defp hkdf(ikm, salt, info) do
    prk = :crypto.mac(:hmac, :sha256, salt, ikm)
    :crypto.mac(:hmac, :sha256, prk, info <> <<1>>)
  end

  defp b64(bytes), do: Base.encode64(bytes, padding: false)

  defp armor(file) do
    "-----BEGIN AGE ENCRYPTED FILE-----\n" <>
      Enum.map_join(wrap_lines(Base.encode64(file)), &(&1 <> "\n")) <>
      "-----END AGE ENCRYPTED FILE-----\n"
  end

  defp wrap_lines(<<line::binary-size(64), rest::binary>>), do: [line | wrap_lines(rest)]
  defp wrap_lines(""), do: []
  defp wrap_lines(rest), do: [rest]
end
