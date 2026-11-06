---
id: S-0008
type: story
nature: improvement
title: Fold the server design and security reviews into design/ and retire server/design
status: review
parent: E-0001
owner: alex
created: 2026-10-24T22:33:36Z
updated: 2026-11-03T18:46:38Z
transitions:
  - to: ready
    at: 2026-10-24T22:33:48Z
    by: claude
  - to: in-progress
    at: 2026-11-03T18:31:48Z
    by: agent-S-0008
  - to: review
    at: 2026-11-03T18:46:31Z
    by: agent-S-0008
tags: [server, design, documentation, security]
topics: [design, security, server, documentation]
touches: [server/design/draft.md, server/design/security-findings.md, server/design/security-review-2026-08-31.md, design/system/README.md, design/system/storage.md, design/system/encryption-and-unsealing.md, design/system/identity-and-policy.md, design/system/api.md, design/system/gitops-and-sops.md, design/system/audit.md, design/system/admin-access.md, design/system/deployment.md, design/adrs, design/adrs/README.md, design/issues, design/issues/summary.md, design/security/README.md, design/security/security-findings.md, design/security/security-review-2026-08-31.md, design/README.md, server/README.md, client-go/sops_encrypt.go, admin/README.md, client-rust/src/sops_encrypt.rs, design/system/overview.md, server/CHECKLIST.md, server/CLAUDE.md, server/deploy/helm/signet/values.yaml, server/internal/api/gitops.go, server/internal/store/audit_ttl_cockroachdb_integration_test.go, design/issues/I-0006-n-1-syncbundle-buffers-the-whole-stream-in-memory-before-size-and-scope-checks.md, design/issues/I-0007-n-2-audit-writes-serialize-the-read-path-through-one-mutex-and-a-synchronous.md, design/issues/I-0008-n-3-watchserviceconfig-authorizes-once-at-subscribe-and-never-re-checks.md, design/issues/I-0009-n-4-patchserviceconfig-has-no-size-or-op-count-bounds-and-json-patch-copy.md, design/issues/I-0010-n-5-webhook-rate-limiter-is-a-single-global-bucket.md, design/issues/I-0011-n-6-extracttargz-bounds-total-bytes-but-not-file-count.md, design/issues/I-0012-n-7-signet-secret-set-and-rm-accept-dot-dot-segments-allowing-local-path.md, design/issues/I-0013-n-8-write-path-audit-entries-omit-peerip.md, design/issues/I-0014-n-9-legacy-nil-aad-decrypt-fallback-is-silent-and-open-ended.md, design/issues/I-0015-a-config-created-by-putserviceconfig-is-replaced-by-its-first-git-sync-contrary.md, design/issues/I-0016-flai-check-finds-roles-baseline-outside-the-story-at-close-out.md]
usage:
  source: log
  seconds: 905
  priced_by: harness
  turns:
    - day: 2026-11-03
      ceremony: 2
      test_runs: 1
      hand_edits: 2
      work: 55
  models:
    - model: claude-haiku-5-5
      input: 18
      output: 8859
      cache_read: 298491
      cache_write: 56926
      cost: 0.0145
    - model: claude-opus-5-5
      input: 216
      output: 88937
      cache_read: 16109325
      cache_write: 332079
      cost: 7.2535
  strategic:
    - kind: planner
      seconds: 1572
      estimated: true
      models:
        - model: claude-fable-5-1
          input: 1730
          output: 1349
          cache_read: 4725904
          cache_write: 201483
          cost: 6.9792
    - kind: orchestrator
      seconds: 38
      estimated: true
      models:
        - model: claude-opus-5-5
          input: 24
          output: 181
          cache_read: 667014
          cache_write: 104785
          cost: 0.4805
forecast:
  duration: 8h
  delivery: 2026-10-25T08:00:00Z
  basis: "A 1282-line draft split into eight topic files, about ten ADRs written and dated, two reviews moved and their open findings recorded as issues, and every link fixed; no story history yet, so sized by hand against the default 1h."
  by: planner-S-0008
  at: 2026-10-24T23:00:17Z
---
# S-0008 Fold the server design and security reviews into design/ and retire server/design

## Goal

The living design of signet is in `design/system/`, split by topic and kept current, and the security reviews are recorded where `design/` expects them, so `server/design/` is empty and removed and agents find the design where the conventions say it is.

