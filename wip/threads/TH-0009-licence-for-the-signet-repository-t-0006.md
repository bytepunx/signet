---
id: TH-0009
title: Licence for the signet repository (T-0006)
anchor:
  path: wip/kanban/stories/S-0001-decide-proto-distribution-go-module-paths-and-the-release-model-for-the-mono-repo.md
  item: S-0001
status: open
participants: [agent-S-0001]
created: 2026-11-03T18:13:32Z
updated: 2026-11-03T18:13:32Z
---

# TH-0009 Licence for the signet repository (T-0006)

On wip/kanban/stories/S-0001-decide-proto-distribution-go-module-paths-and-the-release-model-for-the-mono-repo.md.

## Entries

### 2026-11-03T18:13:32Z agent-S-0001
T-0006 needs the repository's licence, and licensing the server is yours to choose: `server/` never carried a licence, while `proto/LICENSE` and the old signet-clients repository are Apache-2.0.

Recommendation: Apache-2.0 for the whole repository, one root `LICENSE`, and a per-folder copy only in the sub-projects whose registry package ships its folder (client-csharp, client-python, client-rust, client-typescript, client-erlang, proto); `server/` and `admin/` rely on the root file. Reason: it matches what the clients and proto already publish under, and its patent grant suits a security product.

Alternatives:

- Apache-2.0 at the root only, no per-folder copies (packages would point at the root file through their metadata).
- A different licence for `server/` (for example AGPL-3.0 or a source-available licence) with Apache-2.0 for proto and the clients.

Answer with the licence and the per-folder policy; I write the ADR from it. Everything else in S-0001 goes on meanwhile.
