---
id: T-0068
type: task
nature: feature
title: mix.exs reads its version from SIGNET_CLIENT_VERSION and names bytepunx/signet
status: backlog
parent: S-0016
owner: alex
created: 2026-10-27T22:52:33Z
updated: 2026-10-27T22:52:33Z
transitions: []
stream: S-0016
tags: [client-elixir]
touches: [client-elixir/mix.exs]
---
# T-0068 mix.exs reads its version from SIGNET_CLIENT_VERSION and names bytepunx/signet

## Work

Edit `client-elixir/mix.exs`:

- Replace `@version "0.1.0"` with `@version System.get_env("SIGNET_CLIENT_VERSION", "0.0.0-dev")`, so a build with the variable unset is a development build and the release workflow sets it from the tag.
- Point `docs: source_ref` at the new tag form, `"client-elixir/v#{@version}"`, in place of `"elixir-v#{@version}"`.
- Keep `@source_url "https://github.com/bytepunx/signet"`, add a `"Changelog"` link to `"#{@source_url}/releases"` because the GitHub Release is now the changelog, and drop `CHANGELOG.md` from `package: files`.
- Leave the dependencies and the toolchain pins alone.

## Done when

- With the toolchain image from `scripts/test-client-elixir.sh`, `docker run --rm -v "$PWD/client-elixir:/app" -e SIGNET_CLIENT_VERSION=1.2.3 signet-client-elixir:toolchain mix run --no-start -e 'IO.puts(Mix.Project.config()[:version])'` prints `1.2.3`, and prints `0.0.0-dev` without the `-e` flag.
- `scripts/test-client-elixir.sh` passes with the variable unset.
- `mix.exs` has no `elixir-v` and no `CHANGELOG.md`.

## Notes

The placeholder version is the design's rule in `design/system/release.md`, "Versions in packages": no version file is bumped in a commit. Hex versions must be semver, and `0.0.0-dev` is a valid pre-release.
