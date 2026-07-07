# Signet Security Findings & Hardening Checklist

**Audit date:** 2026-07-01
**Scope:** `internal/` (crypto, auth, unseal, api, gitops, store, audit, server), `cmd/signetd`, `cmd/signet`, `deploy/helm`, and `design/draft.md`.
**Audience:** This is a remediation checklist. Each item is self-contained and actionable so a follow-up model (e.g. Sonnet 5) can pick it up, implement the fix, and check it off. Severities follow a rough CVSS-style intuition; treat **Critical/High** as release blockers for a production secret store.

> Convention: `path:line` references point at the code as of this audit. Line numbers may drift — grep for the quoted snippet if they do not match.

---

## Critical

### [x] C-1. Admin API authenticates but never authorizes (broken access control)
> **Fixed.** `internal/auth/token.go`'s `TokenValidator.Validate` now authorizes the TokenReview-authenticated identity via two complementary mechanisms (either sufficient): an allowlist (`SIGNET_ADMIN_SUBJECTS`, checked as a fast path) and a `SubjectAccessReview` against a synthetic `signet.io/adminoperations` resource, verb `administer` (delegates to cluster RBAC, no CRD required). Applies uniformly to both `AdminService` and `GitOpsService` since both share `TokenValidator`. Helm `clusterrole.yaml` grants `create` on `subjectaccessreviews`; `values.yaml` documents the new `adminSubjects` field and the RBAC grant shape. Tests: `internal/auth/token_test.go`.
- **Where:** `internal/api/admin.go:28-37` (`AdminServer.requireToken`), `internal/api/gitops.go:57-66` (`GitOpsServer.requireToken`), `internal/auth/token.go:29-55` (`TokenValidator.Validate`).
- **Problem:** `requireToken` only calls Kubernetes `TokenReview`, which answers *"is this a valid, authenticated token?"* — never *"is this principal allowed to administer signet?"* There is no `SubjectAccessReview`, no ServiceAccount allowlist, and no group/namespace check. Any identity that can (a) obtain a token whose audience includes `signet` and (b) reach the admin listener can call **every** admin RPC: `UnsealKey`, `UnsealShare`, `Seal`, `RotateSOPSKey`, `PruneSOPSKey`, `RegisterRepository`, `RemoveRepository`, `TriggerSync`, and `SyncBundle`.
- **Impact:** Privilege escalation to full control of the secret store. An attacker can register a repository pointing at infrastructure they control (then read every synced plaintext as it is decrypted), prune SOPS keys (destroying the ability to decrypt existing secrets), or seal the server (DoS). The only current barriers are the audience string and network reachability (see C-2 / H-4) — neither is an authorization decision.
- **Fix:**
  1. Add an explicit allowlist of authorized admin identities (ServiceAccount `namespace/name` and/or Kubernetes group) to config, e.g. `SIGNET_ADMIN_SUBJECTS`.
  2. After `TokenReview` succeeds, read `result.Status.User` (`Username`, `UID`, `Groups`) and require it to match the allowlist before proceeding. Reject with `codes.PermissionDenied` otherwise.
  3. Alternatively/additionally, issue a `SubjectAccessReview` for a synthetic resource (e.g. verb `unseal` on `signet.io/servers`) so authorization is delegated to cluster RBAC.
  4. Record the resolved admin identity in the audit log for every admin action (see H-2).
