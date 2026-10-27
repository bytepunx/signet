defmodule Signet.AgeTestSupport do
  @moduledoc """
  Generates age identities and decrypts what `Signet.Age` produces, so
  behaviour tests round-trip without an age binary. The client itself never
  decrypts, which is why this lives in test support.

  The decryptor is deliberately strict about the file's shape: one X25519
  stanza, a header MAC that verifies, and STREAM chunks that open in order.
  """

  alias Signet.Age.Bech32

  @version_line "age-encryption.org/v1"
  @x25519_info "age-encryption.org/v1/X25519"
  @chunk_size 65_536
  @tag_size 16

  @doc "A fresh X25519 identity, with the strings age prints for it."
  @spec generate_identity() :: %{
          identity: String.t(),
          recipient: String.t(),
          private: binary(),
          public: binary()
        }
  def generate_identity do
    {public, private} = :crypto.generate_key(:ecdh, :x25519)

    %{
      identity: String.upcase(Bech32.encode("age-secret-key-", private)),
      recipient: Bech32.encode("age", public),
      private: private,
      public: public
    }
  end

  @doc "Decrypts an armored age file with the identity's private key bytes."
  @spec decrypt(String.t(), binary()) :: {:ok, binary()} | {:error, term()}
  def decrypt(armored, private) when is_binary(armored) and is_binary(private) do
    with {:ok, file} <- dearmor(armored),
         {:ok, header, mac, body} <- split_header(file),
         {:ok, file_key} <- unwrap_file_key(header, private),
         :ok <- check_mac(header, mac, file_key),
         {:ok, nonce, payload} <- split_payload(body) do
      open_chunks(hkdf(file_key, nonce, "payload"), payload, 0)
    end
  end

  defp dearmor(armored) do
    case String.split(armored, "\n") do
      ["-----BEGIN AGE ENCRYPTED FILE-----" | rest] ->
        case Enum.reverse(rest) do
          ["", "-----END AGE ENCRYPTED FILE-----" | lines] ->
            lines |> Enum.reverse() |> Enum.join() |> Base.decode64() |> or_error(:bad_base64)

          _ ->
            {:error, :missing_footer}
        end

      _ ->
        {:error, :missing_header}
    end
  end

  defp split_header(<<@version_line, "\n", rest::binary>>) do
    with [stanzas, after_mac] <- :binary.split(rest, "\n--- "),
         [mac_b64, body] <- :binary.split(after_mac, "\n"),
         {:ok, mac} <- Base.decode64(mac_b64, padding: false) do
      {:ok, @version_line <> "\n" <> stanzas <> "\n---", mac, body}
    else
      _ -> {:error, :malformed_header}
    end
  end

  defp split_header(_), do: {:error, :bad_version_line}

  defp unwrap_file_key(header, private) do
    with [@version_line, "-> X25519 " <> eph_b64, body_b64, "---"] <- String.split(header, "\n"),
         {:ok, <<eph_pub::binary-size(32)>>} <- Base.decode64(eph_b64, padding: false),
         {:ok, <<cipher::binary-size(16), tag::binary-size(@tag_size)>>} <-
           Base.decode64(body_b64, padding: false) do
      {public, _} = :crypto.generate_key(:ecdh, :x25519, private)
      shared = :crypto.compute_key(:ecdh, eph_pub, private, :x25519)
      wrap_key = hkdf(shared, eph_pub <> public, @x25519_info)

      :chacha20_poly1305
      |> :crypto.crypto_one_time_aead(wrap_key, <<0::96>>, cipher, "", tag, false)
      |> or_error(:wrong_identity)
    else
      _ -> {:error, :malformed_stanza}
    end
  end

  defp check_mac(header, mac, file_key) do
    if :crypto.mac(:hmac, :sha256, hkdf(file_key, "", "header"), header) == mac,
      do: :ok,
      else: {:error, :bad_header_mac}
  end

  defp split_payload(<<nonce::binary-size(16), payload::binary>>), do: {:ok, nonce, payload}
  defp split_payload(_), do: {:error, :short_payload}

  defp open_chunks(key, payload, counter) when byte_size(payload) <= @chunk_size + @tag_size do
    open(key, counter, 1, payload)
  end

  defp open_chunks(key, <<chunk::binary-size(@chunk_size + @tag_size), rest::binary>>, counter) do
    with {:ok, plain} <- open(key, counter, 0, chunk),
         {:ok, more} <- open_chunks(key, rest, counter + 1) do
      {:ok, plain <> more}
    end
  end

  defp open(key, counter, last, chunk) when byte_size(chunk) >= @tag_size do
    size = byte_size(chunk) - @tag_size
    <<cipher::binary-size(size), tag::binary>> = chunk

    :chacha20_poly1305
    |> :crypto.crypto_one_time_aead(key, <<counter::88, last::8>>, cipher, "", tag, false)
    |> or_error(:bad_chunk)
  end

  defp open(_key, _counter, _last, _chunk), do: {:error, :short_chunk}

  defp hkdf(ikm, salt, info) do
    prk = :crypto.mac(:hmac, :sha256, salt, ikm)
    :crypto.mac(:hmac, :sha256, prk, info <> <<1>>)
  end

  defp or_error({:ok, value}, _reason), do: {:ok, value}
  defp or_error(:error, reason), do: {:error, reason}
  defp or_error(value, _reason) when is_binary(value), do: {:ok, value}
end
