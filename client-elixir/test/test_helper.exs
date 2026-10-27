# The integration tier (tests/integration/, tagged :integration) needs the
# real sops binary and is run by scripts/integration-client-elixir.sh with
# --include integration; mix test alone is the behaviour tier.
ExUnit.configure(exclude: [:integration])
ExUnit.start()
