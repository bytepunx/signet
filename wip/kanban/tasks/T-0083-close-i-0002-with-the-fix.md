---
id: T-0083
type: task
nature: improvement
title: Close I-0002 with the fix
status: backlog
parent: S-0021
owner: alex
created: 2026-10-27T22:53:00Z
updated: 2026-10-27T22:53:00Z
transitions: []
stream: S-0021
tags: [scripts]
touches: [design/issues/I-0002-scripts-test-client-elixir-sh-rebuilds-the-toolchain-image-on-every-run-and-fails-when-the-registry-stalls.md, design/issues/summary.md]
after: [T-0077]
---
# T-0083 Close I-0002 with the fix

## Work

Waits for T-0077 because the issue can only be closed once the scripts use the fix. Run `scripts/test-client-elixir.sh` twice to confirm the second run skips the image build, then close the issue with `flai issue close I-0002 --reason` saying that the toolchain image is built only when missing or when the hash of client-elixir/Dockerfile changed, and which scripts and test hold it. Tick the story's criteria with `flai criteria tick`.

## Done when

- I-0002 is closed and its reason names scripts/ensure-client-elixir-image.sh and its test.
- design/issues/summary.md no longer lists I-0002.
- Both of S-0021's acceptance criteria are ticked.

## Notes
