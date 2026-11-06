# Toolchain image for building and testing the Elixir client. Nothing is
# installed on the host: scripts/test-client-elixir.sh builds this image and
# runs buf generate, mix deps.get, mix compile, and mix test inside it with
# the repository root mounted at /repo (working directory client-elixir/), so
# deps/, _build/, and lib/gen/ land in the (gitignored) folders on the host and survive between runs.
#
# Versions are pinned here and recorded in design/tech/.
FROM hexpm/elixir:1.19.6-erlang-28.5.0.8-debian-bookworm-20261020-slim

ARG BUF_VERSION=1.71.0
ARG PROTOBUF_VERSION=0.17.1
ARG SOPS_VERSION=3.13.3

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates curl \
    && rm -rf /var/lib/apt/lists/*

RUN curl -fsSL "https://github.com/bufbuild/buf/releases/download/v${BUF_VERSION}/buf-Linux-$(uname -m)" \
      -o /usr/local/bin/buf \
    && chmod +x /usr/local/bin/buf

# The real sops binary, for the integration test only (tests/integration/):
# it decrypts what Signet.SopsEncrypt produces. The client never runs sops.
RUN curl -fsSL "https://github.com/getsops/sops/releases/download/v${SOPS_VERSION}/sops-v${SOPS_VERSION}.linux.$(dpkg --print-architecture)" \
      -o /usr/local/bin/sops \
    && chmod +x /usr/local/bin/sops

# Hex, rebar (gun is an Erlang dependency built with rebar3), and the
# protoc-gen-elixir escript buf.gen.yaml names as a local plugin.
ENV MIX_HOME=/opt/mix \
    HEX_HOME=/opt/hex \
    PATH=/opt/mix/escripts:$PATH
RUN mix local.hex --force \
    && mix local.rebar --force \
    && mix escript.install hex protobuf "${PROTOBUF_VERSION}" --force

WORKDIR /app
