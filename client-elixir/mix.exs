defmodule SignetClient.MixProject do
  use Mix.Project

  @version "0.1.0"
  @source_url "https://github.com/bytepunx/signet"

  def project do
    [
      app: :signet_client,
      version: @version,
      elixir: "~> 1.19",
      elixirc_paths: elixirc_paths(Mix.env()),
      start_permanent: Mix.env() == :prod,
      deps: deps(),
      description:
        "Elixir client for signet: SPIFFE mTLS workload access and bearer-token admin access",
      package: package(),
      docs: [main: "readme", extras: ["README.md"], source_ref: "elixir-v#{@version}"]
    ]
  end

  def application do
    [extra_applications: [:logger, :ssl, :public_key, :crypto]]
  end

  defp elixirc_paths(:test), do: ["lib", "test/support"]
  defp elixirc_paths(_), do: ["lib"]

  defp deps do
    [
      # gRPC client. 1.0 split the old :grpc package into a client (:grpc, on
      # :grpc_core) and a server (:grpc_server); this library only ever calls.
      {:grpc, "~> 1.0.5"},
      # The HTTP/2 transport :grpc drives by default, and the only one of its
      # two adapters whose Unix-socket support (:gun.open_unix/2) the SPIFFE
      # Workload API dial depends on. :grpc lists it optional, so it is
      # pinned here.
      {:gun, "~> 2.4.0"},
      # Runtime for the stubs protoc-gen-elixir generates into lib/gen, and the
      # Google well-known types (Struct, Value, Timestamp) signet-proto imports.
      {:protobuf, "~> 0.17.1"},
      {:ex_doc, "~> 0.40", only: :dev, runtime: false}
    ]
  end

  defp package do
    [
      name: "signet_client",
      licenses: ["Apache-2.0"],
      links: %{"GitHub" => @source_url},
      files: ~w(lib proto buf.gen.yaml buf.gen.spiffe.yaml mix.exs README.md CHANGELOG.md)
    ]
  end
end
