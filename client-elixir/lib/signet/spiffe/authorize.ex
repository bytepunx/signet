defmodule Signet.Spiffe.Authorize do
  @moduledoc """
  Peer authorisation for SPIFFE mTLS, as an Erlang `ssl` `verify_fun`.

  `member_of/1` mirrors go-spiffe's `tlsconfig.AuthorizeMemberOf`: after the
  certificate chain has validated against the trust bundle, the peer's leaf
  certificate must carry exactly one SPIFFE ID in its URI subject alternative
  names, and that ID must belong to the expected trust domain. SPIFFE X.509-SVIDs
  carry no DNS or IP names, so hostname verification is not part of this check;
  `Signet.Workload` disables it on the connection and relies on this instead.
  """

  require Record

  Record.defrecordp(
    :otp_certificate,
    :OTPCertificate,
    Record.extract(:OTPCertificate, from_lib: "public_key/include/public_key.hrl")
  )

  Record.defrecordp(
    :otp_tbs_certificate,
    :OTPTBSCertificate,
    Record.extract(:OTPTBSCertificate, from_lib: "public_key/include/public_key.hrl")
  )

  Record.defrecordp(
    :extension,
    :Extension,
    Record.extract(:Extension, from_lib: "public_key/include/public_key.hrl")
  )

  @subject_alt_name {2, 5, 29, 17}

  @typedoc "An Erlang `ssl` `verify_fun` value: the function and its initial state."
  @type verify_fun :: {(tuple(), term(), term() -> {:valid | :fail | :unknown, term()}), term()}

  @doc """
  A `verify_fun` that accepts a peer whose SPIFFE ID is in `trust_domain` and
  rejects every other peer with a reason naming what was presented.
  """
  @spec member_of(Signet.Spiffe.TrustDomain.t()) :: verify_fun()
  def member_of(trust_domain) when is_binary(trust_domain) do
    {&verify/3, trust_domain}
  end

  @doc false
  def verify(_cert, {:bad_cert, reason}, _trust_domain), do: {:fail, reason}
  def verify(_cert, {:extension, _}, trust_domain), do: {:unknown, trust_domain}
  def verify(_cert, :valid, trust_domain), do: {:valid, trust_domain}

  def verify(cert, :valid_peer, trust_domain) do
    case authorize_peer(cert, trust_domain) do
      :ok -> {:valid, trust_domain}
      {:error, reason} -> {:fail, reason}
    end
  end

  @doc """
  Checks a decoded peer certificate (an `OTPCertificate` record, as `ssl`
  hands to a `verify_fun` or `:public_key.pkix_decode_cert(der, :otp)` returns)
  against `trust_domain`.
  """
  @spec authorize_peer(tuple(), Signet.Spiffe.TrustDomain.t()) :: :ok | {:error, String.t()}
  def authorize_peer(cert, trust_domain) do
    case spiffe_id(cert) do
      {:ok, id} ->
        case Signet.Spiffe.TrustDomain.of_spiffe_id(id) do
          {:ok, ^trust_domain} ->
            :ok

          {:ok, other} ->
            {:error,
             "peer SPIFFE ID #{inspect(id)} is in trust domain #{inspect(other)}, expected trust domain #{inspect(trust_domain)}"}

          {:error, reason} ->
            {:error, reason}
        end

      :none ->
        {:error,
         "peer certificate did not present a SPIFFE ID in its URI subject alternative names"}

      {:many, ids} ->
        {:error, "peer certificate presented #{length(ids)} SPIFFE IDs, expected exactly one"}
    end
  end

  @doc "The single SPIFFE ID among a decoded certificate's URI subject alternative names."
  @spec spiffe_id(tuple()) :: {:ok, String.t()} | :none | {:many, [String.t()]}
  def spiffe_id(cert) do
    ids =
      for extension(extnID: @subject_alt_name, extnValue: names) <- extensions(cert),
          {:uniformResourceIdentifier, uri} <- names,
          uri = to_string(uri),
          String.starts_with?(uri, "spiffe://"),
          do: uri

    case ids do
      [id] -> {:ok, id}
      [] -> :none
      many -> {:many, many}
    end
  end

  defp extensions(otp_certificate(tbsCertificate: tbs)) do
    case otp_tbs_certificate(tbs, :extensions) do
      list when is_list(list) -> list
      _ -> []
    end
  end
end
