---
id: T-0085
type: task
nature: improvement
title: "Write the contributors guide: how each component is built and tested"
status: done
parent: S-0017
owner: alex
created: 2026-10-27T22:53:03Z
updated: 2026-11-06T08:15:59Z
transitions:
  - to: ready
    at: 2026-11-06T08:15:56Z
    by: agent-S-0017
  - to: in-progress
    at: 2026-11-06T08:15:56Z
    by: agent-S-0017
  - to: done
    at: 2026-11-06T08:15:59Z
    by: agent-S-0017
stream: S-0017
tags: [docs]
touches: [docs/contributors/index.md, README.md]
after: [T-0070, T-0074, T-0076, T-0079]
usage:
  source: log
  seconds: 3
  estimated: true
  priced_by: estimate
  models: []
---
# T-0085 Write the contributors guide: how each component is built and tested

## Work

Replace the "Start here." stub in `docs/contributors/index.md` (front matter `status: draft`, set it to `active`) with a guide for a contributor to this mono-repo. It waits for the layer-1 tasks because it links their results: the overview (T-0070), the server README (T-0074), the hook install (T-0076), and the admin README (T-0079).

- Layout: one paragraph and a table of the components (`server`, `proto`, `admin`, `client-go`, `client-python`, `client-typescript`, `client-rust`, `client-csharp`, `client-elixir`, `client-erlang`) with their language and a link to each folder's README. Link `design/system/overview.md`; do not copy its diagram.
- Per component: the commands that build, test, and lint it, taken from the component's own Makefile, README, `scripts/test.sh`, and `scripts/test-client-elixir.sh`, with the toolchain it needs. Verify each command against the file it comes from; mark a component that has no test command yet (`admin/`, `client-erlang/`) as such.
- Proto: how to change the schema and regenerate the stubs of the server and clients (`make proto` in `server/`, each client's `buf generate`), as the folders' files say on this branch.
- Hooks: `make -C server install-hooks` and what the hook checks.
- The workflow: the root `CLAUDE.md` and `design/conventions/`, `flai check --strict`, and the release flow in `design/system/release.md`, as links.
- Add to the root `README.md` a line linking this guide and the overview, under its table of contents. Do not describe components there.

## Done when

- `docs/contributors/index.md` tells a contributor how each component is built and tested, with a command or a stated gap for each of the ten.
- The root `README.md` links the guide.
- Every relative link resolves, and the markdown lint and `flai check --strict` pass.

## Notes

Layer 2: it waits for T-0070, T-0074, T-0076, and T-0079, whose files it links and whose commands it repeats.
