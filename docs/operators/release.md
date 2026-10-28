---
title: Publishing a release
updated: 2026-10-28
status: active
topics: [all, release, ci, github]
---

# Publishing a release

How an operator publishes signet's components from this repository, what each release
produces, where it lands, and what the workflows need from you. The design is in
[design/system/release.md](../../design/system/release.md).

## Publish

Releases are cut by hand, over everything accepted since the last publish, never per
story. From a checkout of `main` with push rights:

```bash
git fetch
flai release --pending --dry-run   # see which components bump, and why
flai release --pending             # tag and push
```

`flai release --pending` writes one annotated tag `<component>/vX.Y.Z` per component with
accepted stories since its last tag and pushes them. Each tag starts the component's
release workflow under `.github/workflows/`; follow it on the repository's Actions page.
No version file is edited and no commit is made: the tag is the version.

If the push is refused because `main` moved, merge the remote branch and run it again.
Never rebase or force-push accepted history.

## What each release produces

| Component | Tag | Produces | Find it at |
|-----------|-----|----------|------------|
| server | `server/vX.Y.Z` | CLI binaries `signet-<os>-<arch>` with checksums; `signetd` image tagged `X.Y.Z` and `latest`; Helm chart `signet` version `X.Y.Z` | GitHub Release `server/vX.Y.Z`; `ghcr.io/bytepunx/signetd` and `docker.io/bytepunx/signet`; `oci://ghcr.io/bytepunx/charts/signet` |
| proto | `proto/vX.Y.Z` | BSR module labeled `vX.Y.Z` | `buf.build/bytepunx/signet-proto`; GitHub Release `proto/vX.Y.Z` |
| client-go | `client-go/vX.Y.Z` | Go module version `vX.Y.Z` | `go get github.com/bytepunx/signet/client-go@vX.Y.Z`; GitHub Release |
| client-python | `client-python/vX.Y.Z` | `signet-client` wheel and sdist | PyPI; GitHub Release |
| client-typescript | `client-typescript/vX.Y.Z` | `@bytepunx/signet-client` with provenance | npm; GitHub Release |
| client-rust | `client-rust/vX.Y.Z` | crate `signet-client` | crates.io; GitHub Release |
| client-csharp | `client-csharp/vX.Y.Z` | NuGet package `Signet.Client` | NuGet; GitHub Release |
| client-elixir | `client-elixir/vX.Y.Z` | Hex package `signet_client` | hex.pm; GitHub Release |

The GitHub Release is the changelog: its body is the tag annotation, which lists the
accepted items the release bundles, and the commits since the component's previous tag.
Pushes to `main` also publish the `signetd` image tagged `main` and push the proto module
to the BSR without a label.

## Credentials and trusted publishers

Set these once in the `bytepunx/signet` repository. Secrets go under Settings, Secrets and
variables, Actions. Trusted publishers are registered on the registry's side against the
repository `bytepunx/signet`, the environment left empty, and the exact workflow filename.

| Registry | Mechanism | What to set up |
|----------|-----------|----------------|
| GitHub Releases, ghcr.io | `GITHUB_TOKEN` | Nothing; the workflows request `contents: write` and `packages: write` |
| Docker Hub | Secrets | `DOCKERHUB_USERNAME`, `DOCKERHUB_TOKEN` (an access token with write scope on `bytepunx/signet`) |
| Buf Schema Registry | Secret | `BUF_TOKEN`, a token with write access to `buf.build/bytepunx/signet-proto` |
| PyPI | Trusted publisher | Register `bytepunx/signet`, workflow `release-client-python.yml`, on the `signet-client` project |
| npm | Trusted publisher | Register `bytepunx/signet`, workflow `release-client-typescript.yml`, on `@bytepunx/signet-client` |
| crates.io | Secret | `CARGO_REGISTRY_TOKEN` with publish scope for `signet-client` |
| NuGet | Trusted publisher | Register `bytepunx/signet`, workflow `release-client-csharp.yml`, on `Signet.Client`; set `NUGET_USER` to the account that owns the policy |
| hex.pm | Secret | `HEX_API_KEY`, a key with `api:write` for the `signet_client` package |

The publishers of the old repositories (`bytepunx/signet-clients`, `bytepunx/signet-proto`)
do not carry over: each registry checks the repository and workflow filename, so every one
is registered again here. S-0008 drives this setup.

### Setup checklist

Tick each item in S-0008's notes with the date when it is done. Check the secrets at any
time with `scripts/set-release-secrets.sh check`; set them with
`scripts/set-release-secrets.sh set`, which reads each value from the environment variable
of the same name (for example `BUF_TOKEN=... DOCKERHUB_TOKEN=... scripts/set-release-secrets.sh set`)
and skips any that is empty. Values never go on the command line.

