---
title: Publishing a release
updated: 2026-10-27
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
is registered again here. Branch protection on `main` requires the `ci.yml` jobs as
status checks. S-0008 drives this setup.

## Baselines and the old repositories

Each component's import commit carries a baseline tag at the version its old repository
last released, so the first publish here continues the numbering. Once a release has
published from this repository, archive `signet-clients`, `signet-proto`, and
`signet-admin` on GitHub with a README pointing here, so release-please there can open no
more pull requests.

## If a release workflow fails

The tag is already pushed and is the version. Fix the workflow on `main`, then re-run the
failed workflow run from the Actions page; do not delete or move the tag. If the artifact
itself is wrong, accept the fix and publish again: the next `flai release --pending` cuts
the next patch.
