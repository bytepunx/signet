# Signet Security & Usability Review — 2026-08-31

**Reviewer:** Fable (second-pass review)
**Scope:** `internal/` (api, auth, crypto, gitops, store, server, unseal, audit),
`cmd/signetd`, `cmd/signet`, and `design/`.
**Relationship to prior work:** `design/security-findings.md` (2026-07-01) is the first
audit; every item there is marked fixed and I did not re-audit those closures line by line.
This pass looks for **new** attack surface, **availability/DoS** exposure, **authorization
consistency**, and the **safety-vs-usability balance** the owner flagged as a concern.

> Convention: `path:line` references point at code as of this review; grep the quoted
> snippet if lines drift. Each entry is written to drop straight into a GitHub issue:
> it has a severity, a location, an impact, a concrete fix, and acceptance criteria.
> "Confidence" notes whether the finding is confirmed by reading or needs a repro/verify.

---

## Severity summary

| ID | Severity | Title |
|----|----------|-------|
| N-1 | High | `SyncBundle` buffers the entire uploaded stream in memory before any size or scope check |
| N-2 | Medium-High | Audit write is a global-mutex synchronous DB round-trip on the hot read path; fail-closed turns an audit slowdown into a secrets outage |
| N-3 | Medium | `WatchServiceConfig` authorizes once at subscribe and never re-checks; revoked access keeps streaming config |
| N-4 | Medium | `PatchServiceConfig` has no document-size / op-count cap; JSON-Patch `copy` allows storage/CPU amplification |
| N-5 | Low-Medium | Webhook rate limiter is a single global bucket — one repo (or attacker) starves delivery for all repos |
| N-6 | Low | `extractTarGz` caps total bytes but not file/inode count |
| N-7 | Low | `signet secret set/rm` accept `..` path segments — local path traversal outside the secrets root |
| N-8 | Low | GitOps/admin write-path audit entries omit `PeerIP`; minor audit-completeness gaps |
| N-9 | Info | Legacy nil-AAD decrypt fallback is silent and open-ended — no migration inventory or deadline |

Usability/safety-balance critiques are in the **U-series** at the end.

---

## N-1 (High) — `SyncBundle` buffers the whole stream in memory before size/scope checks

- **Where:** `internal/api/gitops.go:491-517` (`SyncBundle`): the data-chunk loop does
  `buf.Write(data.Data)` into an unbounded `bytes.Buffer`, then `extractTarGz`. The
  256 MiB cap in `internal/api/bundle.go:17,67-75` applies only to **decompressed**
  extraction — it runs *after* the entire **compressed** stream is already resident in
  `buf`.
- **Reachability:** `SyncBundle` is registered on the **workload mTLS listener**
  (`internal/server/server.go:241,444-447`) and is reachable by **any** holder of a valid
  SPIFFE SVID in the trust domain. `authorizeGitOpsWrite` (`gitops.go:470`) only resolves
  identity; `validateBundleScope` (the namespace/service authorization) runs at
  `gitops.go:524-528`, i.e. *after* full buffering and extraction. So an authenticated but
  otherwise unprivileged workload can drive memory use before any authorization decision.
- **Impact:** A single client streaming an arbitrarily large body (or many 64 KiB chunks)
  exhausts server memory and OOM-kills the process, taking the workload **and** admin
  listeners down with it. gRPC's per-message cap doesn't help — this is unbounded across
  messages.
- **Fix:**
  1. Bound the accumulated compressed bytes (e.g. reject once `buf.Len()` exceeds a
     configured max compressed size), and/or stream chunks straight to a size-capped temp
     file instead of an in-memory buffer.
  2. Move the cheapest authorization signal as early as possible. Full per-file scope needs
     the extracted tree, but a coarse gate (is this identity allowed to push *anything*?)
     could precede buffering for the SPIFFE-scoped path.
  3. Apply the same bound to the admin-token path — an admin token shouldn't be able to OOM
     the process either.
- **Acceptance:** a test that streams > cap bytes to `SyncBundle` returns
  `ResourceExhausted`/`InvalidArgument` with bounded memory, and the process survives.
- **Confidence:** High (confirmed by reading). CWE-770, CWE-400.

---

## N-2 (Medium-High) — Audit write serializes the read path through one mutex + a synchronous durable write

- **Where:** `internal/audit/audit.go:58-82` (`Record` holds `w.mu` across
  `st.WriteAuditLog`, a blocking DB round-trip). Every secret read calls this before
  returning data: `internal/api/secrets.go:81,110,144,370` via `auditOrDeny`.
- **Problem:** All audit writes — and therefore all permitted reads that must audit before
  returning — funnel through a **single process-wide mutex** and each waits for a durable
  DB write. Under concurrent read load, `GetSecret`/`GetServiceBundle`/`WatchSecret` calls
  queue behind one another on the audit mutex regardless of which secret they touch.
