---
id: T-0081
type: task
nature: feature
title: scripts/smoke.sh builds the echo images and runs or points at the harness
status: backlog
parent: S-0019
owner: alex
created: 2026-10-27T22:52:56Z
updated: 2026-10-27T22:52:56Z
transitions: []
stream: S-0019
tags: [ci]
touches: [scripts/smoke.sh, scripts/README.md]
after: [T-0071]
---
# T-0081 scripts/smoke.sh builds the echo images and runs or points at the harness

## Work

Replace the placeholder body of `scripts/smoke.sh` (`make smoke`). It builds the echo images with `scripts/build-echo-images.sh all --tag local`, then says what the harness needs and either runs it or points at it:

- With `SIGNET_SMOKE_TEST_DIR` set to a checkout of `bytepunx/signet-smoke-test`, it runs that harness against the `local` images, using the entry point the harness's own README gives (read it when writing; if it names none that can take a tag, the script prints the command to run by hand rather than guessing one).
- Without it, it prints what the harness needs (Docker; a k3d cluster with signet and SPIRE, from `bytepunx/kluster`; the harness checkout) and where to get it, and exits 0 after building the images, so `make smoke` stays honest about what ran: the exit code is the build's, and the output says the harness did not run.

Update the `smoke.sh` row in `scripts/README.md` and add one for `build-echo-images.sh`. The task waits for T-0071 because it calls that script. It touches `scripts/README.md`, so it must not run beside another task that does.

## Done when

- `scripts/smoke.sh` with `SIGNET_SMOKE_TEST_DIR` unset builds the five images, prints what the harness needs, and says the harness was not run.
- With `SIGNET_SMOKE_TEST_DIR` set to a checkout, it runs the harness or prints the exact command, per what the harness README allows; the task notes which.
- `make smoke` runs it.
- `scripts/README.md` lists `smoke.sh` with the new behavior and `build-echo-images.sh`.
- The script passes `shellcheck` where it is installed and `scripts/lint-md.sh` passes on `scripts/README.md`.

## Notes

The harness is external and not in this repository; the story's Notes say it is wanted only while the operator still uses it. Do not invent its interface.
