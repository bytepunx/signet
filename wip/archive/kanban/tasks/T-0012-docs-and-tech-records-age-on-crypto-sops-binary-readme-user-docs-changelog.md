---
id: T-0012
type: task
nature: feature
title: "Docs and tech records: age on :crypto, sops binary, README, user docs, changelog"
status: cancelled
parent: S-0003
owner: arobson
created: 2026-10-27T21:31:23Z
updated: 2026-10-27T22:16:04Z
transitions:
  - to: cancelled
    at: 2026-10-27T22:16:04Z
    by: agent-S-0003
stream: S-0003
tags: []
touches: [design/tech/age-elixir.md, design/tech/sops.md, design/tech/README.md, client-elixir/README.md, docs/users/elixir-client.md, client-elixir/CHANGELOG.md]
---
# T-0012 Docs and tech records: age on :crypto, sops binary, README, user docs, changelog

## Work

Record the age implementation (hand-rolled X25519 recipient format on OTP `:crypto`) in `design/tech/age-elixir.md` with alternatives considered, the pinned sops binary (3.13.3, test-only, in the Dockerfile) in `design/tech/sops.md`, both in the tech index; document `Signet.SopsEncrypt.encrypt_for_secret/2` in `client-elixir/README.md` and `docs/users/elixir-client.md` (the SOPS part of the "not yet" sections only; S-0002 owns the restart part), and add the changelog entry.

## Done when

- Tech index lists both files; each has version, used in, why, alternatives.
- README and user docs show the call and say what the document is for (SyncBundle).
- CHANGELOG has the feature line.
- `make lint-md` is clean on the touched files.

## Notes

Leave the coordinated-restarts text for S-0002.
- 2026-10-27T22:16:04Z: moved to cancelled: duplicate of T-0019/T-0020/T-0021: created by a first task-creation attempt whose output parsing failed; never worked