- **Interaction with fail-closed (H-3, now default `true`):** when the audit DB is slow or
  briefly unavailable, `auditOrDeny` returns `codes.Unavailable` and the secret is withheld
  (`secrets.go:570-575`). So an audit-path degradation converts directly into a
  **secrets-serving outage** — and because workloads often read secrets at startup, an audit
  hiccup can cascade into a fleet-wide inability to boot. This is the intended security
  posture, but the coupling and the throughput ceiling are worth making explicit and
  tunable.
- **Impact:** (a) throughput ceiling on reads set by audit-write latency; (b) availability
  of the entire secret store is gated on the availability *and speed* of the audit table.
- **Fix (menu — pick per threat model):**
  1. Reduce lock scope: compute the chained HMAC under the mutex, but don't hold the mutex
     across the DB write (needs care to keep the chain ordered — e.g. a single-writer
     goroutine with an ordered queue rather than a shared mutex).
  2. Offer a **bounded, durable async spool** (write-ahead to local disk, confirmed before
     serving) so "durable audit before serve" doesn't mean "synchronous round-trip to a
     possibly-remote DB on every read."
  3. Document the availability coupling loudly and make the fail-closed timeout/behavior
     tunable (a bounded wait, then a deliberate decision) rather than an unbounded block.
- **Acceptance:** a benchmark showing read throughput is no longer bounded by a single
  serialized DB write; documented, tunable behavior when the audit backend is slow.
- **Confidence:** High on the mechanism (confirmed by reading); Medium on real-world impact
  magnitude (depends on DB latency — worth a load test). CWE-636, CWE-400.

---

## N-3 (Medium) — `WatchServiceConfig` authorizes once and never re-checks; `WatchSecret` does

- **Where:** `internal/api/secrets.go:250-306`. The `Allow` check is at `:254`, before the
  loop. `sendCurrent` (`:259-290`) re-sends the **full config document** on every bus
  notification but does **not** re-run `checker.Allow`. Contrast `WatchSecret`
  (`:125-158`), which re-runs the whole `fetchAndDecrypt` → `Allow` pipeline on each update.
- **Problem:** A long-lived config watch keeps delivering config values after the caller's
  policy is revoked (`DeletePolicy`) or narrowed, because authorization is a one-time gate.
  `WatchServiceBundle` (`:388-427`) is similar but only emits change *flags* (no values), so
  its exposure is limited to change-timing metadata; `WatchServiceConfig` leaks actual data.
- **Impact:** Stale authorization on a persistent stream — revocation doesn't take effect
  until the client reconnects. Inconsistent with `WatchSecret`, which the design implies is
  the model.
- **Fix:** Re-run `checker.Allow` inside `sendCurrent` (and in the bundle watch loop before
  each notification), returning `PermissionDenied` and closing the stream when access is
  withdrawn. Add a test that revokes a policy mid-stream and asserts the next push is denied.
- **Acceptance:** revoking access on an open `WatchServiceConfig` stops further config
  delivery on the next update.
- **Confidence:** High (confirmed by reading). CWE-613 (insufficient session expiration),
  CWE-863.

---

## N-4 (Medium) — `PatchServiceConfig` has no size/op-count bounds; JSON-Patch `copy` amplifies

- **Where:** `internal/api/gitops.go:576-629` and `decodeJSONPatch` (`:694-714`). The patch
  is applied via `jsonpatch.Patch.Apply` (`:599`) with no cap on operation count, on the
  resulting document size, or on the stored config size generally.
- **Problem:** RFC 6902 `copy` duplicates a subtree; a short patch that chains `copy`
  operations (each doubling an array/object) produces a document that grows exponentially in
  the number of ops, all applied server-side in one transaction and then written to the DB.
  Even without `copy`, repeated `add` calls can grow a config unboundedly over time. This
  path is reachable by a **scoped workload** (its own namespace/service via the exact-match
  bypass — no policy row needed), so it's not admin-only.
- **Impact:** Memory/CPU amplification during apply and unbounded storage growth for a
  service's config document (which is then served in full by `GetServiceConfig`/bundles).
  Namespace-scoped, but a compromised or buggy workload can degrade itself and the shared
  config table.
- **Fix:**
  1. Cap the number of operations per `PatchServiceConfig` request.
  2. Cap the size of the resulting document (reject the transaction if the patched content
     exceeds a configured limit) — this also protects `SyncBundle`/git-sync config writes.
  3. Consider disallowing/limiting `copy` depth, or bounding total config bytes per
     `(namespace, service)`.
- **Acceptance:** a patch that would exceed the op-count or size cap is rejected with
  `InvalidArgument`/`ResourceExhausted` and nothing is written.
