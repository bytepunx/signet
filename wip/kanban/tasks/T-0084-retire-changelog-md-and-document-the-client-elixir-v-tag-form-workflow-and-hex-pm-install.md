---
id: T-0084
type: task
nature: improvement
title: Retire CHANGELOG.md and document the client-elixir/v tag form, workflow, and hex.pm install
status: backlog
parent: S-0016
owner: alex
created: 2026-10-27T22:53:01Z
updated: 2026-10-27T22:53:08Z
transitions: []
stream: S-0016
tags: [client-elixir]
touches: [client-elixir/README.md, client-elixir/CHANGELOG.md, design/system/release.md]
after: [T-0068, T-0075, T-0082]
---
# T-0084 Retire CHANGELOG.md and document the client-elixir/v tag form, workflow, and hex.pm install

## Work

- Delete `client-elixir/CHANGELOG.md`. Its `Unreleased` list becomes the first release's notes through the tag annotation, so nothing is carried over; no other file in the repository links to it (check with `grep -rn CHANGELOG client-elixir docs design/system`).
- In `client-elixir/README.md`, replace the sentence "The package is not yet published to Hex" with a pointer to `https://hex.pm/packages/signet_client`, and add a `## Releasing` section that names the tag form `client-elixir/vX.Y.Z`, says `flai release --pending` creates it, names the workflow `.github/workflows/release-client-elixir.yml`, says the version comes from the tag through `SIGNET_CLIENT_VERSION` so `mix.exs` carries the placeholder `0.0.0-dev`, says the GitHub Release is the changelog, and links `../docs/operators/release.md` and `scripts/publish-client-elixir.sh --dry-run`.
- In `design/system/release.md`, "Versions in packages", change the `client-elixir` row from `mix.exs` `@version` to `mix.exs` `@version`, read from `SIGNET_CLIENT_VERSION`.
- Bump `updated:` on the design file.

## Done when

- `client-elixir/CHANGELOG.md` is gone and `grep -rn CHANGELOG client-elixir docs design/system` finds nothing that points at it.
- The README's install section points at hex.pm, and its `## Releasing` section names the tag form and the workflow file.
- `flai check --strict` and the markdown lint pass.

## Notes

It waits for the `mix.exs` and script tasks and for the workflow task so that what it names exists. `client-elixir/README.md` is also edited by S-0020 (stub generation section); the two changes are in different sections.
