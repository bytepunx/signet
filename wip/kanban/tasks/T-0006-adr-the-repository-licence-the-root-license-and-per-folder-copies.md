---
id: T-0006
type: task
nature: research
title: "ADR: the repository licence, the root LICENSE, and per-folder copies"
status: backlog
parent: S-0001
owner: alex
created: 2026-10-24T22:53:11Z
updated: 2026-10-24T22:53:32Z
transitions: []
stream: S-0001
tags: [adr, licence]
touches: [design/adrs/README.md]
after: [T-0004]
---
# T-0006 ADR: the repository licence, the root LICENSE, and per-folder copies

## Work

Read `proto/LICENSE`, the licence fields in `client-python/pyproject.toml`, `client-rust/Cargo.toml`, `client-typescript/package.json`, and `client-csharp/*.csproj`, and the signet-clients `LICENSE` on GitHub (Apache-2.0). The server never carried a licence. Decide the licence of the repository, the root `LICENSE` file to add, and whether each sub-project keeps a per-folder copy (packages published to registries ship the file next to their metadata) or only the root file. Record it with `flai adr new --commit` from the story's worktree. Say in Consequences what S-0007 must add. Do not add the `LICENSE` file in this story: S-0007 does.

## Done when

- One ADR under `design/adrs/`, status `accepted`, names the licence, the root `LICENSE` to add, and the per-folder policy, with the losing alternatives each given one line.
- `design/adrs/README.md` lists it and `flai check --strict` is clean.

## Notes

Waits for T-0004 only because the ADR tasks share `design/adrs/README.md`, so they run one after another; the licence decision itself depends on none of the others.
