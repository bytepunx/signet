---
id: S-0017
type: story
nature: remediation
title: Retire the old three-repository layout from agent instructions and READMEs
status: backlog
parent: E-0002
owner: arobson
created: 2026-10-27T19:47:50Z
updated: 2026-10-27T23:01:46Z
transitions: []
tags: [server]
topics: [docs]
touches: [server/CLAUDE.md, admin/CLAUDE.md, server/README.md, admin/README.md, docs/contributors/index.md, design/system/overview.md, server/githooks/pre-commit, server/Makefile, README.md, proto/README.md, client-csharp/README.md, client-elixir/README.md, client-erlang/README.md, client-go/README.md, client-python/README.md, client-rust/README.md, client-typescript/README.md, server/design/draft.md, server/CHECKLIST.md]
agent:
  harness: claude-code
  model: claude-sonnet-5-5
forecast:
  duration: 1h30m
  delivery: 2026-10-29T14:37:00Z
  basis: "Its own forecast of 1h30m; 11th in the pull order with an in-progress limit of 2, behind S-0007, S-0008, S-0009, S-0010, S-0020, S-0011, S-0012, S-0013, S-0014, S-0015 and S-0016."
  by: flai
  at: 2026-10-27T23:01:46Z
---
# S-0017 Retire the old three-repository layout from agent instructions and READMEs

## Goal

No file in the repository describes signet, signet-proto, signet-clients, and signet-admin as separate repositories. `server/CLAUDE.md` and `admin/CLAUDE.md`, which predate the root `CLAUDE.md` and contradict it, are deleted after anything still true moves into the root conventions' project additions or `docs/contributors/`. `server/README.md`'s protocol section, `admin/README.md`'s related-repos section, and `design/system/overview.md`'s placeholder diagram describe the mono-repo. The server pre-commit hook runs from `server/`.

## Acceptance criteria
- [ ] `grep -r "signet-clients\|signet-proto\|signet-admin"` finds only the BSR module name `buf.build/bytepunx/signet-proto` and historical changelog entries
- [ ] `server/CLAUDE.md` and `admin/CLAUDE.md` are gone and `docs/contributors/index.md` tells a contributor how each component is built and tested
- [ ] `design/system/overview.md` shows the real parts (server, proto, admin, the clients) and how they relate

## Tasks
- T-0070 Replace the placeholder diagram in design/system/overview.md with the real parts
- T-0074 Move server/CLAUDE.md's protocol facts into server/README.md and delete it
- T-0076 Run the server pre-commit hook from server/
- T-0079 Describe admin/ as part of the mono-repo and delete admin/CLAUDE.md
- T-0080 Point the client and proto READMEs at the mono-repo's proto/ folder
- T-0085 Write the contributors guide: how each component is built and tested
- T-0087 Sweep the remaining separate-repository references and verify the grep criterion

## Notes

### Planning

Touches, with their source:

- Declared: `server/CLAUDE.md`, `admin/CLAUDE.md`, `server/README.md`, `admin/README.md`, `docs/contributors/index.md`, `design/system/overview.md`, `server/githooks/pre-commit`. `flai touches suggest` found only the declared files (3 of 214 commits changed them) and co-changes `CLAUDE.md` and `Makefile` at 67%.
- Layout: `server/Makefile`, whose `install-hooks` target sets `core.hooksPath githooks`, a path that is wrong at the mono-repo root; it belongs to the hook task. The root `Makefile` and root `CLAUDE.md` need no change: the files being deleted hold nothing the root `CLAUDE.md` lacks, so no project addition is written.
- Layout: `proto/README.md` and the seven `client-*/README.md`, which link `bytepunx/signet-proto` and `signet-clients` as repositories (grep of the tree).
- Design: `README.md` at the root gets one link to the contributors guide. `server/design/draft.md` and `server/CHECKLIST.md` are the places grep finds the old layout in server docs.
- No folder touch is kept: every touch names a file.

Criterion 1 as written is wider than the story's goal. A tree-wide grep also finds `bytepunx/signet-clients#NN` issue references in source comments of every client and the server, the `signet-admin` Kubernetes ServiceAccount and Service names in the Helm chart, docs, and tests, the Go module path `github.com/bytepunx/signet-clients/go` (S-0011), the `buf.gen.yaml` comments (S-0020), and ADR-0001 and `docs/operators/release.md`, which name the old repositories on purpose. The plan fixes descriptions of the layout and leaves those, listing them in T-0087's Notes. The thread on this story asks the operator to confirm the narrower reading or widen the story.

Overlaps: `client-elixir/README.md` and `client-erlang/README.md` are also S-0006's touches; `buf.gen.yaml` text is S-0020's and is not touched here. No `after` is set: the pull order settles it.

Figures:

- Forecast: flai gave 20m (median 116 s per unit over 3 done stories, size 10). Adjusted to 1h30m: seven tasks, and the contributors guide needs each of ten components' commands checked against its files.
- Cost of delay: not set. The story and E-0002 carry no inputs, and these are the operator's; the thread asks for them.

Layers: T-0070, T-0074, T-0076, T-0079, and T-0080 run together (no shared path); T-0085 and T-0087 follow.
