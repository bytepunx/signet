---
id: S-0019
type: story
nature: feature
title: Smoke-test echo images built from the clients' examples
status: backlog
parent: E-0002
owner: arobson
created: 2026-10-27T19:47:50Z
updated: 2026-11-06T07:54:48Z
transitions: []
tags: []
topics: [ci, github]
touches: [".github/workflows/smoke-images.yml", scripts/build-echo-images.sh, scripts/smoke.sh, scripts/README.md, docs/operators/smoke-tests.md, docs/operators/index.md]
after: [S-0020]
agent:
  harness: claude-code
  model: claude-sonnet-5-5
forecast:
  duration: 2h
  delivery: 2026-11-07T03:23:00Z
  basis: "Its own forecast of 2h; 7th in the pull order with an in-progress limit of 2, behind S-0014, S-0015, S-0020, S-0011, S-0016, S-0021, S-0017 and S-0018."
  by: flai
  at: 2026-11-06T07:54:48Z
---
# S-0019 Smoke-test echo images built from the clients' examples

## Goal

The external signet-smoke-test harness pulls `ghcr.io/bytepunx/signet-echo-<language>:latest` images built from each client's `examples/echo`. The old signet-clients `smoke-images.yml` built them on every push; the mono-repo rebuilds each image when its client folder changes on `main`, and `scripts/smoke.sh` documents how to run the harness against them.

## Acceptance criteria
- [ ] `smoke-images.yml` builds and pushes the image for each client whose folder changed on `main`, with the Elixir client added once it has an echo example
- [ ] `scripts/smoke.sh` runs or points at the harness and says what it needs

## Tasks
- T-0071 Script that builds the echo image of a client
- T-0078 smoke-images.yml: build and push each client's echo image when its folder changes on main
- T-0081 scripts/smoke.sh builds the echo images and runs or points at the harness
- T-0086 Operator guide for the smoke-test images and the harness

## Notes

Only if the harness is still in use; the operator confirms before this is refined to ready.

### Planning

Touches. Declared and kept: `.github/workflows/smoke-images.yml`. Layout added `scripts/build-echo-images.sh` (one script so the workflow and `make smoke` build the images alike, as `tooling.md` asks), `scripts/smoke.sh` (criterion 2) and `scripts/README.md` (its table lists every script). Design added `docs/operators/smoke-tests.md` and `docs/operators/index.md` (the documentation convention asks for operator coverage of what an operator must set; a new file avoids `docs/operators/release.md`, which S-0006, S-0007, and S-0008 edit). `flai touches suggest` found nothing: the declared path has no history. No folder touch is kept.

After. `S-0020` only. It moves the python, rust, and csharp echo Dockerfiles' build context to the repository root, so `scripts/build-echo-images.sh` must read the Dockerfiles as S-0020 leaves them, and the workflow's path filter must include `proto/proto/**` from then on. S-0020 already follows S-0005, S-0007, and S-0011, so the Go client's changed module path is on main too. Elixir has no `examples/echo`, so criterion 1's Elixir clause is met by the script refusing it and the workflow naming where it is added; no story is waited for.

Layers of tasks:

| Layer | Tasks |
|-------|-------|
| 1 | T-0071 build-echo-images.sh |
| 2 | T-0078 smoke-images.yml, T-0081 smoke.sh and scripts/README.md (no path in common) |
| 3 | T-0086 operator guide |

Forecast. 2h, adjusted up from flai's 6m (3 done stories of size 3). Five local image builds with remote buf plugins dominate; the rest is small files. The delivery date flai gave is not reliable for the same reason, and the story waits on S-0020 (forecast 2026-10-28T12:40:00Z), so delivery is that plus 2h.

Cost of delay. Not worked out: the story and E-0002 have no inputs. Asked on the plan thread. The harness's use is also unconfirmed, as the note above says.
