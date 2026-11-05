---
id: T-0039
type: task
nature: feature
title: "Write release-server.yml: binaries, images, chart, and GitHub Release on server/v* tags"
status: done
parent: S-0009
owner: alex
created: 2026-10-27T22:25:55Z
updated: 2026-10-28T05:55:47Z
transitions:
  - to: ready
    at: 2026-10-28T05:55:34Z
    by: agent-S-0009
  - to: in-progress
    at: 2026-10-28T05:55:34Z
    by: agent-S-0009
  - to: done
    at: 2026-10-28T05:55:47Z
    by: agent-S-0009
stream: S-0009
tags: [server, ci]
touches: [".github/workflows/release-server.yml"]
usage:
  source: log
  seconds: 13
  estimated: true
  priced_by: harness
  models:
    - model: claude-sonnet-5-5
      input: 1
      output: 310
      cache_read: 51749
      cache_write: 2711
      cost: 0.0191
---
# T-0039 Write release-server.yml: binaries, images, chart, and GitHub Release on server/v* tags

## Work

Create `.github/workflows/release-server.yml`, triggered by `push: tags: ["server/v*"]` only, porting `server/.github/workflows/release.yml` with the `server/` prefix stripped and every build step run from `server/`.

- A `version` step derives `VERSION=${GITHUB_REF_NAME#server/}` (for example `v0.12.0`) and `CHART_VERSION=${VERSION#v}`, and every job uses them in place of `github.ref_name`.
- A test job runs `scripts/test-server.sh` (S-0007) before anything is published, and the publishing jobs `need` it.
- CLI binaries: linux and darwin on amd64 and arm64, built with `-X main.version=${VERSION}`, named `signet-<os>-<arch>`, with one merged `sha256sums.txt`, as before; `go-version-file` and the build run in `server/`.
- Images: build with `context: server`, target `signetd`, platforms linux/amd64 and linux/arm64, and push `ghcr.io/bytepunx/signetd` and `docker.io/bytepunx/signet` tagged `vX.Y.Z`, `vX.Y`, and `latest`. A pre-release tag (a version with a `-` suffix, such as `server/v0.0.0-test`) pushes only `vX.Y.Z` and no `vX.Y` or `latest`, so the test-tag run in the last task cannot move `latest`.
- Chart: `helm package server/deploy/helm/signet --version X.Y.Z --app-version vX.Y.Z` and push to `oci://ghcr.io/bytepunx/charts`.
- GitHub Release: named by the full tag `server/vX.Y.Z`, with the binaries and `sha256sums.txt` attached. Notes are the tag annotation (`git tag -l --format='%(contents)'`) followed by the commits since the previous `server/v*` tag (`git describe --tags --abbrev=0 --match 'server/v*' HEAD^`), or all commits when there is none. A pre-release tag marks the release as a pre-release.
- Keep the old `permissions` (`contents: write`, `packages: write`) and the secrets `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN`, which S-0008 registers on this repository.

The old `server/.github/workflows/release.yml` is retired by S-0006, not here.

## Done when

- The workflow triggers on `server/v*` tags only and has jobs for the tests, the binaries, the images, the chart, and the release.
- Version, image, chart, and release names strip the `server/` prefix as the story's goal states.
- A pre-release tag cannot push `latest` or `vX.Y`.
- The file passes `actionlint` if available, and `flai check --strict` and the markdown lint are clean.

## Notes