- **References:** CWE-862 (Missing Authorization), CWE-306 (Missing Authentication for Critical Function), OWASP API Security Top 10 API5:2023 (Broken Function Level Authorization). Kubernetes docs: [Authenticating — Token Review](https://kubernetes.io/docs/reference/access-authn-authz/authentication/) and [Authorization — SubjectAccessReview](https://kubernetes.io/docs/reference/access-authn-authz/authorization/).

### [x] C-2. Admin token audience can be silently disabled, degrading to "any token"
> **Fixed.** `cmd/signetd/config.go`'s `validate()` now rejects an empty/whitespace-only `SIGNET_KUBE_AUDIENCES` as a fatal startup error instead of falling back to "accept any audience". Tests: `TestValidate_EmptyKubeAudiencesRejected`, `TestValidate_WhitespaceOnlyKubeAudiencesRejected` in `cmd/signetd/config_test.go`.
- **Where:** `internal/auth/token.go:22-24` + `:36-39` (nil audiences ⇒ any audience accepted), `cmd/signetd/config.go:97-98,175-180` (audience parsing).
- **Problem:** `NewTokenValidator(client, nil)` accepts a token with **any** audience — including the default `https://kubernetes.default.svc` audience that is mounted into virtually every pod. The daemon default is `SIGNET_KUBE_AUDIENCES=signet` (good), but if an operator sets it to empty (or a future refactor passes `nil`), the admin endpoint will accept the ambient SA token of any pod that can reach it. Because C-1 means authentication alone is sufficient, this collapses to "any workload is an admin."
- **Impact:** With C-1 unfixed, this is a direct authentication bypass. Even with C-1 fixed, a too-broad audience widens the set of forgeable tokens.
- **Fix:**
  1. Treat an empty/nil audience list as a **fatal misconfiguration** for the admin endpoint rather than "accept any." Require at least one explicit, non-default audience.
  2. Keep the default `signet` and document that operators must mint admin tokens with `--audience=signet`.
  3. Add a startup log line stating exactly which audiences are enforced.
- **References:** CWE-1188 (Insecure Default Initialization), CWE-287 (Improper Authentication). Kubernetes [bound service account tokens / audiences](https://kubernetes.io/docs/concepts/security/service-accounts/#bound-service-account-tokens).

---

## High

### [x] H-1. Envelope encryption has no associated data (AAD) — secrets are not bound to their identity
> **Fixed.** `Encrypt`/`Decrypt`/`WrapKey`/`UnwrapKey` now take an `aad []byte` parameter (`internal/crypto/aes.go`); secret ciphertext and DEK wraps are bound via `icrypto.BindAAD(icrypto.AADSecret, namespace, service, name)`. Data written before this change (no AAD) is handled by `DecryptWithFallback`/`UnwrapKeyWithFallback`, which retry with nil AAD and log a warning identifying the artifact for re-sync. Regression test: `TestGetSecret_CrossRowBlobSwapFailsAuthentication` in `internal/api/secrets_test.go`. Residual scope note: AAD binds `(namespace, service, name)` but not `version`, so a same-secret prior-version rollback via direct DB write is not fully closed — closing that would require reserving the version number transactionally before encryption, which was scoped out of this pass; tracked as a fast-follow.
- **Where:** `internal/crypto/aes.go:38` (`gcm.Seal(nonce, nonce, plaintext, nil)`), `:69` (`gcm.Open(nil, nonce, body, nil)`), and all wrap/unwrap paths (`WrapKey`/`UnwrapKey`, `internal/api/secrets.go:435-444`, `internal/gitops/sync.go:285`).
- **Problem:** AES-GCM is always invoked with `nil` additional authenticated data. The ciphertext and wrapped DEK are therefore not cryptographically bound to the row that stores them (`namespace`, `service`, `secret_name`, `version`). GCM authenticates the *bytes*, not their *location*.
- **Impact:** An adversary with **write** access to CockroachDB (SQL injection reaching the DB, a compromised DB node, or a malicious DB admin — explicitly *not* excluded by the design's trust boundary) can:
  - **Substitute secrets:** copy `(encrypted_dek, ciphertext)` from a high-value row (e.g. `payments/db/password`) into a row a low-privilege workload is authorized to read. Decryption succeeds and the plaintext is served to the wrong caller.
  - **Roll back versions:** overwrite the current row's blob with an older `(encrypted_dek, ciphertext)` pair; nothing detects the downgrade.

  The design (`design/draft.md:126-134`, `:350-353`) sells envelope encryption as the *primary* at-rest protection, so this materially weakens the stated guarantee.
- **Fix:**
  1. Thread a context/identity string into `Encrypt`/`Decrypt` and `WrapKey`/`UnwrapKey` and pass it as GCM AAD, e.g. `namespace|service|secret_name|version` for the ciphertext and the same tuple for the DEK wrap.
  2. On decrypt, reconstruct the AAD from the row coordinates being requested so a swapped blob fails authentication.
  3. Add a regression test that swaps two rows' blobs and asserts decryption fails.
- **References:** CWE-353 (Missing Support for Integrity Check), CWE-565 (Reliance on Cookies/Data Without Integrity), NIST SP 800-38D §5.1.1 (purpose of AAD). Go [`cipher.AEAD.Seal`](https://pkg.go.dev/crypto/cipher#AEAD) `additionalData` parameter.

### [x] H-2. Config and bundle reads are not audited — audit trail is incomplete
> **Fixed.** `GetConfig`, `GetServiceConfig`, `WatchServiceConfig`, `GetServiceBundle`, and `WatchServiceBundle` (`internal/api/secrets.go`) now all call `record`/`auditOrDeny` for both permitted and denied outcomes, using sentinel `SecretName` values `<config>`/`<bundle>` for whole-document actions (`audit.Entry` already required non-empty `SecretName`, so no relaxation was needed there). `WatchServiceConfig` re-audits on every push (real data flows each time); `WatchServiceBundle` audits once at subscribe time only (no secret/config values ever flow through its notifications). Tests: 15 new cases in `internal/api/secrets_test.go` under "H-2: audit coverage for config/bundle paths".
- **Where:** `internal/api/secrets.go` — `GetSecret`/`WatchSecret` call `s.record(...)`, but `GetConfig` (:138), `GetServiceConfig` (:175), `WatchServiceConfig` (:207), `GetServiceBundle` (:274), and `WatchServiceBundle` (:342) **never** write an audit entry.
- **Problem:** `GetServiceBundle` decrypts and returns **every** secret for a service in one call, yet produces zero audit records. The design states (`design/draft.md:95`, `:440`) that every access — permitted or denied — is logged. `validateEntry` (`internal/audit/audit.go:135-148`) also *requires* a non-empty `SecretName`, so these paths would need a sentinel (e.g. `"<bundle>"`, `"<config>"`).
- **Impact:** The highest-exposure read path (full bundle) is invisible to the audit log and to the out-of-cluster HMAC-chained forwarder. Post-incident forensics and anomaly detection are defeated for config/bundle access.
- **Fix:**
  1. Add `s.record(...)` calls to all config/bundle read and watch paths, for both permitted and denied outcomes, mirroring `GetSecret`.
  2. Relax `validateEntry` to allow a documented sentinel for non-single-secret actions, or add distinct actions (`get_config`, `get_bundle`, `watch_config`, `watch_bundle`).
  3. Add tests asserting an audit row is written for each path.
- **References:** CWE-778 (Insufficient Logging), OWASP A09:2021 (Security Logging and Monitoring Failures).

### [x] H-3. Audit logging is fail-open — secrets are served even when the audit write fails
> **Fixed.** `SecretsServer` gained an `auditFailClosed bool` (constructor param, wired from `SIGNET_AUDIT_FAIL_CLOSED`, default `true`) and an `auditOrDeny` helper: when an access was otherwise permitted but its audit write fails, and fail-closed is enabled, the RPC returns `codes.Unavailable` instead of serving data. Applied to every read path (`GetSecret`, `WatchSecret`, and the H-2 additions). A denied access's own audit-write failure does not change its outcome (it was already being denied). Tests: fail-closed and fail-open cases across `GetSecret`, `GetConfig`, `GetServiceConfig`, `WatchServiceConfig`, `GetServiceBundle`, `WatchServiceBundle` in `internal/api/secrets_test.go`.
- **Where:** `internal/api/secrets.go:451-465` (`record` logs the error and returns), `:57-67` (`GetSecret` returns the plaintext regardless of `record`'s outcome).
- **Problem:** If `WriteAuditLog` fails (DB unavailable, disk full, chain key zeroed), the access is still completed and the secret returned. For a system whose security model leans on tamper-evident auditing, this is fail-open.
- **Impact:** An attacker who can degrade the audit path (e.g. exhaust DB connections) can read secrets without leaving a trace.
- **Fix:**
  1. Make audit durability a policy decision: add a configurable `SIGNET_AUDIT_FAIL_CLOSED` (default **true** for a secret store). When true and `Record` fails, return `codes.Unavailable` and do **not** return the secret.
  2. Ensure the audit write happens (and is confirmed durable) *before* the plaintext is sent for the permitted case.
- **References:** CWE-778, CWE-636 (Not Failing Securely / "Failing Open").

### [x] H-4. Access policies ignore the `service` dimension — cross-service secret confusion
> **Fixed.** `evalPolicies`/`Checker.Allow` (`internal/auth/auth.go`) now match against a three-segment `namespace/service/secret_name` target, matching the format the design doc already documented. Migration `000007_policy_service_dim.sql` rewrites existing two-segment patterns by inserting a wildcard service segment (`ns/secret` → `ns/*/secret`), the safest non-narrowing equivalent to their prior (non-)behavior — documented loudly in the migration file and here: **this is a breaking change to policy semantics; review `access_policies` after upgrading.** Regression test: `TestAllow_PolicyGrant_DistinguishesServiceWithSameSecretName` in `internal/auth/checker_test.go`; migration behavior verified against a real DB in `internal/store/policy_migration_integration_test.go`.
- **Where:** `internal/auth/auth.go:154-178` (`evalPolicies`): `target := namespace + "/" + secretName` and `path.Match(p.Pattern, target)`. The `service` argument passed into `Allow` (`:111`) is used **only** for the exact-match convention (`:119-122`), never for explicit policy matching.
- **Problem:** The design's policy examples are three-segment `namespace/service/secret` (`design/draft.md:109-116`), but the implementation matches only two segments `namespace/secretName`. Consequences:
  - Two services in the same namespace with a secret of the same name are indistinguishable to a policy — a grant for one silently grants the other.
  - Policies cannot express "service X may read service Y's secret Z" precisely; the service is dropped.
- **Impact:** Over-broad grants and potential cross-service secret disclosure that the operator believes is scoped.
- **Fix:**
  1. Decide the canonical policy target shape (recommend `namespace/service/secretName`) and update both `evalPolicies` and the `secret_pattern` semantics/documentation to match the design.
  2. Include `service` in the matched target and in `store.Policy` evaluation.
  3. Migrate existing policy rows / document the format change. Add tests for same-name-different-service isolation.
- **References:** CWE-863 (Incorrect Authorization), CWE-284 (Improper Access Control).

### [x] H-5. Secret expiry (`expires_at`) is never enforced — expired secrets are served
> **Fixed.** `GetSecret` and `FetchServiceSecrets` (`internal/store/secrets.go`) now filter on `expires_at IS NULL OR expires_at > now()`; an entirely-expired secret returns `ErrNotFound`, and a secret whose newest version has expired falls back to its latest non-expired version if one exists. Residual note (documented in `design/draft.md`): no ingestion path currently sets `expires_at` at all, so this enforcement has no effect until a write path for it exists — tracked as a follow-on feature, not a security gap in itself. Tests: `internal/store/secrets_expiry_integration_test.go` (real DB).
- **Where:** `internal/store/secrets.go:83-98` (`GetSecret`) and `:170-216` (`FetchServiceSecrets`) select the latest version with no `expires_at` predicate; schema has the column (`migrations/000001_init.sql`).
- **Problem:** The design lists expiry as a first-class feature (`design/draft.md:33`, `:49`). Nothing filters out or refuses expired rows at read time, and there is no reaper.
- **Impact:** Secrets intended to become invalid remain readable indefinitely, defeating time-boxed credentials.
- **Fix:**
  1. In `GetSecret`/`FetchServiceSecrets`, add `AND (expires_at IS NULL OR expires_at > now())`, returning `ErrNotFound` (or a distinct `ErrExpired` mapped to `codes.FailedPrecondition`) for expired secrets.
  2. Optionally add a background purge and/or a CockroachDB row-TTL on `secrets`.
  3. Add tests covering expired-secret reads.
- **References:** CWE-613 (Insufficient Session/Credential Expiration).

### [x] H-6. CLI sends bearer tokens over an unauthenticated, unencrypted channel
> **Fixed.** `cmd/signet/root.go`'s `adminTransportCreds` now uses plaintext only for loopback `--server` addresses (the documented `kubectl port-forward` workflow); every other address is automatically upgraded to TLS (system trust store, or `--ca <path>`), and `tokenCreds.RequireTransportSecurity()` reflects the actual transport so gRPC enforces the invariant. `--tls` forces TLS even for loopback. Tests: `cmd/signet/root_test.go`.
- **Where:** `cmd/signet/root.go:55-59` (`insecure.NewCredentials()`), `:92` (`RequireTransportSecurity() ⇒ false`).
- **Problem:** The admin gRPC channel is plaintext and the client injects `Authorization: Bearer <SA token>` over it with transport security explicitly disabled. The design's safety argument is "admin is localhost-only, reached via `kubectl port-forward`" (which is TLS to the API server). But nothing stops an operator from pointing `--server` at a non-loopback address, sending a live SA token in cleartext across the network.
- **Impact:** Token interception → admin access (compounded by C-1). Also no server authentication: the CLI cannot tell it is talking to the real signetd.
- **Fix:**
  1. Refuse `RequireTransportSecurity()==false` unless the target host is loopback; require TLS otherwise.
  2. Support a `--ca`/`--tls` mode so the CLI can verify the admin server certificate for non-loopback use.
  3. Emit a clear warning whenever a token is sent over an insecure channel.
- **References:** CWE-319 (Cleartext Transmission of Sensitive Information), CWE-295 (Improper Certificate Validation).

---

## Medium

### [x] M-1. Design/implementation gap: KEK tier and key rotation do not exist
> **Fixed.** Implemented the full three-tier hierarchy: `key_encryption_keys` table + `secrets.kek_id` (migration `000006_kek.sql`), `internal/store/kek.go` for CRUD, `internal/gitops/kek.go` for lazy KEK bootstrap on first secret write, and new `AdminService` RPCs `RotateKEK`/`ListKEKs`/`PruneKEK`/`RotateMasterKey` (`internal/api/admin.go`) with matching CLI (`signet kek rotate|list|prune`, `signet master-key rotate`). `RotateMasterKey` re-wraps KEKs+key-check value in one DB transaction before adopting the new key, with best-effort DB rollback if the in-memory adoption fails. `design/draft.md` §5 updated with the AAD/rotation details.
- **Where:** Design describes Master → KEK → DEK with independent rotation (`design/draft.md:122-144`). Implementation wraps DEKs **directly** under the master key: `internal/gitops/sync.go:285` (`WrapKey(masterKey, dek)`), `internal/api/secrets.go:436` (`UnwrapKey(masterKey, ...)`). There is **no KEK** and **no rotation code** for master key, KEK, or DEKs anywhere.
- **Impact:** Master-key rotation would require re-wrapping every DEK, but no such routine exists — so rotation is effectively impossible without downtime/bespoke scripting. The "compromise isolation" and "independent rotation" benefits claimed in the design are not delivered. This is both a security capability gap (no key rotation) and a documentation accuracy problem.
- **Fix:** Either (a) implement the KEK tier and rotation flows the design promises (master rotate → re-wrap KEKs; KEK rotate → re-wrap DEKs), or (b) update `design/draft.md` to describe the actual two-tier model and add at minimum a master-key rotation routine that re-wraps all DEKs. Per `CLAUDE.md`, the design doc must reflect reality either way.
- **References:** NIST SP 800-57 Part 1 (key management / cryptoperiods & rotation), CWE-320 (Key Management Errors).

### [x] M-2. Kubernetes auto-unseal stores the master key in plaintext in etcd
> **Fixed.** Added an explicit "etcd encryption-at-rest is a prerequisite, not optional" note to `design/draft.md`'s Mechanism 3 section, and a `slog.Warn` at signetd startup (`cmd/signetd/main.go`) whenever `SIGNET_KUBE_UNSEAL_SECRET` is set, since signetd cannot itself verify the cluster's `EncryptionConfiguration`. Did not add a runtime check refusing to enable auto-unseal without confirmed etcd encryption (no portable way to verify this from inside a pod) or a backup-exclusion mechanism (operator-side concern, not signet's).
- **Where:** `cmd/signetd/kube_unseal.go:40-58`, `cmd/signet/init_cmd.go` (creates the `master.key` Secret), design `Mechanism 3` (`design/draft.md:195-301`).
- **Problem:** The master key is written to a Kubernetes `Secret` (`master.key`), which is base64 — not encrypted — and stored in etcd (unencrypted at rest unless the cluster explicitly configures an `EncryptionConfiguration`). Anyone with `get secret` in the namespace obtains the master key, which unwraps every DEK. This contradicts the headline "master key never touches disk" (`design/draft.md:137,155`).
- **Impact:** Cluster-admin (and any RBAC-granted namespace reader) = full compromise. Documented as a weaker trust boundary, but it is the single largest key-exposure surface and is easy to enable.
- **Fix:**
  1. Keep it opt-in (already default-off) and prominently document the etcd-encryption prerequisite.
  2. Verify/require etcd encryption-at-rest before enabling, or refuse to write the Secret if the cluster cannot confirm it.
  3. Recommend TPM/Shamir for anything beyond dev in docs and `NOTES.txt`.
  4. Ensure the `master.key` Secret is excluded from any backup/GitOps export tooling.
- **References:** CWE-312 (Cleartext Storage of Sensitive Information), CWE-522 (Insufficiently Protected Credentials). [Encrypting Secret Data at Rest](https://kubernetes.io/docs/tasks/administer-cluster/encrypt-data/).

### [x] M-3. Reconstructed Shamir key is not verified (no key-check value)
> **Fixed.** `internal/api/kcv.go` adds `VerifyOrInitKeyCheckValue`, called synchronously after every successful unseal (direct key, Shamir, and Kubernetes auto-unseal) before the server is reported operational. On mismatch it re-seals and returns/logs a clear error rather than leaving a wrong key silently loaded. See `design/draft.md` §6 "Key-Check Value".
- **Where:** `internal/unseal/shamir.go:67-86`, `internal/unseal/shamir_math.go:62-100` (`combineSecret`).
- **Problem:** A well-formed but *wrong* share yields a wrong 32-byte key that passes `store.Set`, transitions to `Unsealed`, and then fails every DEK unwrap at serve time with an opaque `ErrAuthenticationFailed`. There is no way to distinguish "a keyholder submitted a bad share" from "the data is corrupt."
- **Impact:** Availability + operability: a single bad/malicious share silently poisons the unseal, and the wrong key is briefly loaded into memory as if valid.
- **Fix:** At initialization, store a key-check value — e.g. encrypt a known constant under the master key (with AAD, per H-1) and persist the ciphertext. After any unseal (key, Shamir, or auto), verify the KCV before declaring `Unsealed`; on mismatch, seal and return a clear error.
- **References:** CWE-354 (Improper Validation of Integrity Check Value). HashiCorp Vault stores an unseal-key SHA-256 fingerprint for exactly this reason.

### [x] M-4. Reconciler creates a new secret version on every cycle (unbounded growth)
> **Fixed.** `Syncer.storeSecret` (`internal/gitops/sync.go`) now calls `isUnchanged` before encrypting/writing: if the freshly-decrypted plaintext matches what's already stored AND the stored row is on the current active KEK, the write is skipped. Deliberately does **not** persist any hash of the plaintext (which would be a new oracle for an attacker with DB read access) — instead it decrypts the currently-stored version in memory for a direct comparison, discarding it immediately. A row on a legacy/rotated-away KEK is never "unchanged" so it is naturally rewritten onto the current epoch, converging the H-1/M-1 migration. Tests: `internal/gitops/dedup_test.go`.
- **Where:** `internal/gitops/sync.go:263-298` (`storeSecret` → `PutSecret`), `internal/store/secrets.go:43-79` (`PutSecret` always auto-increments `version`), `internal/gitops/reconcile.go:30-44` (default 5-minute `FullSync` of all repos), `SyncFromDir` re-stores every file each run.
- **Problem:** `FullSync`/`SyncFromDir` unconditionally re-writes every secret, and `PutSecret` inserts a brand-new version each time — even when the plaintext is unchanged. At the default interval that is ~288 new versions/secret/day, each with its own DEK and ciphertext row.
- **Impact:** Storage bloat, index growth, audit/version-history noise, and slow degradation of query performance — a gradual availability risk.
- **Fix:**
  1. Detect no-op syncs: compare against the current stored version (e.g. hash the plaintext, or compare source file content/SHA) and skip `PutSecret` when unchanged.
  2. Optionally cap retained versions per secret (prune old versions), independent of the change-detection fix.
- **References:** CWE-770 (Allocation of Resources Without Limits or Throttling).

### [x] M-5. Admin gRPC server has no panic-recovery interceptor for streaming RPCs
> **Fixed.** Added `grpc.ChainStreamInterceptor(recoveryStreamInterceptor)` to `adminSrv` in `internal/server/server.go` (the interceptor already existed and was already applied to the workload server). Test: `TestRun_PanicInStreamHandlerReturnsInternal` in `internal/server/server_test.go`, which panics inside `SyncBundle` and confirms the process survives across two calls.
- **Where:** `internal/server/server.go:131-139` — `adminSrv` is built with `grpc.ChainUnaryInterceptor(recoveryInterceptor)` but **no** `grpc.ChainStreamInterceptor`. The `GitOpsService.SyncBundle` RPC (`internal/api/gitops.go:305`) is a client-streaming handler registered on `adminSrv`.
- **Problem:** A panic inside `SyncBundle` (e.g. triggered by a malformed bundle reaching a nil-deref path) is not recovered and will crash the whole process, taking down the workload listener too.
- **Impact:** Denial of service via the admin/bundle path.
- **Fix:** Add `grpc.ChainStreamInterceptor(recoveryStreamInterceptor)` to `adminSrv` (the interceptor already exists at `server.go:300-308`). Add a test that panics in a stream handler and asserts the server survives.
- **References:** CWE-248 (Uncaught Exception), CWE-755 (Improper Handling of Exceptional Conditions).

### [x] M-6. Webhook decrypts the stored secret before cheaply rejecting bad requests (unauth work amplification)
> **Fixed.** `WebhookHandler` (`internal/api/webhook.go`) now rate-limits (20 req/s, burst 40, global) before doing any other work, and returns the identical `401 unauthorized` response for both an unknown `repo_id` and a bad HMAC signature (the distinguishing detail is only logged). Did not implement webhook-secret caching (the other suggested mitigation) — the rate limit already bounds the amplification, and caching would need its own invalidation-on-rotation logic; left as a possible future optimization. Tests: `internal/api/webhook_test.go` (new file — this handler had no dedicated tests before).
- **Where:** `internal/api/webhook.go:71-97` — for every request to a valid `repo_id`, the handler performs a master-key `Use` + `Decrypt` of the webhook secret *before* verifying the HMAC signature; there is no rate limiting.
- **Problem:** An unauthenticated caller who knows/guesses a `repo_id` can force a master-key decryption operation per request. Also, distinct status codes (`404` for unknown repo vs `401` for bad signature, `:74`/`:95`) allow enumeration of valid repo IDs (mitigated somewhat by high-entropy UUIDs).
- **Impact:** CPU/crypto amplification DoS and minor information disclosure.
- **Fix:**
  1. Add rate limiting / connection limits on the webhook listener (per-IP and global).
  2. Consider caching the decrypted webhook secret in memory (zeroed on seal) to avoid repeated master-key ops, or verify cheaper preconditions first.
  3. Return a uniform response for unknown-repo and bad-signature to reduce enumeration signal.
- **References:** CWE-770 (Resource Allocation Without Limits/Throttling), CWE-204 (Observable Response Discrepancy).

### [x] M-7. TPM auto-unseal is advertised but unimplemented
> **Documented.** Added a prominent "Status: design only" callout to the top of `design/draft.md`'s TPM/vTPM section stating that `UnsealWithTPM` returns "not yet implemented" and nothing in that section is available to operators yet. Did not implement TPM 2.0 support itself (a substantial feature, `github.com/google/go-tpm` integration, out of scope for a documentation-accuracy fix).
- **Where:** `internal/unseal/tpm.go:14-30` (`return fmt.Errorf("TPM unseal: not yet implemented")`, behind `//go:build tpm`), `tpm_stub.go` for the default build; design `Mechanism 4` (`design/draft.md:303-321`) presents it as a supported, hardware-rooted option with a hard-fail-safe.
- **Problem:** Operators reading the design may believe hardware-rooted auto-unseal is available; it is a stub.
- **Impact:** Misplaced trust in a non-existent control; potential deployment planning errors.
- **Fix:** Either implement TPM 2.0 seal/unseal (e.g. `github.com/google/go-tpm`, sealing to PCR state as described) or clearly mark it "planned / not implemented" in `design/draft.md` and any user docs.
- **References:** CWE-1059 (Insufficient/Incorrect Documentation) — accuracy of security-relevant docs.

---

## Low / Hardening / Defense-in-depth

### [x] L-1. Age private-key "zeroing" is a no-op (immutable string)
> **Fixed** as part of H-1. `GenerateAgeKey` (`internal/gitops/keys.go`) now handles the private key as `[]byte` end to end (`privKeyBytes := []byte(id.String())`, `defer ZeroBytes(privKeyBytes)`) instead of a `string`, so it is actually zeroed. The misleading loop and comment were removed.
- **Where:** `internal/gitops/keys.go:21-38` (`GenerateAgeKey`).
- **Problem:** `privKeyStr := id.String()` produces an immutable Go `string`; the loop that "zeros" it (`strings.Repeat("\x00", ...)` + `break`) merely reassigns the variable and leaves the original backing bytes in memory until GC. Multiple `[]byte(privKeyStr)` conversions create further uncleared copies. The comment/design claim that plaintext "never persists beyond this function's stack frame" is false.
- **Fix:** Avoid `string` for key material — obtain/handle the age private key as `[]byte` end to end and zero it (as `DecryptAgeKey` does for its plaintext at `:63-65`). Remove the misleading loop and comment.
- **References:** CWE-226 (Sensitive Information in Resource Not Removed Before Reuse), CWE-316 (Cleartext Storage in Memory).

### [x] L-2. Master-key `KeyStore.Use` exposes raw locked-buffer bytes to callbacks by contract
> **Fixed (documentation).** Strengthened the `Use` doc comment in `internal/crypto/key.go` into an explicit "HARD INVARIANT" block spelling out what callbacks must never do (store the slice anywhere that outlives the call, return it to their own caller) and why (defeats guard pages/canaries/zeroing, leaks master key material to the unlocked heap). No code change — this is a discipline/review aid, not a runtime enforcement mechanism, as scoped by the finding.
- **Where:** `internal/crypto/key.go:66-78` (`Use` passes `s.buf.Bytes()` directly), call sites in `secrets.go`, `gitops.go`, `webhook.go`, `sync.go`, `keys.go`.
- **Problem:** Safety depends entirely on every callback not retaining the `masterKey` slice past return. Current callers are disciplined, but there is no defensive copy or post-call invalidation, so a future careless callback could leak the master key onto the unlocked heap.
- **Fix:** Document the invariant as a hard rule (already partially in the doc comment) and add a review/lint checklist item; consider a debug build that scribbles a canary after `fn` returns to catch retention in tests.
- **References:** CWE-244 (Improper Clearing of Heap Memory Before Release).

### [x] L-3. GF(2^8) arithmetic in Shamir combine is not constant-time
> **Improved.** `gfMul` (`internal/unseal/shamir_math.go`) no longer branches on secret-dependent bits — both conditionals (the low bit of `b`, the high bit of `a`) are replaced with bitmasks (`-(bit)` → `0x00`/`0xFF`) applied via AND/XOR. `gfPow`'s loop count is already a fixed public constant (254), so it had no branching concern. This does not guarantee true hardware-level constant time (cache/pipeline effects are outside Go's control) but removes the most direct source of secret-dependent branching, as the finding's "if raising the bar is desired" scope anticipated. All existing GF/Shamir tests pass unchanged, confirming correctness was preserved.
- **Where:** `internal/unseal/shamir_math.go:105-155` (`lagrangeAt0`, `gfMul`, `gfDiv` via `gfPow(b,254)`).
- **Problem:** `gfMul` branches on data bits and `gfDiv` performs a data-dependent 254-iteration exponentiation over secret-derived values. Not constant-time.
- **Impact:** Low — reconstruction happens once per unseal, locally, not attacker-triggered at scale. Still a deviation from constant-time hygiene for master-key material.
- **Fix:** Use a constant-time GF(2^8) implementation (precomputed log/exp tables accessed obliviously, or a vetted SSS library) if raising the bar here is desired.
- **References:** CWE-208 (Observable Timing Discrepancy). See also the design note that a well-reviewed SSS library may be preferable to hand-rolled field math.

### [x] L-4. Audit chain key is delivered via environment variable
> **Fixed.** Added `SIGNET_AUDIT_CHAIN_KEY_FILE` / `-audit-chain-key-file` (`cmd/signetd/config.go`) as an alternative to the inline `SIGNET_AUDIT_CHAIN_KEY` — reads and trims the hex key from a file (e.g. a projected Secret volume), avoiding the key sitting in process environment. Exactly one of the two sources must be set; both or neither is a startup error. The original env-var path remains supported for backward compatibility. Tests: `TestValidate_AuditChainKeyFromFile`, `TestValidate_AuditChainKeyFileMissing`, `TestValidate_AuditChainKeyBothInlineAndFileRejected` in `cmd/signetd/config_test.go`. Not wired into the Helm chart's volume mounts in this pass — the chart still uses the env-var Secret pattern; adding a projected-volume option is a natural follow-up.
- **Where:** `cmd/signetd/config.go:103-104` (`SIGNET_AUDIT_CHAIN_KEY`), injected via `envFrom`/`secretKeyRef` in `deploy/helm/signet/templates/deployment.yaml`.
- **Problem:** The HMAC chain key lives in the process environment (readable via `/proc/<pid>/environ`, core dumps, some log/telemetry paths). It only protects audit tamper-evidence (not confidentiality), and it is zeroed on shutdown (`audit.go:86-97`), so impact is limited — but env-var key delivery is a known weak channel. Note the tamper-evidence guarantee is also only as strong as keeping this key outside the DB's trust boundary: anyone with both DB write access and this key can recompute the chain.
- **Fix:** Prefer a file-based or projected-secret delivery read into memory and cleared, or derive/rotate the chain key. Ensure it is never co-located with the audit DB backups. Document the trust assumption (chain key must be held outside the audited store).
- **References:** CWE-526 (Cleartext Storage of Sensitive Information in an Environment Variable).

### [x] L-5. Audit chain key is not zeroed on `Seal` (only on process exit)
> **Resolved as documentation, not a code change.** Confirmed this is the *correct* behavior, not a bug: keeping the chain key loaded for the whole process lifetime (not tied to Seal/unseal) is what lets denied access attempts made against a sealed server still be audited — zeroing it on every seal would silently break that. Fixed the misleading comment in `internal/audit/audit.go` (`Writer.chainKey` field doc and `Zero`'s doc comment) to state the actual, deliberate lifecycle instead of the aspirational-but-wrong "zeroed when the server seals."
- **Where:** `internal/unseal/manager.go:137-145` (`Seal` zeroes master key + shares but not the audit writer), `internal/audit/audit.go:84-97` (`Zero` only called via `defer` in `cmd/signetd/main.go:70`).
- **Problem:** The design note says the chain key "should be zeroed when the server seals" (`audit.go:23`), but `Seal()` does not touch it; it persists in memory across a sealed period.
- **Fix:** If the intended lifecycle is "chain key present only while unsealed," wire the audit writer into the seal/unseal transition. Otherwise update the comment to reflect that it lives for the process lifetime. (Note: keeping audit working *while sealed* is arguably desirable so denied accesses are still logged — decide deliberately and document.)
- **References:** CWE-226.

### [x] L-6. `extractTarGz` `..` rejection is coarse (over-rejects, relies on prefix check)
> **Fixed.** Removed the `strings.Contains(clean, "..")` substring check from `internal/api/bundle.go`; the canonical `filepath.Clean` + prefix comparison (already present) is now the sole traversal guard. Tests: `TestExtractTarGz_AllowsDotDotSubstringInFilename` (new — confirms a legitimate filename like `foo..bar.yaml` now extracts) and `TestExtractTarGz_RejectsTraversalViaJoinedSubpath` (new — confirms a traversal buried in a longer path is still caught) in `internal/api/bundle_test.go`.
- **Where:** `internal/api/bundle.go:47-58`.
- **Problem:** `strings.Contains(clean, "..")` rejects legitimate names containing `..` (e.g. `foo..bar.yaml`). The actual traversal protection is the subsequent prefix check (correct), and symlinks/hardlinks/devices are skipped (`:43-45`, good — this avoids the classic tar symlink escape). No security hole, but the `..` substring test is imprecise.
- **Fix:** Drop the substring check and rely on the canonical `filepath.Rel`/prefix comparison (already present) to detect escapes; this both tightens and de-noises the logic. Keep the size cap and the non-regular-file skip.
- **References:** CWE-22 (Path Traversal / "Zip Slip"), CWE-59 (Link Following). The prefix + non-regular-file skip already mitigate the primary attack; this is a refinement.

### [x] L-7. `--force` on `signet init` regenerates the master key, orphaning all existing secrets
> **Fixed.** `signet init --force` against an existing Secret now requires typing `yes` at an interactive prompt (via the new shared `confirmDestructive` helper in `cmd/signet/confirm.go`), or `--yes` for scripted use; skipped for `--dry-run` and for first-time Secret creation (not destructive). Tests: `TestInitForce_RequiresConfirmation`, `TestInitForce_ConfirmedViaPrompt`, `TestInitForce_DryRunSkipsConfirmation` in `cmd/signet/init_cmd_test.go`.
- **Where:** `cmd/signet/init_cmd.go` (`--force` path), design `signet init` step 3 (`design/draft.md:228-241`).
- **Problem:** Regenerating the master key makes every previously wrapped DEK undecryptable — effectively destroying all stored secrets — with no confirmation of the blast radius.
- **Impact:** Catastrophic, irreversible data loss if run by mistake.
- **Fix:** Require an interactive confirmation (or `--yes`/`--i-understand-this-destroys-secrets`) and a loud warning when `--force` would overwrite an existing key. Consider refusing if secrets already exist unless an explicit override is passed.
- **References:** CWE-1188 / operational-safety hardening (destructive default).

### [x] L-8. Plain-YAML service configs are stored unencrypted
> **Documented.** Added a "Plain Config vs. Secrets" subsection to `design/draft.md` (Section 12) making explicit that `config_path` files are plaintext JSON at rest, protected only by CockroachDB's storage-layer encryption, and that `secrets_path` is the only ingestion route with per-secret envelope encryption. No code change — this is a scan/lint suggestion the finding marked optional; not implemented in this pass.
- **Where:** `internal/gitops/sync.go:433-448` (`storeConfig` → `PutServiceConfig`), served by `GetConfig`/`GetServiceConfig`/`GetServiceBundle`.
- **Problem:** Unlike secrets (SOPS + envelope), config files under `ConfigPath` are stored as plaintext JSON, protected only by CockroachDB's at-rest encryption. This is by design (config ≠ secret), but operators may inadvertently place sensitive values in config.
- **Fix:** Document clearly that config values are **not** enveloped and must not contain secrets; optionally scan/lint config for secret-looking values at sync time and warn.
- **References:** CWE-312 (Cleartext Storage) — informational / documentation.

### [x] L-9. Exact-match convention does not validate the SPIFFE trust domain
> **Fixed.** `parseKubeSpiffeID` (`internal/auth/auth.go`) now takes the configured trust domain and rejects any SPIFFE ID whose `spiffe://<trust-domain>/...` host does not match, independent of the TLS-layer check. `Checker` gained a `trustDomain` field, threaded from `cfg.TrustDomain` in `cmd/signetd/main.go`. Tests: `TestParseKubeSpiffeID_TrustDomainMismatch`/`TestParseKubeSpiffeID_TrustDomainMatch`.
- **Where:** `internal/auth/auth.go:119-122,138-150` (`parseKubeSpiffeID` ignores the trust domain segment).
- **Problem:** The auto-grant compares only `ns`/`sa` path segments, not the trust domain. Currently mitigated because `server.go:328-333` (`SpireCredentials` + `tlsconfig.AuthorizeMemberOf(td)`) only admits SVIDs from the configured trust domain. But if federated trust domains are ever added, an identity from a foreign domain with matching `ns/sa` would be auto-granted.
- **Fix:** Validate the trust domain in `parseKubeSpiffeID`/`Allow` against the server's configured trust domain (belt-and-suspenders with the TLS-layer check) so the auth decision is self-contained and federation-safe.
- **References:** CWE-284 (Improper Access Control), SPIFFE ID [trust domain semantics](https://spiffe.io/docs/latest/spiffe-about/spiffe-concepts/).

---

## Verified-good (no action needed — recorded so they are not re-flagged)

- **SQL is fully parameterized** across `internal/store/*` (pgx placeholders); no string-built queries observed → no SQL injection surface found.
- **AES-256-GCM construction is correct**: 12-byte CSPRNG nonce prefix, `gcm.Seal(nonce, nonce, ...)`, length checks before slicing (`internal/crypto/aes.go`). Randomness uses `crypto/rand` everywhere (no `math/rand`).
- **Webhook HMAC uses `hmac.Equal`** (constant-time) and validates the `sha256=` prefix (`internal/gitops/github.go:21-40`).
- **Audit HMAC chaining is length-prefixed** to prevent field-boundary collisions (`internal/audit/audit.go:112-133`).
- **Shamir `combineSecret` rejects duplicate and zero x-coordinates** (`shamir_math.go:80-88`), preventing the div-by-zero DoS and same-share replay to threshold.
- **Master key held in `memguard` locked buffer**, DEKs/plaintext zeroed after use (`internal/crypto/key.go`, `internal/api/secrets.go`).
- **Master key is zeroed on graceful shutdown** via `server.drain()` → `mgr.Seal()` → `KeyStore.Zero()` (`internal/server/server.go:284`). (Note: this covers SIGTERM/SIGINT; a `SIGKILL` obviously cannot run cleanup — inherent.)
- **gRPC error mapping** avoids leaking internal details to callers (`internal/api/errors.go`).
- **Helm hardening defaults are sound**: `runAsNonRoot: true`, `runAsUser: 1000`, `readOnlyRootFilesystem: true`, `allowPrivilegeEscalation: false`, `capabilities.drop: [ALL]`, distroless `nonroot` base image, NetworkPolicy enabled by default, admin port bound to `127.0.0.1`, auto-unseal RBAC scoped to a namespaced `Role` with `resourceNames` pinned to the single Secret.

---

## Suggested remediation order

1. **C-1, C-2** — admin authorization + audience enforcement (highest exploitability).
2. **H-1** — AAD binding on envelope encryption (protects the core at-rest guarantee).
3. **H-2, H-3** — complete + fail-closed audit (needed to trust everything else).
4. **H-4, H-5** — correct authorization scope and expiry enforcement.
5. **H-6, M-5** — transport security for CLI + admin stream panic recovery.
6. **M-1** — reconcile the KEK/rotation design gap (implement or document).
7. Remaining **M** and **L** items as hardening.

Per `CLAUDE.md`, any change here must be reflected in `design/draft.md` (e.g. M-1, M-2, M-7 explicitly, and the policy-format change in H-4).
</content>
</invoke>
