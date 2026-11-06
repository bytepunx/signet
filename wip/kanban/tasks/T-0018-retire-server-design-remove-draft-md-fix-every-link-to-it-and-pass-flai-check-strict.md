---
id: T-0018
type: task
nature: improvement
title: "Retire server/design: remove draft.md, fix every link to it, and pass flai check --strict"
status: done
parent: S-0008
owner: alex
created: 2026-10-24T23:01:17Z
updated: 2026-11-03T18:45:36Z
transitions:
  - to: ready
    at: 2026-11-03T18:44:59Z
    by: agent-S-0008
  - to: in-progress
    at: 2026-11-03T18:44:59Z
    by: agent-S-0008
  - to: done
    at: 2026-11-03T18:45:36Z
    by: agent-S-0008
stream: S-0008
tags: [design, documentation]
touches: [server/design/draft.md, server/README.md, client-go/sops_encrypt.go, design/system/README.md, admin/README.md, client-rust/src/sops_encrypt.rs, design/system/overview.md, server/CHECKLIST.md, server/CLAUDE.md, server/deploy/helm/signet/values.yaml, server/internal/api/gitops.go, server/internal/store/audit_ttl_cockroachdb_integration_test.go]
after: [T-0016, T-0017]
usage:
  source: log
  seconds: 37
  estimated: true
  priced_by: harness
  models:
    - model: claude-opus-5-5
      input: 10
      output: 4151
      cache_read: 751953
      cache_write: 15501
      cost: 0.3386
---
# T-0018 Retire server/design: remove draft.md, fix every link to it, and pass flai check --strict

## Work

Waits for T-0016 and T-0017: the draft goes only once its content lives in the topic files and the ADRs, and the reviews at their new paths no longer link it. Check once more that every section of `server/design/draft.md` is covered by a topic file or an ADR, then `git rm` it, which leaves `server/design/` empty and gone. Sweep the tree for links and mentions: `server/README.md` (the documentation table rows for the design summary and the security findings now point at `design/system/README.md` and `design/security/`), the comment in `client-go/sops_encrypt.go` that cites `design/draft.md` (point it at `design/system/gitops-and-sops.md`), and anything else `grep -rn "design/draft\|security-findings\|server/design" --exclude-dir=wip --exclude-dir=.flai-cache .` still finds. Mentions inside `wip/` items are history and stay. Run `make lint-md` and `flai check --strict` and clear every `doc.topics` warning by giving the file it names `topics`.

## Done when

- `server/design/` does not exist on the story's branch.
- The grep above finds no link to the removed or moved files outside `wip/`.
- `flai check --strict` passes with no `doc.topics` warning, and `make lint-md` passes.

## Notes

Third layer, the last. `design/system/README.md` is touched only if the sweep finds the index lacking a file; the other three paths come from the grep the planner ran on 2026-10-24.
