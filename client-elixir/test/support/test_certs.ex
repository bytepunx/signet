defmodule Signet.TestCerts do
  @moduledoc """
  Throwaway certificate chains for behaviour tests: a root per trust domain and
  leaf certificates that carry a SPIFFE URI SAN and nothing else, the shape of
  an X.509-SVID. All keys are P-256, generated on the spot, and every
  signature is SHA-256: `:public_key`'s test data defaults to SHA-1, which
  TLS 1.3 refuses.
  """

  @p256 {1, 2, 840, 10045, 3, 1, 7}
  @subject_alt_name {2, 5, 29, 17}

  @doc "A SPIFFE subject alternative name extension for `uris`, plus any other names."
  def san(names) do
    {:Extension, @subject_alt_name, false, names}
  end

  @doc """
  A chain whose leaf has the given subject alternative names. Returns the leaf
  DER, its key as `ssl` takes it, and the root DER.
  """
  def chain(names) do
    %{cert: root_der, key: root_key} =
      :public_key.pkix_test_root_cert(~c"signet test root",
        key: {:namedCurve, @p256},
        digest: :sha256
      )

    root = %{cert: root_der, key: root_key}

    %{server_config: config} =
      :public_key.pkix_test_data(%{
        server_chain: %{
          root: root,
          intermediates: [],
          peer: [key: {:namedCurve, @p256}, digest: :sha256, extensions: [san(names)]]
        },
        client_chain: %{
          root: root,
          intermediates: [],
          peer: [key: {:namedCurve, @p256}, digest: :sha256]
        }
      })

    %{cert: config[:cert], key: config[:key], cacerts: [root_der]}
  end

  @doc "The leaf of `chain/1` decoded as `ssl` hands it to a `verify_fun`."
  def decode(der), do: :public_key.pkix_decode_cert(der, :otp)

  @doc "A PKCS#8 DER encoding of a P-256 key, as the Workload API sends keys."
  def pkcs8(key) do
    {:ECPrivateKey, ec_der} = key
    ec_key = :public_key.der_decode(:ECPrivateKey, ec_der)
    {:PrivateKeyInfo, der, :not_encrypted} = :public_key.pem_entry_encode(:PrivateKeyInfo, ec_key)
    der
  end
end
