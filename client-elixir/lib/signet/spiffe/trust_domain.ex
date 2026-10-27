defmodule Signet.Spiffe.TrustDomain do
  @moduledoc """
  A SPIFFE trust domain name, validated as the SPIFFE ID standard requires:
  lowercase letters, digits, dots, dashes, and underscores, at most 255 bytes.

  `parse/1` accepts either a bare name (`"example.org"`) or a SPIFFE ID
  (`"spiffe://example.org/ns/default/sa/signet"`), whose host is the trust
  domain, as go-spiffe's `TrustDomainFromString` does.
  """

  @type t :: String.t()

  @max_length 255

  @doc "Parses and validates a trust domain name or SPIFFE ID."
  @spec parse(String.t()) :: {:ok, t()} | {:error, String.t()}
  def parse("spiffe://" <> rest) do
    rest |> String.split("/", parts: 2) |> hd() |> parse()
  end

  def parse(""), do: {:error, "trust domain is missing"}

  def parse(name) when is_binary(name) do
    cond do
      byte_size(name) > @max_length ->
        {:error, "trust domain #{inspect(name)} is longer than #{@max_length} bytes"}

      not Regex.match?(~r/^[a-z0-9._-]+$/, name) ->
        {:error,
         "trust domain #{inspect(name)} may only contain lowercase letters, numbers, dots, dashes, and underscores"}

      true ->
        {:ok, name}
    end
  end

  @doc "The trust domain of a SPIFFE ID, or an error if `id` is not one."
  @spec of_spiffe_id(String.t()) :: {:ok, t()} | {:error, String.t()}
  def of_spiffe_id("spiffe://" <> _ = id), do: parse(id)
  def of_spiffe_id(other), do: {:error, "#{inspect(other)} is not a SPIFFE ID"}

  @doc "The SPIFFE ID of a trust domain itself, the key the Workload API uses for federated bundles."
  @spec id(t()) :: String.t()
  def id(name), do: "spiffe://" <> name
end