- **Confidence:** Medium-High (mechanism confirmed by reading; exact amplification factor
  worth a quick repro against `evanphx/json-patch/v5`). CWE-770, CWE-409.

---

## N-5 (Low-Medium) — Webhook rate limiter is a single global bucket

- **Where:** `internal/api/webhook.go:52-54,61-65` — one `rate.NewLimiter(20, 40)` shared
  across all repositories and all callers.
- **Problem:** The limiter correctly bounds crypto work (M-6), but because it's global, one
  repository's legitimate burst — or an attacker hammering a single guessed/known `repo_id`
  — consumes the shared budget and **starves webhook delivery for every other repo**. GitHub
  will retry, but sustained pressure on one path degrades convergence latency fleet-wide.
- **Impact:** Availability/fairness: a noisy or targeted repo delays sync for all repos.
- **Fix:** Add a per-repo (and/or per-source-IP) limiter layered under a global ceiling, so
  one repo can't exhaust the shared budget. The prior finding (M-6) already suggested
  per-IP + global; only global landed.
- **Acceptance:** saturating one `repo_id` does not prevent a different repo's valid webhook
  from being processed within the global ceiling.
- **Confidence:** High (confirmed by reading). CWE-770.

---

## N-6 (Low) — `extractTarGz` bounds total bytes but not file count

- **Where:** `internal/api/bundle.go:33-77`. `total` caps decompressed bytes at 256 MiB, but
  there is no cap on the **number** of entries. A bundle of millions of empty/tiny files
  passes the byte cap while creating a huge number of inodes/directory entries in the temp
  dir and a long synchronous walk in `SyncFromDir`.
- **Impact:** Local inode/FD/CPU pressure and slow syncs from an authenticated caller.
  Bounded by the byte cap in practice, so low severity.
- **Fix:** Add a maximum entry count (and optionally a max path depth) alongside the byte
  cap; abort past the limit.
- **Acceptance:** a bundle exceeding the entry cap is rejected before extraction completes.
- **Confidence:** High (confirmed by reading). CWE-770, CWE-410.

---

## N-7 (Low) — `signet secret set/rm` accept `..` segments — local path traversal

- **Where:** `cmd/signet/secret_cmd.go:92-99` (`splitSecretRef`) only checks for exactly 3
  non-empty `/`-separated segments; `resolveSecretPath` (`:169-171`) then
  `filepath.Join`s them under the secrets root. A ref like `../../x/y` splits into three
  non-empty parts (`..`, `..`, `x/...`) — wait, that's four; but `../a/b` → (`..`,`a`,`b`)
  is three valid parts, and `filepath.Join(repoRoot, "secrets", "..", "a", "b.yaml")`
  escapes the `secrets/` root. Chaining `..` in the namespace/service segments can climb
  above the repo root entirely.
- **Impact:** A CLI user can write/delete a `*.yaml` file outside the intended secrets tree.
  This is a **client-side, operator-only** tool operating on the operator's own checkout, so
  the blast radius is the operator's own filesystem — hence Low — but it's still an input
  that should be validated.
- **Fix:** Reject any segment equal to `.`/`..` (or containing a path separator) in
  `splitSecretRef`; assert the resolved absolute path stays under `repoRoot/secretsRoot`.
- **Acceptance:** `signet secret set '../x/y' ...` fails with a clear validation error.
- **Confidence:** Medium (mechanism clear from reading; worth a 2-line repro to confirm the
  exact `..` arrangement that escapes). CWE-22.

---

## N-8 (Low) — Write-path audit entries omit `PeerIP`; minor completeness gaps

- **Where:** `internal/gitops/sync.go:388-401` (`recordAudit` sets no `PeerIP`) and
  `internal/api/gitops.go:612-621`, `:667-677` (Patch/GetServiceConfig audit entries set no
  `PeerIP`). The read path does populate it (`internal/api/secrets.go:546-561`).
- **Impact:** Forensics on writes (who pushed a bundle / patched a config, from where) lack
  the peer address the read path records. Low, but audit trails are load-bearing here.
- **Fix:** Thread `peerIP(ctx)` into the GitOps write audit entries where a live peer exists
  (SPIFFE-scoped and admin-token calls both have one). Reconciler/webhook paths legitimately
  have no live peer.
- **Confidence:** High (confirmed by reading). CWE-778.

---

## N-9 (Info) — Legacy nil-AAD decrypt fallback is silent and open-ended

- **Where:** `internal/crypto/aes.go:136-149` (`DecryptWithFallback`), used across
  `secrets.go`, `webhook.go`, `sync.go`. Each fallback logs a per-artifact warning but there
  is no aggregate inventory of un-migrated artifacts and no enforcement deadline.
