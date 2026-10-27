---
id: T-0028
type: task
nature: improvement
title: "Server: generate gen/ from ../proto/proto"
status: backlog
parent: S-0020
owner: alex
created: 2026-10-27T22:25:29Z
updated: 2026-10-27T22:25:29Z
transitions: []
stream: S-0020
tags: [server, proto]
touches: [server/buf.gen.yaml]
---
# T-0028 Server: generate gen/ from ../proto/proto

## Work

Replace the `module: buf.build/bytepunx/signet-proto:v0.4.0` input in `server/buf.gen.yaml` with `directory: ../proto/proto` and drop the comment that explains the pin. Keep the managed-mode options and the plugin pins as they are. Run `buf generate` in `server/` and diff `server/gen/` against what the `v0.4.0` pin produced: it must be empty. The server README and `server/Makefile` already say only `buf generate`, so they need no change; `server/CLAUDE.md` belongs to S-0017 and `server/.github/workflows/ci.yml` to S-0006 and S-0007.

## Done when

- `server/buf.gen.yaml` names `directory: ../proto/proto` as its only input.
- `buf generate` in `server/` leaves `git diff server/gen/` empty.
- `go test ./...` in `server/` passes.

## Notes
