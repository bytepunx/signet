defmodule Signet.SopsEncrypt do
  @moduledoc """
  Produces the SOPS-encrypted YAML document a workload submits through
  `GitOpsService.SyncBundle`, mirroring the Go client's `EncryptForSecret`.

      {:ok, public_key} = ... # GetSOPSPublicKey
      {:ok, yaml} = Signet.SopsEncrypt.encrypt_for_secret(public_key, "s3cr3t")

  `SyncBundle` stores secret file content as-is and never encrypts on the
  server's behalf: only SOPS ciphertext leaves the workload. A plaintext value
  submitted through it fails to decrypt and is silently never stored, so this
  is the way to produce a value signet will persist. The result is a YAML
  document with a single top-level `value` key, SOPS-encrypted to one age
  recipient, the shape signetd's gitops sync path and `signet secret set`
  both expect.

  The narrow slice of the SOPS file format a single-leaf, single-recipient
  document needs is implemented here by hand, as every other signet client
  does: AES-256-GCM for the leaf and the MAC, a SHA-512 MAC, and the data key
  wrapped by `Signet.Age`. No SOPS library exists for the BEAM, and the Go
  client made the same choice to keep `github.com/getsops/sops/v3`'s every
  backend out of a client library. The algorithm follows the Go
  implementation line for line, which was verified against sops's own
  `Tree.Encrypt` and `aes.Cipher.Encrypt`; the integration test under
  `tests/integration/` confirms the real sops binary decrypts the output.
  """

  alias Signet.Age

  # Recorded in sops.version. The sops CLI only requires it to be non-empty
  # to decrypt; it does not gate decryption on its own build version.
  @sops_file_version "3.13.1"

  # sops's own non-standard AES-GCM nonce size (aes.Cipher's nonceSize), not
  # the usual 12 bytes.
  @gcm_nonce_size 32
  @gcm_tag_size 16
  @data_key_size 32

  @doc """
  Encrypts `value` to the age recipient `public_key` as a SOPS YAML document
  with a single `value` leaf. Returns `{:ok, yaml}` or `{:error, reason}`.
  """
  @spec encrypt_for_secret(String.t(), binary()) :: {:ok, String.t()} | {:error, String.t()}
  def encrypt_for_secret(public_key, value) when is_binary(public_key) and is_binary(value) do
    with {:ok, _} <- Age.parse_recipient(public_key),
         data_key = :crypto.strong_rand_bytes(@data_key_size),
         {:ok, enc_data_key} <- Age.encrypt(public_key, data_key) do
      # The MAC is SHA-512 over the plaintext of every leaf in tree order; for
      # the fixed {value: ...} shape that is the one leaf's raw bytes.
      mac_hex = Base.encode16(:crypto.hash(:sha512, value))
      last_modified = DateTime.utc_now() |> DateTime.truncate(:second) |> DateTime.to_iso8601()
      enc_value = encrypt_leaf(data_key, value, "value:")
      enc_mac = encrypt_leaf(data_key, mac_hex, last_modified)
      {:ok, render(enc_value, public_key, enc_data_key, last_modified, enc_mac)}
    else
      {:error, reason} -> {:error, "encrypt for secret: #{reason}"}
    end
  end

  # AES-256-GCM in sops's wire format: a 32-byte random nonce, the
  # additional data as GCM's AAD, and the ciphertext and tag as separate
  # base64 fields.
  defp encrypt_leaf(data_key, plaintext, additional_data) do
    iv = :crypto.strong_rand_bytes(@gcm_nonce_size)

    {data, tag} =
      :crypto.crypto_one_time_aead(
        :aes_256_gcm,
        data_key,
        iv,
        plaintext,
        additional_data,
        @gcm_tag_size,
        true
      )

    "ENC[AES256_GCM,data:#{Base.encode64(data)},iv:#{Base.encode64(iv)},tag:#{Base.encode64(tag)},type:str]"
  end

  # The document as yaml.v3 renders the Go client's struct: four-space
  # indentation, the armored age file as a literal block, lastmodified quoted
  # so YAML does not resolve it as a timestamp.
  defp render(enc_value, recipient, enc_data_key, last_modified, enc_mac) do
    """
    value: #{enc_value}
    sops:
        age:
            - recipient: #{recipient}
              enc: |
    #{indent(enc_data_key, 12)}
        lastmodified: "#{last_modified}"
        mac: #{enc_mac}
        version: #{@sops_file_version}
    """
  end

  defp indent(text, columns) do
    pad = String.duplicate(" ", columns)

    text
    |> String.trim_trailing("\n")
    |> String.split("\n")
    |> Enum.map_join("\n", &(pad <> &1))
  end
end
