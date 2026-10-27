---
id: T-0043
type: task
nature: improvement
title: "Document the server/v* tag form, image tags, chart version, and install.sh"
status: backlog
parent: S-0009
owner: alex
created: 2026-10-27T22:26:09Z
updated: 2026-10-27T22:26:09Z
transitions: []
stream: S-0009
tags: [server, docs]
touches: [server/docs/installation.md, server/README.md, server/deploy/helm/signet/Chart.yaml, server/Makefile]
after: [T-0037, T-0039, T-0040, T-0041]
---
# T-0043 Document the server/v* tag form, image tags, chart version, and install.sh

## Work

Update the server documentation to the new release scheme, after the workflows and the module rename exist so the text names what they do.

- `server/docs/installation.md` and `server/README.md`: the release tag form `server/vX.Y.Z`; images `ghcr.io/bytepunx/signetd` and `docker.io/bytepunx/signet` tagged `vX.Y.Z`, `vX.Y`, and `latest`, and `main` for the main branch; the chart at `oci://ghcr.io/bytepunx/charts/signet` with version `X.Y.Z` and appVersion `vX.Y.Z`; `go install github.com/bytepunx/signet/server/cmd/signet@server/vX.Y.Z`; and `install.sh` installing the newest `server/v*` release, with `SIGNET_VERSION` to pin one. Link to `docs/operators/release.md` for how a release is cut rather than repeating it.
- `server/deploy/helm/signet/Chart.yaml`: comment that `version` and `appVersion` are placeholders replaced at release by `helm package --version --app-version`, and point `home` at the mono-repo's `server/` folder.
- `server/Makefile`: change `go install` or version references only if they name the old module path; leave the targets as they are.
- Do not touch the other READMEs: S-0017 rewrites the old three-repository wording.

This task waits for T-0037 (the new module path), T-0039 (tag, image, chart, and release names), T-0040 (the `main` tag), and T-0041 (`install.sh` behaviour).

## Done when

- `server/docs/installation.md` and `server/README.md` name the tag form, image tags, chart version, `go install` path, and `install.sh` behaviour above, and no link or command in them uses the old module path.
- `flai check --strict` and the markdown lint are clean.

## Notes
