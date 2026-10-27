---
id: T-0058
type: task
nature: improvement
title: Rewrite the Go client README install and publish sections for the new path and tags
status: backlog
parent: S-0011
owner: alex
created: 2026-10-27T22:52:01Z
updated: 2026-10-27T22:52:01Z
transitions: []
stream: S-0011
tags: [client-go]
touches: [client-go/README.md]
after: [T-0056, T-0057]
---
# T-0058 Rewrite the Go client README install and publish sections for the new path and tags

## Work

Edit `client-go/README.md`. It waits for T-0056 (the module path it names) and T-0057 (the workflow it names).

- Install: `go get github.com/bytepunx/signet/client-go@vX.Y.Z`, and every import and `go get` line in the Usage section on the new path.
- Publish: replace the release-please and `publish-go.yml` paragraph. State that `flai release --pending` pushes the tag `client-go/vX.Y.Z`, which is the tag form Go requires for a module in a subdirectory, so the tag alone makes the version resolvable; that `release-client-go.yml` runs on that tag, regenerates and tests, asks `proxy.golang.org` to index the version, and creates the GitHub Release; and that the pkg.go.dev link uses the new path.
- Say plainly that the module path changed from `github.com/bytepunx/signet-clients/go`, that this breaks importers of the old path, and that they must update their imports. Link to `docs/operators/release.md` for the release steps.
- Fix the opening paragraph's references to the old repositories and the BSR pin only to the extent that they are now untrue; the pin itself stays until S-0020.
- Leave `CHANGELOG.md` alone: the ADR retires it, and the GitHub Release replaces it.

## Done when

- The README names the module path `github.com/bytepunx/signet/client-go`, the tag form `client-go/vX.Y.Z`, and `release-client-go.yml`.
- No command or import in the README uses `signet-clients/go`, other than the one sentence about the old path.
- The markdown lint passes, and `flai check --strict` is clean.

## Notes
