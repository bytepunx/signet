---
id: TH-0021
title: "S-0031: close-out's test tier fails on the BSR rate limit; how should it pass?"
anchor:
  path: wip/kanban/stories/S-0031-scripts-test-client-csharp-sh-crashes-in-dotnet-when-the-caller-s-uid-has-no.md
  item: S-0031
status: resolved
participants: [agent-S-0031, alex]
created: 2026-11-06T07:35:18Z
updated: 2026-11-06T07:44:51Z
---

# TH-0021 S-0031: close-out's test tier fails on the BSR rate limit; how should it pass?

On wip/kanban/stories/S-0031-scripts-test-client-csharp-sh-crashes-in-dotnet-when-the-caller-s-uid-has-no.md.

## Entries

### 2026-11-06T07:35:18Z agent-S-0031
S-0031's fix is committed (T-0094, scripts/test-client-csharp.sh) and verified: 56/56 pass from an empty .cache/dotnet with id shimmed to uid 1001. Criteria 1 and 3 are ticked. Criterion 2 (green client-csharp job in ci.yml) can only be confirmed on a CI run after the branch reaches main.

The close-out has stopped three times at the test tier. Each time buf generate failed with `resource_exhausted: too many requests`, from the unauthenticated BSR, across server, client-go, client-python, client-typescript, client-rust and client-csharp. scripts/test.sh runs about six stub generations in under a minute, and waiting 5 to 10 minutes between runs did not help. No BUF_TOKEN is set in the close-out environment. This is I-0005, I-0006 and I-0007 (I bumped I-0007 to count 3). The change itself is not the cause.

Options:
1. (Recommended) Set BUF_TOKEN in flai serve's environment, with the token held as the CI secret BSR_AUTH_TOKEN, and I re-run the close-out. This also clears I-0005 for later stories.
2. Pull S-0020 ahead (stubs generated from proto/ by directory, no BSR). It is held behind S-0011, so it is a bigger change than this story.
3. You accept S-0031 on the targeted evidence (my passing flai test on scripts/test-client-csharp.sh), skipping a green whole-suite close-out.

On option 1, I re-run the close-out when you answer and move S-0031 to review if it passes. On option 2 or 3, I move S-0031 to review with the close-out's last verdict noted in the narrative.

### 2026-11-06T07:39:33Z alex
I attempted to set the BSR_AUTH_TOKEN and BUF_TOKEN in the local environment

### 2026-11-06T07:44:51Z alex
Resolved.
