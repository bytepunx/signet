defmodule Signet.Spiffe.X509Context do
  @moduledoc """
  The material one `X509SVIDResponse` carries, decoded into what Erlang `ssl`
  takes: the default SVID's certificate chain and private key, and the trust
  bundle of each trust domain the workload may trust.
  """

  alias Spiffe.Workload.{X509SVID, X509SVIDResponse}

  @type t :: %__MODULE__{
          spiffe_id: String.t(),
          trust_domain: Signet.Spiffe.TrustDomain.t(),
          cert_chain: [binary()],
          key: {:PrivateKeyInfo, binary()},
          bundles: %{Signet.Spiffe.TrustDomain.t() => [binary()]}
        }

  defstruct spiffe_id: nil, trust_domain: nil, cert_chain: [], key: nil, bundles: %{}

  @doc """
  Decodes a response. The first SVID is the default identity, as in go-spiffe.
  """
  @spec from_response(X509SVIDResponse.t()) :: {:ok, t()} | {:error, String.t()}
  def from_response(%X509SVIDResponse{svids: []}), do: {:error, "Workload API returned no SVIDs"}

  def from_response(%X509SVIDResponse{
        svids: [%X509SVID{} = svid | _],
        federated_bundles: federated
      }) do
    with {:ok, trust_domain} <- Signet.Spiffe.TrustDomain.of_spiffe_id(svid.spiffe_id),
         {:ok, chain} <- split_der(svid.x509_svid, "SVID certificate chain"),
         {:ok, bundle} <- split_der(svid.bundle, "trust bundle"),
         {:ok, federated} <- federated_bundles(federated) do
      {:ok,
       %__MODULE__{
         spiffe_id: svid.spiffe_id,
         trust_domain: trust_domain,
         cert_chain: chain,
         key: {:PrivateKeyInfo, svid.x509_svid_key},
         bundles: Map.put(federated, trust_domain, bundle)
       }}
    end
  end

  @doc "The trust bundle for `trust_domain`, if the workload may trust it."
  @spec bundle_for(t(), Signet.Spiffe.TrustDomain.t()) :: {:ok, [binary()]} | :error
  def bundle_for(%__MODULE__{bundles: bundles}, trust_domain),
    do: Map.fetch(bundles, trust_domain)

  defp federated_bundles(federated) do
    Enum.reduce_while(federated, {:ok, %{}}, fn {id, der}, {:ok, acc} ->
      with {:ok, trust_domain} <- Signet.Spiffe.TrustDomain.of_spiffe_id(id),
           {:ok, certs} <- split_der(der, "federated bundle for #{id}") do
        {:cont, {:ok, Map.put(acc, trust_domain, certs)}}
      else
        {:error, reason} -> {:halt, {:error, reason}}
      end
    end)
  end

  @doc """
  Splits concatenated DER certificates, as the Workload API sends chains and
  bundles, into one binary per certificate.
  """
  @spec split_der(binary(), String.t()) :: {:ok, [binary()]} | {:error, String.t()}
  def split_der(der, what) when is_binary(der), do: split_der(der, what, [])

  defp split_der(<<>>, _what, []), do: {:error, "empty DER"}
  defp split_der(<<>>, _what, acc), do: {:ok, Enum.reverse(acc)}

  defp split_der(<<0x30, rest::binary>> = der, what, acc) do
    case der_length(rest) do
      {:ok, len, header} when byte_size(der) >= header + len ->
        <<cert::binary-size(header + len), tail::binary>> = der
        split_der(tail, what, [cert | acc])

      _ ->
        {:error, "malformed DER in #{what}"}
    end
  end

  defp split_der(_other, what, _acc), do: {:error, "malformed DER in #{what}"}

  # The length octets after the SEQUENCE tag: short form, or long form with a
  # byte count followed by that many length bytes. The header is the tag plus
  # the length octets.
  defp der_length(<<len, _::binary>>) when len < 0x80, do: {:ok, len, 2}

  defp der_length(<<1::1, n::7, rest::binary>>) when n in 1..4 and byte_size(rest) >= n do
    <<len::unsigned-big-size(n * 8), _::binary>> = rest
    {:ok, len, 2 + n}
  end

  defp der_length(_), do: :error
end
