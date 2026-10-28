---
id: T-0049
type: task
nature: feature
title: TypeScript client test script in Docker
status: done
parent: S-0007
owner: alex
created: 2026-10-27T22:26:28Z
updated: 2026-10-27T23:45:57Z
transitions:
  - to: ready
    at: 2026-10-27T23:04:51Z
    by: agent-S-0007
  - to: in-progress
    at: 2026-10-27T23:04:51Z
    by: agent-S-0007
  - to: done
    at: 2026-10-27T23:45:57Z
    by: agent-S-0007
stream: S-0007
tags: [client-typescript, ci]
touches: [scripts/test-client-typescript.sh]
after: [T-0042]
usage:
  source: log
  seconds: 2466
  estimated: true
  priced_by: harness
  models:
    - model: claude-sonnet-5-5
      input: 26
      output: 4465
      cache_read: 1245751
      cache_write: 52902
      cost: 0.333
---
# T-0049 TypeScript client test script in Docker

## Work

Add `scripts/test-client-typescript.sh`, POSIX `sh` under `set -eu`, modelled on `scripts/test-client-elixir.sh`: run `scripts/buf-generate.sh client-typescript` and fail if `git diff --quiet client-typescript/src/gen` finds drift (those stubs are committed), then with `client-typescript/` mounted in a pinned `node` LTS image run `npm ci`, `npm run build`, and `npm test` (`package.json`). Keep `node_modules/` in a named volume or the gitignored folder, not root-owned. Extra arguments run instead of `npm test`.

It waits for `scripts/buf-generate.sh` (T-0042).

## Done when

- `scripts/test-client-typescript.sh` exits 0 on a clean checkout and runs the compiled `dist/**/*.test.js` tests.
- Editing a file under `client-typescript/src/gen/` makes it fail with the diff.
- Nothing but Docker is installed on the host, and `sh -n` is clean.

## Notes

Layer 2 of the plan. `package.json` still names the old repository; story S-0013 changes it.
