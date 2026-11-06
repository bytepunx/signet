---
id: D-0024
title: "S-0031: close-out's test tier fails on the BSR rate limit; how should it pass: I attempted to set the BSR_AUTH_TOKEN and BUF_TOKEN in the local environment"
kind: answer
sources: [TH-0021]
topics: [github, release, ci, client-csharp, csharp, code, all]
paths: [scripts/test-client-csharp.sh, design/issues/I-0007-the-bsr-rate-limit-resource-exhausted-too-many-requests-fails-the-buf-generate.md, design/issues/summary.md, design/issues/I-0008-flai-check-finds-roles-baseline-outside-the-story-at-close-out.md, design/issues/I-0009-flai-check-finds-roles-definition-outside-the-story-at-close-out.md, design/issues/I-0010-flai-check-finds-item-blocked-closed-outside-the-story-at-close-out.md]
date: 2026-11-06
status: active
absorbed_by: ""
asked_again: []
cited: []
---

# D-0024 S-0031: close-out's test tier fails on the BSR rate limit; how should it pass: I attempted to set the BSR_AUTH_TOKEN and BUF_TOKEN in the local environment

The operator, alex, answered TH-0021 at 2026-11-06T07:39:33Z.

What was asked:

```text
S-0031's fix is committed (T-0094, scripts/test-client-csharp.sh) and verified: 56/56 pass from an empty .cache/dotnet with id shimmed to uid 1001. Criteria 1 and 3 are ticked. Criterion 2 (green client-csharp job in ci.yml) can only be confirmed on a CI run after the branch reaches main.
```

The answer:

```text
I attempted to set the BSR_AUTH_TOKEN and BUF_TOKEN in the local environment
```

Source: TH-0021.
