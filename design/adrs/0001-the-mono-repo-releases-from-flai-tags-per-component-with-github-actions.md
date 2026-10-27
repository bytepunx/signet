---
id: ADR-0001
title: The mono-repo releases from flai tags per component with GitHub Actions publishing
status: proposed
date: 2026-10-27
supersedes: []
superseded_by: []
---

# ADR-0001 The mono-repo releases from flai tags per component with GitHub Actions publishing

## Context

The server, the proto schema, and the six clients were imported into this mono-repo from
three repositories, each with its own release-please setup, its own tag scheme (`v*` for
the server and proto, `go/v*`, `python-v*`, `rust-v*`, `typescript-v*`, `csharp-v*` for
the clients), and publish workflows that bumped version files and `CHANGELOG.md` in
commits. GitHub reads workflows only from the root `.github/`, so the imported workflows
under `server/.github/` and `proto/.github/` are dead, and the client workflows were not
imported at all. Nothing is built, tested, or published from this repository, and two
version authorities would coexist if release-please were revived here next to the
`flai release --pending` flow that `design/conventions/git.md` already prescribes. The
gap analysis in E-0002 lists every difference.

## Decision

The mono-repo releases each component from a `<component>/vX.Y.Z` tag that `flai release --pending` alone creates, with a GitHub Actions workflow per component that derives the version from the tag, publishes the artifacts, and writes the GitHub Release.

In detail:

- `flai release --pending` is the only version authority. It computes one bump per
  component from the stories accepted since that component's last release, tags, and
  pushes. Nothing in CI computes, bumps, or commits a version. release-please is retired.
- Tags are `<component>/vX.Y.Z`, where `<component>` is the component's `name` in
  `system-flow.yaml` (`server`, `proto`, `client-go`, `client-python`,
  `client-typescript`, `client-rust`, `client-csharp`, `client-elixir`).
- Baseline tags are placed on each component's import commit at the version the old
  repository last released (`server/v0.11.0`, `proto/v0.4.0`, `client-go/v0.7.0`,
  `client-python/v0.5.0`, `client-rust/v0.6.0`, `client-typescript/v0.9.0`,
  `client-csharp/v0.6.0`), so numbering continues from the old repositories.
- Package versions are derived from the tag at build time, by stripping the
  `<component>/v` prefix. No version file (`package.json`, `Cargo.toml`,
  `SignetClient.csproj`, `pyproject.toml`, `mix.exs`, `Chart.yaml`) is bumped in a commit;
  the value checked in is a placeholder the release workflow overrides.
- The changelog of a component release is its GitHub Release, named by the tag and built
  from the tag annotation and the commits since the previous tag of that component.
  `CHANGELOG.md` files are no longer maintained.
- One `ci.yml` at the root runs a path-filtered job per component on pull requests and
  pushes to `main`, calling the same `scripts/test-<component>.sh` developers run; one
  `release-<component>.yml` per component runs on `push` of `<component>/v*` tags, builds
  and tests as CI does, publishes the component's artifacts, and creates the GitHub
  Release.
- In-repo consumers (the server and every client) generate stubs from `proto/proto` by
  directory input, so a schema change and its adopters land in one change. The BSR module
  `buf.build/bytepunx/signet-proto` stays for external consumers: pushed on every change
  to `proto/` on `main`, labeled on `proto/v*` tags.
- The old repositories `signet-clients`, `signet-proto`, and `signet-admin` are archived,
  each with a README pointing here, once publishing works from this repository.

## Consequences

- One place decides versions, so a component's release is intentional and traceable to
  the stories it bundles; a story that touches a component incidentally contributes a
  patch, so stories should touch one component each.
- Each client's package metadata and the Go module paths must name this repository
  (`github.com/bytepunx/signet/server`, `github.com/bytepunx/signet/client-go`), since
  `client-go/vX.Y.Z` is the tag form Go modules in a subdirectory require.
- Release workflows need a step that writes the tag's version into the package before
  building; release notes, image tags, and chart versions strip the `server/` prefix.
- Every publisher (PyPI, NuGet, npm, crates.io, hex.pm, Docker Hub, BSR) is re-registered
  against `bytepunx/signet` and the new workflow filenames, and every secret is re-created
  here; `docs/operators/release.md` is the runbook.
- `install.sh` must resolve the newest `server/v*` release rather than `releases/latest`,
  which may be a client.
- `proto-bump.yml` and the per-client BSR pins are retired for in-repo consumers.
- `client-erlang/` stops being a component so an incidental touch cannot release a client
  that does not exist.

## Alternatives considered

- Keep release-please per component with a root manifest: a second version authority next
  to flai, and version bumps as commits that flai's acceptance flow does not make.
- Root `v*` tags releasing every component together: couples unrelated components and
  breaks Go's subdirectory module tagging.
- Keep `CHANGELOG.md` files bumped by the release workflow: a commit from CI on `main`
  after every tag, racing with acceptance merges.
- Keep in-repo consumers on the BSR pin with `proto-bump.yml`: a schema change needs a
  publish and a bump PR per client before it can be used in the same repository.
