---
id: T-0051
type: task
nature: feature
title: "C# client test script in Docker"
status: done
parent: S-0007
owner: alex
created: 2026-10-27T22:26:34Z
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
tags: [client-csharp, ci]
touches: [scripts/test-client-csharp.sh]
after: [T-0042]
usage:
  source: log
  seconds: 2466
  estimated: true
  priced_by: harness
  models:
    - model: claude-sonnet-5-5
      input: 33
      output: 5683
      cache_read: 1585432
      cache_write: 67326
      cost: 0.4238
---
# T-0051 C# client test script in Docker

## Work

Add `scripts/test-client-csharp.sh`, POSIX `sh` under `set -eu`, modelled on `scripts/test-client-elixir.sh`: run `scripts/buf-generate.sh client-csharp` (stubs under `gen/` are gitignored), then with `client-csharp/` mounted in the `mcr.microsoft.com/dotnet/sdk:8.0` image (the project targets `net8.0`, pinned to a patch tag) run `dotnet test SignetClient.sln`, with `DOTNET_CLI_HOME` and the NuGet cache in a writable path so no root-owned `bin/` or `obj/` is left in the working tree (they are gitignored). Extra arguments run instead of `dotnet test`.

It waits for `scripts/buf-generate.sh` (T-0042).

## Done when

- `scripts/test-client-csharp.sh` exits 0 on a clean checkout and runs the tests under `client-csharp/SignetClient.Tests/`.
- A failing test makes it exit non-zero.
- Nothing but Docker is installed on the host, and `sh -n` is clean.

## Notes

Layer 2 of the plan.