## Acceptance criteria
- [x] `server/design/draft.md` is split into topic files under `design/system/` (at least: storage, encryption and unsealing, identity and policy, API surface, GitOps and SOPS, audit, admin access, deployment), each with front matter and `topics`, and `design/system/README.md` indexes them.
- [x] Each decision in `draft.md` that is architectural has an accepted ADR under `design/adrs/` dated to when it was taken, with the living design linking to it.
- [x] `server/design/security-review-2026-08-31.md` and `security-findings.md` are moved under `design/` (an issues or security section), with every open finding recorded in `design/issues/` and `summary.md` current.
- [x] `server/design/` no longer exists; nothing links to it.
- [x] `flai check --strict` passes with no `doc.topics` warnings.

## Tasks
- T-0014 Split server/design/draft.md into topic files under design/system and index them
- T-0015 Move the two security reviews to design/security with their history and index the folder
- T-0016 Record each architectural decision of draft.md as an accepted, dated ADR and link it from the living design
- T-0017 Record every open security finding as an issue under design/issues and keep summary.md current
- T-0018 Retire server/design: remove draft.md, fix every link to it, and pass flai check --strict

## Notes

Keep history: `git mv` where a file moves whole. The design sections should describe what is, not the draft's deliberation; deliberation that still matters goes into the ADR's context.

### Planning

Planned by planner-S-0008 on 2026-10-24. `flai touches suggest S-0008` had nothing to start from (the story declared no touches), so every touch below comes from the criteria, the design layout, or a grep of the tree. The tag `server` is the component the story delivers to: the design is the server's, and the one `client-go` touch is a code comment.

Touches and where each came from:

- `server/design/draft.md`, `server/design/security-findings.md`, `server/design/security-review-2026-08-31.md`: criteria 1, 3, and 4; the files the story splits, moves, and removes.
- `design/system/README.md` and the eight topic files `storage.md`, `encryption-and-unsealing.md`, `identity-and-policy.md`, `api.md`, `gitops-and-sops.md`, `audit.md`, `admin-access.md`, `deployment.md`: criterion 1 names the eight topics; the file names are the planner's, one per topic, kebab-case as `documentation.md` asks. `design/system/overview.md` is left out: S-0001's T-0002 writes it, and the index in `README.md` is enough to reach the topic files; the story's agent adds it if a link from the overview is needed.
- `design/adrs` (folder) and `design/adrs/README.md`: criterion 2. The folder touch stays because `flai adr new` numbers each ADR when it is made, across every worktree and branch, so no task can name the files; the index is the one file named. The folder overlaps S-0001's `design/adrs` touch, so this story is held while S-0001 is in progress; that is the pull order's business, not an `after`.
- `design/issues` (folder) and `design/issues/summary.md`: criterion 3. The folder touch stays because `flai issue new` numbers each `I-nnnn` file when it is made; `summary.md` is regenerated by every issue command.
- `design/security/README.md`, `design/security/security-findings.md`, `design/security/security-review-2026-08-31.md`, `design/README.md`, `CLAUDE.md`: criterion 3 allows "an issues or security section"; the planner's assumption is a new `design/security/` folder, with its README as `documentation.md` requires, listed in `design/README.md` and in the layout table of `CLAUDE.md`. The two reviews keep their names so `git mv` keeps their history.
- `server/README.md` and `client-go/sops_encrypt.go`: criterion 4; a grep of the tree outside `wip/` finds these two linking to `design/draft.md` or `design/security-findings.md` (a documentation table and a code comment). Mentions inside `wip/` items are history and stay.

Figures:

- Forecast: `flai forecast` gave 1h, the `planning.default_duration` for size 5 with no done stories to learn from. Adjusted to 8h, delivery 2026-10-25T08:00:00Z (the default's start, third in the pull order, plus 8h): the draft is 1282 lines across twelve sections to be rewritten as living design, not copied; its decision matrix yields about ten ADRs each needing context and a date from `git log`; the two reviews hold nine N-series findings whose fix status must be checked against the code before each open one is recorded as an issue; and the link sweep ends with `flai check --strict`.
- Cost of delay: `flai cod` could not work out a value because neither S-0008 nor E-0001 has inputs. Asked on TH-0003, recommending `time_lost_per_cycle: 4h` on E-0001 as TH-0001 does. The value is written here once the inputs are set.
