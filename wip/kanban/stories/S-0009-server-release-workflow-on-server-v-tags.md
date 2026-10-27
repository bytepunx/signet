---
id: S-0009
type: story
nature: feature
title: "Server release workflow on server/v* tags"
status: backlog
parent: E-0002
owner: arobson
created: 2026-10-27T19:47:49Z
updated: 2026-10-27T22:48:44Z
transitions: []
tags: [server]
topics: [release, github]
touches: [".github/workflows/release-server.yml", ".github/workflows/publish-server-main.yml", server/go.mod, server/buf.gen.yaml, server/gen, server/Makefile, server/deploy/helm/signet/Chart.yaml, server/docs/installation.md, server/README.md, install.sh]
after: [S-0007]
forecast:
  duration: 3h
  delivery: 2026-10-28T10:26:00Z
  basis: "Its own forecast of 3h; 4th in the pull order with an in-progress limit of 2, behind S-0006, S-0007 and S-0008."
  by: flai
  at: 2026-10-27T22:48:44Z
---
# S-0009 Server release workflow on server/v* tags

## Goal

A push of `server/vX.Y.Z` builds and publishes everything the old `server/.github/workflows/release.yml` did, with the prefix stripped: CLI binaries for linux and darwin on amd64 and arm64 with checksums, `ghcr.io/bytepunx/signetd` and `docker.io/bytepunx/signet` images tagged `vX.Y.Z`, `vX.Y`, and `latest`, the Helm chart pushed to `oci://ghcr.io/bytepunx/charts` with version `X.Y.Z` and appVersion `vX.Y.Z`, and a GitHub Release named `server/vX.Y.Z` with the binaries attached and notes from the tag annotation and the commits since the previous `server/` tag. A push to `main` that touches `server/` publishes the `main` image tag. The Go module becomes `github.com/bytepunx/signet/server` so `go install github.com/bytepunx/signet/server/cmd/signet@server/vX.Y.Z` works, and `install.sh` resolves the newest `server/v*` release instead of `releases/latest`.

## Acceptance criteria
- [ ] `release-server.yml` triggers on `server/v*` tags only and produces the binaries, images, chart, and GitHub Release described above
- [ ] `publish-server-main.yml` pushes the `main` image only when `server/` changed
- [ ] `server/go.mod` is `github.com/bytepunx/signet/server`, every import and the `go_package_prefix` follow, and `make build` and `make test` pass in `server/`
- [ ] `install.sh` installs the newest `server/v*` release on linux and darwin and ignores other components' releases
- [ ] `server/docs/installation.md` and `server/README.md` name the new tag form, image tags, and chart version
- [ ] A test tag on a branch (for example `server/v0.0.0-test`) run of the workflow is green, then deleted

## Tasks
- T-0037 Rename the Go module to github.com/bytepunx/signet/server
- T-0039 Write release-server.yml: binaries, images, chart, and GitHub Release on server/v* tags
- T-0040 Write publish-server-main.yml: push the main image when server/ changes
- T-0041 install.sh: install the newest server/v* release
- T-0043 Document the server/v* tag form, image tags, chart version, and install.sh
- T-0045 Run the release workflow on a test tag server/v0.0.0-test and delete it

## Notes

A breaking change for anyone importing the old module path; this story is the one that delivers it to `server`.

### Planning

Touches, by source:

- Declared: `.github/workflows/release-server.yml`, `.github/workflows/publish-server-main.yml`, `server/go.mod`, `server/buf.gen.yaml`, `server/Makefile`, `server/deploy/helm/signet/Chart.yaml`, `server/docs/installation.md`, `server/README.md`, `install.sh`, and the folder `server/gen`. All kept.
- Layout: the declared folders `server/cmd` and `server/internal` are replaced by the 56 Go files in them that import `github.com/bytepunx/signet/`, now named by T-0037 (a `grep` of the tree); a story's claim includes its tasks' touches, so nothing is lost.
- Design: the old `server/.github/workflows/release.yml` and `publish.yml`, and E-0002's gap analysis row "Server", set the workflow files and `install.sh`.
- Co-change: `flai touches suggest` listed only the old root files of the pre-migration repository (`server/.github/workflows/*`, `Dockerfile`, `Makefile`, `CHECKLIST.md`), 2 of 3 commits each. The old workflows are retired by S-0006, not here, and the `Dockerfile` needs no change because the workflows pass `context: server`, so none were added.

Folder touch kept: `server/gen`. T-0037 regenerates it with `buf generate` after the `go_package_prefix` change, which can rewrite `secrets_grpc.pb.go` and `admin_grpc.pb.go` as well as the two `.pb.go` files it names, and a task cannot name them until the regeneration shows what changed.

Overlaps to know of, not `after` reasons:

- S-0020 also edits `server/buf.gen.yaml` (inputs) and S-0017 edits `server/README.md`. The pull order sets who goes first; each touches different lines.
- S-0006 deletes the old `server/.github/workflows/*` and release-please files; this story does not touch them.

After: S-0007 only, because T-0039 runs `scripts/test-server.sh`, which S-0007 creates, and the two workflows follow the root `ci.yml` it adds. The credentials in S-0008 are needed for T-0045 only; that task stops and asks if they are missing, so no `after` is set.

Forecast: flai gave 1h (the default for size 18 with no usable history). Adjusted to 3h: six tasks, a 58-file mechanical rename plus code generation, three small files that need care (the metadata tags and the notes), and a real run on GitHub that can need a retry. The delivery moves to 2026-10-28T02:25:00Z with it.

Cost of delay: not set. `flai cod` stopped because neither the story nor E-0002 has the operator's inputs; asked on the plan thread.