| Done | Item | Created at | Scope | Names workflow |
|------|------|------------|-------|----------------|
| [ ] | Secret `BUF_TOKEN` | buf.build, user settings, tokens | write on `bytepunx/signet-proto` | `release-proto.yml`, `publish-proto-main.yml`, `ci.yml` |
| [ ] | Secret `DOCKERHUB_USERNAME` | Docker Hub account name | none | `release-server.yml`, `publish-server-main.yml` |
| [ ] | Secret `DOCKERHUB_TOKEN` | Docker Hub, account settings, security | write on `bytepunx/signet` | `release-server.yml`, `publish-server-main.yml` |
| [ ] | Secret `CARGO_REGISTRY_TOKEN` | crates.io, account settings, API tokens | publish for `signet-client` | `release-client-rust.yml` |
| [ ] | Secret `NUGET_USER` | NuGet account name that owns the policy | none | `release-client-csharp.yml` |
| [ ] | Secret `HEX_API_KEY` | `mix hex.user key generate` | `api:write` for `signet_client` | `release-client-elixir.yml` |
| [ ] | PyPI trusted publisher | pypi.org, project `signet-client`, Publishing | repository `bytepunx/signet`, no environment | `release-client-python.yml` |
| [ ] | npm trusted publisher | npmjs.com, package `@bytepunx/signet-client`, Settings | repository `bytepunx/signet`, no environment | `release-client-typescript.yml` |
| [ ] | NuGet trusted publishing policy | nuget.org, account, Trusted Publishing | repository `bytepunx/signet`, no environment | `release-client-csharp.yml` |
| [ ] | Branch protection on `main` | `scripts/protect-main.sh --apply` | see below | none |
| [ ] | Old repositories archived | `scripts/archive-old-repos.sh --apply` | see below | none |

Client stories may add items; they add their row here.

### Checks to require on `main`

Branch protection on `main` requires these status checks:

| Check | Workflow | Runs |
|-------|----------|------|
| `server` | `ci.yml` | `scripts/test-server.sh` |
| `proto` | `ci.yml` | `scripts/test-proto.sh` |
| `client-go` | `ci.yml` | `scripts/test-client-go.sh` |
| `client-python` | `ci.yml` | `scripts/test-client-python.sh` |
| `client-typescript` | `ci.yml` | `scripts/test-client-typescript.sh` |
| `client-rust` | `ci.yml` | `scripts/test-client-rust.sh` |
| `client-csharp` | `ci.yml` | `scripts/test-client-csharp.sh` |
| `client-elixir` | `ci.yml` | `scripts/test-client-elixir.sh` |
| `check` | `system-flow-check.yml` | Markdown lint and `flai check --strict` |

A job skipped because its component did not change counts as passing, so a pull request
that leaves a component alone is not blocked by its check. To reproduce a failure
locally, run `make test-<component>`, such as `make test-client-go`.

### Protect `main`

```bash
scripts/protect-main.sh            # dry run: prints the body and the command
scripts/protect-main.sh --apply    # sets the protection
scripts/protect-main.sh --show     # prints what is set now
```

It requires the checks above with the branch not required to be up to date, so a branch
rebased by `flai stream sync` need not re-run CI. Administrators are not enforced and no
pull request review is required, because acceptance is local and `flai release --pending`
pushes `main` directly. A required check that never reports blocks a pull request, so
`ci.yml` must skip a path-filtered job at job level (`if:` on the job), never with a
workflow-level `paths:` filter.

## Baselines and the old repositories

Each component's import commit carries a baseline tag at the version its old repository
last released, so the first publish here continues the numbering. Once a release has
published from this repository, archive `signet-clients`, `signet-proto`, and
`signet-admin` on GitHub with a README pointing here, so release-please there can open no
more pull requests.

### Push the baseline tags

Run once, from a checkout of `main` with push rights, before the first `flai release
--pending`. Each tag is annotated, sits on the commit that imported the component, and
names the old repository's tag it continues.

```bash
git tag -a server/v0.11.0 db3b802 -m "Baseline: continues signet v0.11.0"
git tag -a proto/v0.4.0 681c311 -m "Baseline: continues signet-proto v0.4.0"
git tag -a client-go/v0.7.0 4db67f1 -m "Baseline: continues signet-clients go v0.7.0"
git tag -a client-python/v0.5.0 ebd3306 -m "Baseline: continues signet-clients python v0.5.0"
git tag -a client-rust/v0.6.0 206245b -m "Baseline: continues signet-clients rust v0.6.0"
git tag -a client-typescript/v0.9.0 3e0dac7 -m "Baseline: continues signet-clients typescript v0.9.0"
git tag -a client-csharp/v0.6.0 4d9e937 -m "Baseline: continues signet-clients csharp v0.6.0"
git push origin server/v0.11.0 proto/v0.4.0 client-go/v0.7.0 client-python/v0.5.0 \
  client-rust/v0.6.0 client-typescript/v0.9.0 client-csharp/v0.6.0
flai release --pending --dry-run   # only client-elixir may plan from 0.0.0
```

The old root `v*` tags stay as the server's history. These tags start no workflow.

### Archive the old repositories

Order: publish a release from this repository, confirm its workflow succeeded and the
artifact is in the registry, then run:

```bash
scripts/archive-old-repos.sh            # dry run: prints each step
scripts/archive-old-repos.sh --apply    # updates the READMEs, then archives
```

For `bytepunx/signet-clients`, `bytepunx/signet-proto`, and `bytepunx/signet-admin` it
replaces `README.md` with a pointer to `bytepunx/signet` and the `<component>/vX.Y.Z` tag
scheme, then archives the repository; one already archived is skipped, and one whose README
update fails is not archived. To undo, unarchive it under Settings, Danger Zone.

## If a release workflow fails

The tag is already pushed and is the version. Fix the workflow on `main`, then re-run the
failed workflow run from the Actions page; do not delete or move the tag. If the artifact
itself is wrong, accept the fix and publish again: the next `flai release --pending` cuts
the next patch.
