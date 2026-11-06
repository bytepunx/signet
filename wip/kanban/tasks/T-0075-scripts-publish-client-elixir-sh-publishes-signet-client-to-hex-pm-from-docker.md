---
id: T-0075
type: task
nature: feature
title: scripts/publish-client-elixir.sh publishes signet_client to hex.pm from Docker
status: done
parent: S-0016
owner: alex
created: 2026-10-27T22:52:45Z
updated: 2026-11-06T07:59:06Z
transitions:
  - to: ready
    at: 2026-11-06T07:59:04Z
    by: agent-S-0016
  - to: in-progress
    at: 2026-11-06T07:59:04Z
    by: agent-S-0016
  - to: done
    at: 2026-11-06T07:59:06Z
    by: agent-S-0016
stream: S-0016
tags: [client-elixir]
touches: [scripts/publish-client-elixir.sh, scripts/README.md]
after: [T-0068]
usage:
  source: log
  seconds: 2
  estimated: true
  priced_by: harness
  models: []
---
# T-0075 scripts/publish-client-elixir.sh publishes signet_client to hex.pm from Docker

## Work

Write `scripts/publish-client-elixir.sh`, POSIX `sh` under `set -eu`, in the style of `scripts/test-client-elixir.sh`:

- It takes the version as its first argument (or `SIGNET_CLIENT_VERSION`), and refuses an empty one or one with the `client-elixir/v` prefix still on it.
- It builds the `signet-client-elixir:toolchain` image, generates the stubs (`buf generate` and `buf generate --template buf.gen.spiffe.yaml`), runs `mix deps.get`, and then `mix hex.publish --yes` in `MIX_ENV=dev` (`ex_doc` is a dev dependency, and `mix hex.publish` builds the docs), passing `SIGNET_CLIENT_VERSION` and `HEX_API_KEY` into the container with `-e`. It never prints `HEX_API_KEY`.
- `--dry-run` runs `mix hex.publish --dry-run --yes` instead and needs no `HEX_API_KEY`, so a developer or CI can check the package that would be published.
- Add a row for it to `scripts/README.md`.

It does not run the tests: the workflow runs `scripts/test-client-elixir.sh` first. It waits for the `mix.exs` task because the dry run only reports the tag's version once `mix.exs` reads it from the environment.

## Done when

- `scripts/publish-client-elixir.sh --dry-run 0.0.0-test` builds the package and docs without publishing and without `HEX_API_KEY`, and its output names `signet_client 0.0.0-test`.
- The script exits non-zero with a message when the version is missing, empty, or prefixed, and when run without `--dry-run` and without `HEX_API_KEY`.
- `scripts/README.md` lists the script.

## Notes

Putting the publish commands in a script rather than in the workflow follows `design/conventions/tooling.md` and lets an operator publish by hand the same way CI does.
