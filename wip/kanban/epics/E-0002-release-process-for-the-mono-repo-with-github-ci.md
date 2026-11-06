---
id: E-0002
type: epic
nature: feature
title: Release process for the mono-repo with GitHub CI
status: in-progress
owner: arobson
created: 2026-10-27T19:44:30Z
updated: 2026-11-06T07:19:26Z
transitions:
  - to: ready
    at: 2026-10-27T19:47:51Z
    by: arobson
  - to: in-progress
    at: 2026-10-27T20:53:09Z
    by: agent-S-0004
tags: []
topics: [release, ci, github]
usage:
  source: sum
  seconds: 12162
  estimated: true
  priced_by: estimate
  empty_wakes: 2
  turns:
    - day: 2026-10-27
      ceremony: 8
      test_runs: 4
      hand_edits: 2
      work: 106
    - day: 2026-10-28
      test_runs: 1
      work: 66
    - day: 2026-10-29
      ceremony: 1
      work: 37
    - day: 2026-10-30
      ceremony: 2
      work: 24
    - day: 2026-11-03
      ceremony: 1
      work: 5
    - day: 2026-11-05
      ceremony: 2
      empty_wakes: 2
      work: 41
    - day: 2026-11-06
      ceremony: 1
      work: 21
  models:
    - model: claude-fable-5-1
      input: 770
      output: 20698
      cache_read: 2226396
      cache_write: 123555
      cost: 4.0703
    - model: claude-sonnet-5
      input: 74
      output: 18786
      cache_read: 3155207
      cache_write: 109675
      cost: 1.2577
    - model: claude-sonnet-5-5
      input: 792
      output: 129867
      cache_read: 30021134
      cache_write: 1541719
      cost: 9.8444
  strategic:
    - kind: planner
      seconds: 403
      estimated: true
      models:
        - model: claude-haiku-5-5
          input: 30
          output: 94
          cache_read: 810527
          cache_write: 87952
          cost: 0
        - model: claude-sonnet-5-5
          input: 240
          output: 44089
          cache_read: 8954470
          cache_write: 517166
          cost: 4.683
    - kind: orchestrator
      seconds: 124
      estimated: true
      models:
        - model: claude-sonnet-5-5
          input: 72
          output: 528
          cache_read: 1764017
          cache_write: 510644
          cost: 0.9228
---
# E-0002 Release process for the mono-repo with GitHub CI

## Outcome

Every component of the mono-repo (server, proto, and the six clients) is tested on every pull request and published from one release process: `flai release --pending` decides versions and pushes `<component>/vX.Y.Z` tags, and GitHub Actions workflows at the repository root react to those tags to build, verify, and publish each component's artifacts (binaries, images, Helm chart, BSR label, packages) and create a GitHub Release per component. release-please is retired, the old repositories are archived with pointers here, and an operator runbook under `docs/operators/` describes the credentials, trusted publishers, branch protection, and the publish step.

## Stories
- S-0004 Make the system-flow check green on main
- S-0005 Decide and record the mono-repo release and CI model
- S-0006 Baseline component tags and retire release-please
- S-0007 Root CI workflow that builds and tests every component on pull requests
- S-0008 Operator setup: credentials, trusted publishers, branch protection, and archiving the old repositories
- S-0009 Server release workflow on server/v* tags
- S-0010 Proto publish workflow: BSR push from main and label on proto/v* tags
- S-0011 Go client: module path for the mono-repo and release workflow on client-go/v* tags
- S-0012 Python client: release workflow on client-python/v* tags with PyPI trusted publishing
- S-0013 TypeScript client: release workflow on client-typescript/v* tags with npm provenance
- S-0014 Rust client: release workflow on client-rust/v* tags publishing to crates.io
- S-0015 C# client: release workflow on client-csharp/v* tags with NuGet trusted publishing
- S-0016 Elixir client: release workflow on client-elixir/v* tags publishing to hex.pm
- S-0017 Retire the old three-repository layout from agent instructions and READMEs
- S-0018 Decide the admin sub-project's place in the mono-repo
- S-0019 Smoke-test echo images built from the clients' examples
- S-0020 Generate in-repo stubs from proto/ by directory instead of the BSR pin
- S-0029 Verify publish-server-main.yml and install.sh after S-0009 reaches main
- S-0031 scripts/test-client-csharp.sh crashes in dotnet when the caller's uid has no passwd entry in the SDK image
- S-0032 system-flow check fails on main: flai is not on PATH, and flai check --strict has findings behind it

