---
id: T-0079
type: task
nature: remediation
title: Describe admin/ as part of the mono-repo and delete admin/CLAUDE.md
status: backlog
parent: S-0017
owner: alex
created: 2026-10-27T22:52:51Z
updated: 2026-10-27T22:52:51Z
transitions: []
stream: S-0017
tags: [admin, docs]
touches: [admin/CLAUDE.md, admin/README.md]
---
# T-0079 Describe admin/ as part of the mono-repo and delete admin/CLAUDE.md

## Work

`admin/CLAUDE.md` predates the root `CLAUDE.md`. Delete it after moving what is still true into `admin/README.md`.

- Retitle `admin/README.md` and its first paragraph so they call the component `admin/` (the signet admin UI) and link `server/` in this repository, not `github.com/bytepunx/signet`.
- Replace `## Related repos` with `## Related components`: `server/` (the service this UI fronts) and `proto/` (the schema, published as `buf.build/bytepunx/signet-proto`), as relative links.
- Carry over from `admin/CLAUDE.md` the rule that matters: the UI consumes the `AdminService`/`GitOpsService` gRPC surface only, with no direct database access and no duplicated business logic, and a backend capability the UI needs but the server lacks is built in `server/`, not worked around here. Put it under a `## Relationship to server` heading, worded for a mono-repo ("a separate deployable frontend/BFF, not part of the server's Go module").
- Replace the "Design documentation" paragraph: design lives in the root `design/`, as the root `CLAUDE.md` says.
- Keep the status as it is (pre-implementation). S-0018 decides the sub-project's place; do not pre-empt it.
- Delete `admin/CLAUDE.md`.

## Done when

- `admin/CLAUDE.md` does not exist.
- `grep -n "signet-admin\|signet-clients\|Related repos\|github.com/bytepunx" admin/README.md` finds nothing.
- `flai check --strict` and the markdown lint pass.

## Notes

Layer 1: it shares no path with another task.
