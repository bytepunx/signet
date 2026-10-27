# Fetches one secret over SPIFFE mTLS, the Elixir counterpart of the Go
# client's examples/getsecret. From client-elixir/:
#
#   mix run examples/getsecret.exs -- --addr signet.internal:8443 \
#     --socket unix:///run/spire/sockets/agent.sock --trust-domain example.org \
#     --namespace default --service example --name api-key
#
# Or in Docker, from the repository root:
#
#   scripts/test-client-elixir.sh mix run examples/getsecret.exs -- --addr ...

{opts, _, _} =
  OptionParser.parse(System.argv(),
    strict: [
      addr: :string,
      socket: :string,
      trust_domain: :string,
      namespace: :string,
      service: :string,
      name: :string
    ]
  )

addr = opts[:addr] || "localhost:8443"
socket = opts[:socket] || "unix:///run/spire/sockets/agent.sock"
trust_domain = opts[:trust_domain] || "example.org"

request = %Signet.V1.GetSecretRequest{
  namespace: opts[:namespace] || "default",
  service: opts[:service] || "example",
  name: opts[:name] || "api-key"
}

case Signet.Workload.connect(addr, socket, trust_domain) do
  {:ok, channel, source} ->
    case Signet.V1.SecretsService.Stub.get_secret(channel, request, timeout: 10_000) do
      {:ok, resp} ->
        IO.puts("version=#{resp.version} value=#{inspect(resp.value)}")

      {:error, reason} ->
        IO.puts(:stderr, "GetSecret: #{inspect(reason)}")
        System.halt(1)
    end

    Signet.Workload.disconnect(channel, source)

  {:error, reason} ->
    IO.puts(:stderr, "dial: #{inspect(reason)}")
    System.halt(1)
end
