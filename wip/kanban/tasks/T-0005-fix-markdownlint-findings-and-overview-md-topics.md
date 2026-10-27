---
id: T-0005
type: task
nature: feature
title: Fix markdownlint findings and overview.md topics
status: done
parent: S-0004
owner: alex
created: 2026-10-27T20:55:12Z
updated: 2026-10-27T20:55:20Z
transitions:
  - to: ready
    at: 2026-10-27T20:55:14Z
    by: agent-S-0004
  - to: in-progress
    at: 2026-10-27T20:55:16Z
    by: agent-S-0004
  - to: done
    at: 2026-10-27T20:55:20Z
    by: agent-S-0004
stream: S-0004
tags: []
touches: [server/docs/secrets-workflow.md, design/system/overview.md]
usage:
  source: log
  seconds: 4
  estimated: true
  priced_by: harness
  models:
    - model: claude-sonnet-5
      input: 4
      output: 1077
      cache_read: 180810
      cache_write: 6285
      cost: 0.0721
---
# T-0005 Fix markdownlint findings and overview.md topics

## Work

Add a language to every unlabeled fenced code block and a blank line around every fence missing one in `server/docs/secrets-workflow.md`. Add `topics: [all]` to `design/system/overview.md`'s front matter.

## Done when

- `make lint-md` (or `npx markdownlint-cli2 server/docs/secrets-workflow.md`) reports no findings for the file.
- `flai check --strict` reports no warning about `design/system/overview.md` having no topics.

## Notes
