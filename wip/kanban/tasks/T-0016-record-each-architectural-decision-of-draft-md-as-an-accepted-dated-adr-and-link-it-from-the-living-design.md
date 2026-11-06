---
id: T-0016
type: task
nature: improvement
title: Record each architectural decision of draft.md as an accepted, dated ADR and link it from the living design
status: done
parent: S-0008
owner: alex
created: 2026-10-24T23:00:52Z
updated: 2026-11-03T18:44:53Z
transitions:
  - to: ready
    at: 2026-11-03T18:40:18Z
    by: agent-S-0008
  - to: in-progress
    at: 2026-11-03T18:40:18Z
    by: agent-S-0008
  - to: done
    at: 2026-11-03T18:44:53Z
    by: agent-S-0008
stream: S-0008
tags: [design, documentation]
touches: [design/adrs/README.md, design/system/storage.md, design/system/encryption-and-unsealing.md, design/system/identity-and-policy.md, design/system/api.md, design/system/gitops-and-sops.md, design/system/audit.md, design/system/admin-access.md, design/system/deployment.md, design/adrs/0004-signet-stores-secrets-itself-rather-than-in-kubernetes-secrets.md, design/adrs/0005-cockroachdb-is-signet-s-secret-store.md, design/adrs/0006-networkpolicy-and-rbac-isolate-signet-s-storage-and-cluster-admin-is-the.md, design/adrs/0007-workloads-authenticate-to-signet-by-mtls-with-spiffe-svids-issued-by-spire.md, design/adrs/0008-authorization-is-convention-first-with-glob-policies-on-namespace-service-and.md, design/adrs/0009-secrets-use-envelope-encryption-under-a-master-key-kek-and-dek-hierarchy-held.md, design/adrs/0010-every-encryption-in-the-key-hierarchy-is-bound-by-aad-to-the-identity-of-what.md, design/adrs/0011-signet-starts-sealed-and-the-master-key-is-loaded-by-direct-key-or-shamir-m-of.md, design/adrs/0012-kubernetes-secret-unseal-with-optional-auto-unseal-keeps-the-master-key-in-a.md, design/adrs/0013-every-unseal-verifies-the-candidate-master-key-against-a-stored-key-check-value.md, design/adrs/0014-signet-s-data-lives-on-local-persistent-volumes-with-replication-left-to-raft.md, design/adrs/0015-spire-s-kubernetes-node-attestor-is-the-baseline-node-identity-with-the-tpm.md, design/adrs/0016-cert-manager-with-step-ca-and-an-offline-root-provides-the-pki-and-spire-is.md, design/adrs/0017-admin-access-is-a-tokenreview-followed-by-an-allowlist-or-subjectaccessreview.md, design/adrs/0018-the-admin-listener-is-reached-on-loopback-through-port-forward-the-cli-sends.md, design/adrs/0019-audit-records-are-hmac-chained-and-forwarded-out-of-cluster-fail-closed-on.md, design/adrs/0020-git-with-sops-is-the-source-of-truth-for-secrets-and-config-and-plain-config-is.md, design/adrs/0021-sops-files-are-encrypted-with-age-keys-that-signet-generates-and-holds-one-per.md, design/adrs/0022-signet-secret-set-and-rm-work-on-local-files-only-and-encrypt-by-calling-the.md, design/adrs/0023-sync-runs-from-a-rate-limited-webhook-and-a-periodic-reconciler-that-runs-only.md, design/adrs/0024-deletes-are-explicit-and-absence-from-a-git-walk-deletes-nothing.md, design/adrs/0025-signet-bundle-push-streams-committed-git-tree-files-to-the-server-which.md, design/adrs/0026-workloads-push-gitops-bundles-through-syncbundle-scoped-by-their-own-spiffe.md, design/adrs/0027-patchserviceconfig-applies-an-rfc-6902-json-patch-to-a-config-document-in-one.md, design/adrs/0028-getsopspublickey-accepts-a-caller-s-spiffe-identity-and-applies-no-policy-check.md, design/adrs/0029-putserviceconfig-creates-or-replaces-a-config-document-only-against-an-expected.md, design/adrs/0030-signet-only-imports-from-git-and-reconciles-api-config-writes-with-git-sync-by.md]
after: [T-0014]
usage:
  source: log
  seconds: 275
  estimated: true
  priced_by: harness
  models:
    - model: claude-opus-5-5
      input: 60
      output: 24654
      cache_read: 4465715
      cache_write: 92057
      cost: 2.0108
---
# T-0016 Record each architectural decision of draft.md as an accepted, dated ADR and link it from the living design

## Work

Waits for T-0014 so that the topic files exist to link from. From `server/design/draft.md`'s decision matrix and its sections, list the decisions that are architectural as `decisions.md` defines them (structure, technology, contracts between parts): at least CockroachDB as the store, mTLS via SPIFFE/SPIRE as the only client identity, the `namespace/service/secret` policy model, envelope encryption with AAD binding and the master/KEK/DEK tiers, the four unseal mechanisms and the key-check value, Kubernetes auto-unseal with etcd encryption as a prerequisite, the network isolation of the admin listener, SPIRE as the PKI, admin authorization by allowlist or SubjectAccessReview, fail-closed audit logging, SOPS with age (not PGP or KMS) for GitOps, and the git-free config paths (`PutServiceConfig`, `PatchServiceConfig`, SPIFFE-scoped `GetSOPSPublicKey`). Record each with `flai adr new "<the decision, as a sentence>" --status accepted --body-stdin --commit` from the story's worktree, one decision per ADR, the draft's deliberation and the alternatives it rejected in `## Context` and `## Alternatives considered`. Date each to when it was taken: take the date from `git log --follow --format=%ad -- server/design/draft.md` for the commit that introduced the section (the initial checkin of 2026-06-26 for most, 2026-07 and 2026-09 for the hardening and GitOps additions), set `date` in the ADR's front matter to it, and run `flai check` after. Then fill the `## Decisions` list of each topic file under `design/system/` with a relative link to each ADR it rests on, and bump `updated`.

## Done when

- Each decision in the list above has an accepted ADR under `design/adrs/`, dated to when it was taken, with a row in `design/adrs/README.md`.
- Each topic file under `design/system/` links the ADRs it rests on, and no ADR goes unlinked.
- `make lint-md` and `flai check --strict` pass on the worktree.

## Notes

Second layer. The ADR numbers cannot be named ahead: `flai adr new` numbers across every branch and worktree, and S-0001 writes ADRs too, which is why the story keeps the `design/adrs` folder touch. If `flai adr new` refuses `--commit` with a hand-set `date`, create the ADR, edit the date, and commit it with `flai task done`.
