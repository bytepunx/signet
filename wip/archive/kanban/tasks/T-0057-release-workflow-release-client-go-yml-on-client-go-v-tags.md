---
id: T-0057
type: task
nature: feature
title: "Release workflow release-client-go.yml on client-go/v* tags"
status: done
parent: S-0011
owner: alex
created: 2026-10-27T22:51:54Z
updated: 2026-11-06T08:28:21Z
transitions:
  - to: ready
    at: 2026-11-06T08:24:46Z
    by: agent-S-0011
  - to: in-progress
    at: 2026-11-06T08:24:46Z
    by: agent-S-0011
  - to: done
    at: 2026-11-06T08:28:21Z
    by: agent-S-0011
stream: S-0011
tags: [client-go, ci]
touches: [".github/workflows/release-client-go.yml"]
usage:
  source: log
  seconds: 215
  estimated: true
  priced_by: harness
  models:
    - model: claude-fable-5-1
      input: 187
      output: 3501
      cache_read: 775124
      cache_write: 21922
      cost: 0.7799
---
# T-0057 Release workflow release-client-go.yml on client-go/v* tags

## Work

Write `.github/workflows/release-client-go.yml`, modelled on `publish-go.yml` of the old signet-clients repository, with the shape `design/system/release.md` gives for client-go.

- Trigger on `push` of tags `client-go/v*` only, with `permissions: contents: write`.
- Derive the version from the tag by stripping `client-go/`; it names the module version, so no file is written.
- Check out with the tag's full history and tags. Set up Go from `client-go/go.mod`.
- Regenerate the stubs and verify them, then `go vet ./...` and `go test ./...` in `client-go/`. Call the shared buf generate and Go client test scripts S-0007 adds under `scripts/` (`scripts/test-client-go.sh` runs what CI runs), rather than inlining the commands; if the script's name differs, use the name S-0007 settled on.
- Ask `proxy.golang.org` to index `github.com/bytepunx/signet/client-go@vX.Y.Z` with `GOPROXY=https://proxy.golang.org go list -m` for that version, retrying a few times, since the proxy may need a moment to see the tag.
- Create the GitHub Release named by the tag with `gh release create`, its notes from the tag annotation and the commits since the previous `client-go/v*` tag, as the other release workflows do. Never commit to `main`.

This task waits for nothing in the story: it names the module path as text, and the new path is written by the sibling task. S-0007, which the story waits for, supplies the scripts it calls.

## Done when

- The workflow triggers on `client-go/v*` only, and on no branch or pull request.
- It verifies (stubs, vet, tests) before it indexes or releases, and a failed step stops the rest.
- It requests the module version from the proxy and creates a GitHub Release named by the tag, with notes from the tag annotation.
- `flai check --strict` and the markdown lint pass, and the file is valid workflow YAML (`actionlint`, or a parse, where available).

## Notes
