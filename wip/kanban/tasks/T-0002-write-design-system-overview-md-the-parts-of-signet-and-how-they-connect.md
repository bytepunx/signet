---
id: T-0002
type: task
nature: research
title: "Write design/system/overview.md: the parts of signet and how they connect"
status: done
parent: S-0001
owner: alex
created: 2026-10-24T22:52:29Z
updated: 2026-11-03T18:16:57Z
transitions:
  - to: ready
    at: 2026-11-03T18:13:20Z
    by: agent-S-0001
  - to: in-progress
    at: 2026-11-03T18:13:20Z
    by: agent-S-0001
  - to: done
    at: 2026-11-03T18:16:57Z
    by: agent-S-0001
stream: S-0001
tags: [design, docs]
touches: [design/system/overview.md]
usage:
  source: log
  seconds: 217
  estimated: true
  priced_by: harness
  models:
    - model: claude-opus-5-5
      input: 29
      output: 9422
      cache_read: 1746830
      cache_write: 33581
      cost: 0.765
---
# T-0002 Write design/system/overview.md: the parts of signet and how they connect

## Work

Replace the template placeholder in `design/system/overview.md` with the real system. Read `server/design/draft.md`, `server/README.md`, `server/docs/getting-started.md`, `server/deploy/helm`, `server/deploy/spire`, `proto/README.md`, the client READMEs, and `admin/README.md`. Describe `signetd`, the `signet` CLI, the proto schema, the five client libraries, the admin UI, SPIRE and go-spiffe for workload identity, and CockroachDB for storage, with one Mermaid diagram of how they connect and which protocol each edge uses. Keep the Principles section. Link to `server/design/draft.md` for the detail S-0008 will fold in later rather than copying it. Bump `updated`.

## Done when

- `design/system/overview.md` names every part in the story's fifth criterion, with a Mermaid diagram of their connections, and no template text remains.
- The document follows `design/conventions/documentation.md`: front matter with `title`, `updated`, `status`, relative links only, no duplicated paragraphs.
- The markdown lint and `flai check --strict` are clean.

## Notes

Waits for nothing: it describes what exists today, and the ADRs change nothing it says.