## Notes

### Gap analysis (2026-10-27)

The migration imported each repository's files and history into a folder but dropped everything that lived at the old repositories' roots, and the mono-repo has not yet adopted a release model of its own.

| Area | Found | Gap |
|------|-------|-----|
| CI on GitHub | The only workflow at the root is `system-flow-check.yml`, and its last run on `main` failed on markdownlint errors in `server/docs/secrets-workflow.md` | No component is built, linted, or tested on pull requests. The one check that exists is red. |
| Inherited workflows | `server/.github/workflows/` (CI, Publish, Release Please, Release) and `proto/.github/workflows/` (CI, Publish, Release Please, Release) are intact but dead: GitHub only reads `.github/` at the root. The client workflows of signet-clients (`ci.yml`, `publish-{go,python,typescript,rust,csharp}.yml`, `proto-bump.yml`, `smoke-images.yml`, `release-please.yml`) were not imported at all and exist only in the old repository | Nothing publishes anything. The old files are the reference for what each component's publish step did. |
| Versioning | Old scheme: release-please per repository, root `v*` tags for the server (v0.2.0 to v0.11.0 are in this repository's history and as GitHub Releases), `v*` for proto (v0.4.0), and `go/v*`, `python-v*`, `rust-v*`, `typescript-v*`, `csharp-v*` for the clients. New scheme per `design/conventions/git.md`: `flai release --pending` computes one bump per component from accepted stories and tags `<component>/vX.Y.Z`. `flai release --pending --dry-run` today plans `client-elixir/v0.1.0` and `client-erlang/v0.0.1`, and would start every other component at 0.0.0 | Two version authorities. Baseline tags (`server/v0.11.0`, `proto/v0.4.0`, `client-go/v0.7.0`, `client-python/v0.5.0`, `client-rust/v0.6.0`, `client-typescript/v0.9.0`, `client-csharp/v0.6.0`) must be placed on the import commits so flai continues the numbering; release-please configs and manifests in `server/` and `proto/` must go. |
| Version files | release-please bumped `package.json`, `Cargo.toml`, `SignetClient.csproj`, `pyproject.toml`, and `CHANGELOG.md` in commits. flai tags code components and bumps no file in them | Publish workflows must derive the version from the tag at build time, and the changelog moves to the GitHub Release (from the tag annotation and the commits since the previous component tag), or a story decides otherwise. |
| Releasing incidentally | Under git.md, any story that touches a component folder contributes a patch to that component when published, even a README edit. S-0001 touched `client-erlang/README.md`, so the plan releases `client-erlang` 0.0.1, a client that does not exist | `client-erlang/` must stop being a component (fold its note into the Elixir client, delete the folder). Stories should touch one component each so releases stay intentional. |
| Server | `go.mod` is `github.com/bytepunx/signet` but the module lives in `server/`; `buf.gen.yaml` overrides `go_package_prefix` to `github.com/bytepunx/signet/gen`. The old release built CLI binaries, pushed `ghcr.io/bytepunx/signetd` and `docker.io/bytepunx/signet` with `vX.Y.Z` and `latest` tags, and pushed the Helm chart to `oci://ghcr.io/bytepunx/charts` with version from the tag | Module path must become `github.com/bytepunx/signet/server` so `go install .../server/cmd/signet@server/vX.Y.Z` resolves. Image tags, chart version, and release notes must strip the `server/` prefix. The main-branch image publish must be path-filtered to `server/`. |
| `install.sh` | Resolves `releases/latest` of `bytepunx/signet` and downloads `signet-<os>-<arch>` | With per-component releases, "latest" may be a client. It must pick the newest `server/v*` release. |
| Proto | Published to the BSR as `buf.build/bytepunx/signet-proto` on every push to main, labeled on `v*` tags; `buf breaking` compared against the last `v*` tag. Server and every client pin `buf.build/bytepunx/signet-proto:v0.4.0` in their `buf.gen.yaml`; `proto-bump.yml` opened PRs to bump those pins | The BSR push needs `working-directory: proto` and a path filter; breaking runs against the last `proto/v*` tag. In-repo consumers can generate from `../proto/proto` directly, which makes `proto-bump.yml` unnecessary; the BSR stays for external consumers. This is a decision for the ADR story. |
| Clients | Package metadata still names the old repository: `go.mod` module `github.com/bytepunx/signet-clients/go`, `Cargo.toml` and `SignetClient.csproj` repository URLs, `package.json` repository. README publish sections link to `../.github/workflows/publish-*.yml` and release-please tags | Go module path must become `github.com/bytepunx/signet/client-go` (its tag form `client-go/vX.Y.Z` is exactly what flai produces). Every other metadata field and README section needs the new repository and tag scheme. |
| Credentials and trust | The `bytepunx/signet` repository has no Actions secrets and no branch protection. PyPI and NuGet trusted publishing were registered for `bytepunx/signet-clients` with specific workflow filenames; npm used OIDC provenance from that repository too. Docker Hub, BSR (`BUF_TOKEN`, `BSR_AUTH_TOKEN`), crates.io (`CARGO_REGISTRY_TOKEN`), and NuGet (`NUGET_USER`) secrets lived in the old repositories | Every publisher must be re-registered against `bytepunx/signet` and the new workflow filenames, every secret re-created, and branch protection with required checks set up. Operator work, to be driven by a runbook. |
| Old repositories | `signet-clients`, `signet-proto`, and `signet-admin` are still live and unarchived on GitHub | Archive them with a README pointer once publishing works here, so release-please there cannot open PRs and consumers find the new home. |
| Local entry points | `make test` runs only the Elixir client; `scripts/integration.sh` and `scripts/smoke.sh` are placeholders; the server has its own `Makefile` with `test`, `lint`, `build` | CI must call the same scripts as developers do (`design/conventions/tooling.md`), so `scripts/` needs one test script per component and `test.sh` runs them all. |
| Nested agent instructions and stale docs | `server/CLAUDE.md` and `admin/CLAUDE.md` describe the old three-repository layout; `server/README.md`, `proto/README.md`, `admin/README.md`, and every client README reference the old repositories | Fold what is still true into the root conventions and docs, delete the rest. |
| Admin | `admin/` holds only a README and CLAUDE.md but is registered as a SvelteKit project | Either scaffold it or remove it from `system-flow.yaml` until it is started. |
| Smoke tests | `smoke-images.yml` built `ghcr.io/bytepunx/signet-echo-<language>` images for the external signet-smoke-test harness on every push | Restore as a path-filtered workflow if the harness is still in use. |

### Recommended model

1. flai is the only version authority. `flai release --pending` tags and pushes; nothing in CI computes a version.
2. One `ci.yml` at the root with a job per component, each path-filtered to its folder and calling `scripts/test-<component>.sh`, plus the existing system-flow check. Those jobs are the required status checks on `main`.
3. One `release-<component>.yml` per component, triggered by `push: tags: ["<component>/v*"]`, deriving the version by stripping the prefix, building and testing as CI does, then publishing and creating a GitHub Release named by the tag with notes from the tag annotation.
4. In-repo consumers generate stubs from `proto/proto` by directory input; the BSR module is published for external consumers and labeled on `proto/v*` tags.
5. Conventional commit subjects stay (`<type>: [S-nnnn] ...`) for readable history, but do not drive versions.
- 2026-10-27T19:47:51Z: moved to ready: follows S-0004, which moved to ready
- 2026-10-27T20:53:09Z: moved to in-progress: follows S-0004, which moved to in-progress
