---
id: T-0077
type: task
nature: improvement
title: Use the ensure script in the Elixir test scripts and document it
status: in-progress
parent: S-0021
owner: alex
created: 2026-10-27T22:52:47Z
updated: 2026-11-06T08:02:59Z
transitions:
  - to: ready
    at: 2026-11-06T08:02:59Z
    by: agent-S-0021
  - to: in-progress
    at: 2026-11-06T08:02:59Z
    by: agent-S-0021
stream: S-0021
tags: [scripts]
touches: [scripts/test-client-elixir.sh, scripts/integration-client-elixir.sh, scripts/test.sh, scripts/README.md]
after: [T-0073]
---
# T-0077 Use the ensure script in the Elixir test scripts and document it

## Work

Waits for T-0073 because it calls the script and the test that task adds.

In `scripts/test-client-elixir.sh` and `scripts/integration-client-elixir.sh`, replace the `docker build -q ...` line with a call to `scripts/ensure-client-elixir-image.sh`, and update their header comments (cached by Dockerfile hash, `SIGNET_REBUILD=1` forces a rebuild). In `scripts/test.sh`, run `scripts/test-ensure-client-elixir-image.sh` before the Elixir client's tests. In `scripts/README.md`, add the two new scripts to the table and correct the descriptions of the two Elixir scripts, which no longer build on every run.

## Done when

- Neither Elixir script runs `docker build` itself.
- A second `scripts/test-client-elixir.sh` run prints that the image build was skipped and still passes.
- `scripts/test.sh` runs the ensure script's test.
- `scripts/README.md` lists every script in `scripts/`.
- `flai check --strict` and `flai test` on the changed paths pass.

## Notes