- **Impact:** The AAD-binding security upgrade (H-1) converges only as artifacts happen to be
  re-synced/re-registered. An operator can't easily answer "what is still on the weaker,
  swap-able encoding?" and there's no way to *require* completion. Also note the residual
  from H-1: AAD binds `(namespace, service, name)` but **not `version`**, so a same-secret
  prior-version rollback by a party with DB write access remains open (already tracked).
- **Fix:** Add an admin/CLI report of artifacts still on the legacy encoding (e.g. secrets
  with `kek_id IS NULL`, repos/SOPS keys that decrypted via fallback), and optionally a
  "refuse legacy" mode once migration is complete. Close the version-binding residual by
  including `version` in the secret AAD once a transactional version reservation exists.
- **Confidence:** High (confirmed by reading). CWE-353 (residual), operational hygiene.

---

## Usability vs. safety balance (U-series)

The owner asked specifically about places where safety may have over-rotated against
usability. These are judgment calls, not defects — each names the tension and a lighter-touch
option.

- **U-1 — Audit fail-closed couples audit availability to secret availability.**
  (Ties to N-2.) Default `SIGNET_AUDIT_FAIL_CLOSED=true` is the right *security* default, but
  it means a slow/unavailable audit table stops the whole store from serving — including
  workloads trying to boot. Consider a bounded degraded mode (durable local spool with a
  hard cap, then decide) so a transient audit outage doesn't become a fleet outage. At
  minimum, document this coupling prominently in the ops runbook.

- **U-2 — The exact-match convention hard-couples SA name to the `service` field.**
  `parseKubeSpiffeID` grants only when the SPIFFE `sa/<name>` equals the secret's `service`
  (`internal/auth/auth.go:141-144`). When a workload's ServiceAccount is named differently
  from its logical service (common: shared SAs, `default`, platform-generated names), there
  is **no** auto-grant and the operator must write an explicit policy — with little feedback
  explaining *why* the convention didn't fire. Consider (a) documenting this coupling front
  and center, and (b) a `signet policy explain <spiffe-id> <ns/svc/secret>` that says which
  rule matched or why the convention was skipped.

- **U-3 — Expiry is enforced on read but no write path sets `expires_at`.**
  (`design/draft.md:60-66` acknowledges this.) Operators reading the feature list may believe
  time-boxed secrets work end to end; today the column is inert because nothing populates it.
  Either ship a write path (CLI/admin surface) or label expiry "read-enforcement only; no
  ingestion path yet" wherever it's advertised, so it isn't mistaken for a working control.

- **U-4 — The `*` permission wildcard grows silently as new RPCs are added.**
  Warn-but-allow (`internal/api/admin.go:413-424`) is a defensible design, but a blanket
  grant written today silently authorizes operations that don't exist yet. Consider a
  periodic "here are your wildcard grants" report (not just a one-time creation warning) so
  blanket grants stay visible over time.

- **U-5 — The H-4 policy migration silently changed matching semantics (2→3 segment).**
  Existing `ns/secret` patterns were rewritten to `ns/*/secret`. That's the safe
  non-narrowing choice, but it's a **behavioral** change an operator could miss. Make sure the
  upgrade notes / release changelog call it out loudly and recommend a post-upgrade
  `signet policy list` review (the migration file comment isn't enough on its own).

- **U-6 — No operator read-back of secret values via the API.**
  By design, only workloads (mTLS) can read secret values; operators manage via git. This is
  a coherent trust boundary, but it makes incident debugging ("what value is actually stored
  for X?") awkward. Consider a deliberately-audited, admin-authenticated read-back that is
  itself loudly logged, rather than forcing operators toward out-of-band decryption.

- **U-7 — The admin/unseal path is multi-step and easy to get subtly wrong.**
  port-forward + short-lived token + RBAC/allowlist + audience is strong defense in depth,
  but the failure modes (wrong audience, missing SAR grant, non-loopback without TLS) surface
  as opaque `Unauthenticated`/`PermissionDenied`/transport errors. A `signet doctor` /
  preflight that checks token audience, SAR result, transport security, and seal state before
  the real call would remove a lot of friction without weakening anything.

---

## Suggested triage order

1. **N-1** (workload-reachable OOM) — highest exploitability from an ordinary SVID holder.
2. **N-2 / U-1** (audit coupling + throughput) — decide the availability posture deliberately.
3. **N-3** (stale authz on config watch) — small fix, real authorization gap.
4. **N-4** (patch amplification) — bound sizes before this path sees heavier use.
5. **N-5, N-6, N-7, N-8** — hardening.
6. **U-series** — product/ops decisions; fold into docs and the changelog.

Per `CLAUDE.md`, any code change here must be reflected in `design/draft.md`
(notably N-2's audit posture, N-3's watch-authz semantics, N-4's config caps, and the
U-series decisions if adopted).
