---
id: T-0020
type: task
nature: remediation
title: scripts/test.sh regenerates client stubs only when they are missing or older than buf.gen.yaml
status: done
parent: S-0004
owner: alex
created: 2026-11-03T18:29:00Z
updated: 2026-11-03T18:30:31Z
transitions:
  - to: ready
    at: 2026-11-03T18:29:01Z
    by: agent-S-0004
  - to: in-progress
    at: 2026-11-03T18:29:01Z
    by: agent-S-0004
  - to: done
    at: 2026-11-03T18:30:31Z
    by: agent-S-0004
stream: S-0004
tags: []
touches: [scripts/test.sh, docs/contributors/index.md]
usage:
  source: log
  seconds: 90
  estimated: true
  priced_by: harness
  models:
    - model: claude-opus-5-5
      input: 16
      output: 4196
      cache_read: 1051488
      cache_write: 15106
      cost: 0.4024
---
# T-0020 scripts/test.sh regenerates client stubs only when they are missing or older than buf.gen.yaml

## Work

The second close-out stopped at the test tier: `buf generate` in `client-rust` got `resource_exhausted: too many requests` from the Buf Schema Registry, so the tests never ran. `scripts/test.sh` regenerated the Python, Rust, and C# stubs on every run. `buf.gen.yaml` pins every input to the stubs (the signet-proto tag and each plugin version), so regenerate only when the stub folder is absent or older than `buf.gen.yaml`, and say which it did. Update the `make test` row of `docs/contributors/index.md` to match.

## Done when

- `scripts/test.sh` runs `buf generate` in a client only when its stub folder is missing or older than its `buf.gen.yaml`.
- The close-out passes its test tier.

## Notes

Recorded as I-0005. S-0002 removes the remote dependency.
