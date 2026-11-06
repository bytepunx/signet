---
id: D-0024
title: "S-0031: close-out's test tier fails on the BSR rate limit; how should it pass: I attempted to set the BSR_AUTH_TOKEN and BUF_TOKEN in the local environment"
kind: answer
sources: [TH-0021]
topics: [ci, release, github, client-csharp]
paths: [scripts/test-client-csharp.sh, design/issues/I-0007-the-bsr-rate-limit-resource-exhausted-too-many-requests-fails-the-buf-generate.md, design/issues/summary.md, design/issues/I-0008-flai-check-finds-roles-baseline-outside-the-story-at-close-out.md, design/issues/I-0009-flai-check-finds-roles-definition-outside-the-story-at-close-out.md, design/issues/I-0010-flai-check-finds-item-blocked-closed-outside-the-story-at-close-out.md]
date: 2026-11-06
status: active
absorbed_by: ""
asked_again: []
cited: []
---

# D-0024 S-0031: close-out's test tier fails on the BSR rate limit; how should it pass: I attempted to set the BSR_AUTH_TOKEN and BUF_TOKEN in the local environment

The operator, alex, answered TH-0021 at 2026-11-06T07:39:33Z.

What was asked: S-0031's close-out stopped three times at the test tier because `buf generate` hit the unauthenticated BSR rate limit (`resource_exhausted: too many requests`, I-0005, I-0006, I-0007); no BUF_TOKEN is set in the close-out environment. Options: (1, recommended) set BUF_TOKEN in flai serve's environment from the CI secret BSR_AUTH_TOKEN and re-run; (2) pull S-0020 (stubs from `proto/` by directory) ahead; (3) accept S-0031 on its targeted evidence.

The answer:

```text
I attempted to set the BSR_AUTH_TOKEN and BUF_TOKEN in the local environment
```

What it settles: the operator took option 1. The BSR credential belongs in the environment of the process that runs the close-out (BUF_TOKEN, held as the secret BSR_AUTH_TOKEN), not in the repository. The close-out then passed in 1m29s and S-0031 was accepted. The token's value is not recorded here.

Source: TH-0021.
