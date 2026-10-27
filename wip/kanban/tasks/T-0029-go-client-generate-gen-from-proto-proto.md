---
id: T-0029
type: task
nature: improvement
title: "Go client: generate gen/ from ../proto/proto"
status: backlog
parent: S-0020
owner: alex
created: 2026-10-27T22:25:31Z
updated: 2026-10-27T22:25:31Z
transitions: []
stream: S-0020
tags: [client-go, proto]
touches: [client-go/buf.gen.yaml, client-go/README.md]
---
# T-0029 Go client: generate gen/ from ../proto/proto

## Work

Replace the pinned BSR module input in `client-go/buf.gen.yaml` with `directory: ../proto/proto` and rewrite its comment: the schema is the one in this repository, so there is nothing to bump. Keep `go_package_prefix` as S-0011 leaves it. In `client-go/README.md`, rewrite the opening paragraph (about line 4, "pinned") and the regenerate section (about lines 135 to 145, which says it pulls from `buf.build/bytepunx/signet-proto`) so they say `buf generate` reads `../proto/proto` and a schema change and the code that uses it go in one pull request. Run `buf generate` and diff `client-go/gen/` against the output of the `v0.4.0` pin.

## Done when

- `client-go/buf.gen.yaml` names `directory: ../proto/proto` as its only input.
- `buf generate` in `client-go/` leaves `git diff client-go/gen/` empty.
- `go test ./...` in `client-go/` passes.
- `client-go/README.md` no longer tells readers to bump a BSR pin.

## Notes
