---
id: T-0014
type: task
nature: improvement
title: Split server/design/draft.md into topic files under design/system and index them
status: done
parent: S-0008
owner: alex
created: 2026-10-24T23:00:28Z
updated: 2026-11-03T18:38:32Z
transitions:
  - to: ready
    at: 2026-11-03T18:32:42Z
    by: agent-S-0008
  - to: in-progress
    at: 2026-11-03T18:32:42Z
    by: agent-S-0008
  - to: done
    at: 2026-11-03T18:38:32Z
    by: agent-S-0008
stream: S-0008
tags: [design, documentation]
touches: [design/system/README.md, design/system/storage.md, design/system/encryption-and-unsealing.md, design/system/identity-and-policy.md, design/system/api.md, design/system/gitops-and-sops.md, design/system/audit.md, design/system/admin-access.md, design/system/deployment.md]
usage:
  source: log
  seconds: 350
  estimated: true
  priced_by: harness
  models:
    - model: claude-opus-5-5
      input: 54
      output: 22120
      cache_read: 4006639
      cache_write: 82593
      cost: 1.8041
---
# T-0014 Split server/design/draft.md into topic files under design/system and index them

## Work

Read `server/design/draft.md` (1282 lines, sections 1 to 12 plus the architecture summary and decision matrix) and write eight living-design documents under `design/system/`, each with front matter (`title`, `updated`, `status`, `topics`) and describing what is, not the draft's deliberation:

| File | Draft sections |
|------|----------------|
| `storage.md` | 2 Storage backend and schema, 7 Persistent storage |
| `encryption-and-unsealing.md` | 5 Envelope encryption, AAD, KEK and master key rotation; 6 Unsealing mechanisms, key-check value, sealing |
| `identity-and-policy.md` | 4 Client authentication via SPIFFE/SPIRE, authorization flow, policy model; 8 Node identity |
| `api.md` | The RPC surface: workload, admin, and GitOps services, from sections 4, 10, and 12 and the architecture summary |
| `gitops-and-sops.md` | 12 GitOps/SOPS integration: repository format, key lifecycle, environments, sync flow, webhook, bundle push, PutServiceConfig and PatchServiceConfig |
| `audit.md` | 11 Audit logging and GitOps write coverage |
| `admin-access.md` | 10 Admin authorization, operator CLI access during unseal, in-cluster admin access, local development |
| `deployment.md` | 1 Why not Kubernetes Secrets (as context), 3 Network isolation, 9 PKI, 10 Deployment order |

Leave out alternatives and reasoning that only matter as the "why": those go into the ADRs of the next task, and each topic file gets a `## Decisions` list the ADR task fills with links. Mark the TPM/vTPM unseal as not implemented, as the draft does. Add a row per file to `design/system/README.md`. Do not delete `draft.md` here; the retire task does, once the ADRs and reviews are in place. Give each file `topics` that name the component and subject (for instance `server`, `storage`; `server`, `encryption`, `unseal`) so `flai check --strict` reports no `doc.topics` warning.

## Done when

- The eight files exist under `design/system/` with front matter and `topics`, and `design/system/README.md` lists each with what it covers.
- Every heading of `draft.md` sections 1 to 12 is covered by one of the files, or named in the task's log as deliberately left for an ADR.
- `make lint-md` and `flai check --strict` pass on the worktree.

## Notes

First layer; runs together with the review move. The file names are the planner's; keep them unless a better split appears while reading, and then rename the story's touches with `flai touches` too.
