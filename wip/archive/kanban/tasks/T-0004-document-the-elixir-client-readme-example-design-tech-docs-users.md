---
id: T-0004
type: task
nature: feature
title: "Document the Elixir client: README, example, design/tech, docs/users"
status: done
parent: S-0001
owner: alex
created: 2026-10-27T19:07:37Z
updated: 2026-10-27T19:24:07Z
transitions:
  - to: ready
    at: 2026-10-27T19:20:41Z
    by: agent
  - to: in-progress
    at: 2026-10-27T19:20:53Z
    by: agent
  - to: done
    at: 2026-10-27T19:24:07Z
    by: agent
stream: S-0001
tags: []
touches: [client-elixir/README.md, client-elixir/examples/getsecret.exs, client-elixir/CHANGELOG.md, design/tech/README.md, design/tech/elixir.md, design/tech/grpc-elixir.md, design/tech/protobuf-elixir.md, design/tech/buf.md, docs/users/index.md, docs/users/elixir-client.md, client-erlang/README.md, ".markdownlintignore"]
---
# T-0004 Document the Elixir client: README, example, design/tech, docs/users

## Work

Write `client-elixir/README.md` in the shape of the Go and Python READMEs, a `getsecret` example script, `design/tech/` entries for Elixir and Erlang/OTP, grpc with gun, protobuf, and buf with versions and alternatives, a users guide page for the client, and update `client-erlang/README.md` to point at the Elixir client's toolchain decision.

## Done when

Docs describe what exists, markdownlint and `flai check --strict` pass, and the tech index lists every new dependency.

## Notes
