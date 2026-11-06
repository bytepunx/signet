---
id: TH-0022
title: "S-0029: close-out's test tier fails on the BSR rate limit, as in TH-0021"
anchor:
  path: wip/kanban/stories/S-0029-verify-publish-server-main-yml-and-install-sh-after-s-0009-reaches-main.md
  item: S-0029
status: open
participants: [agent-S-0029]
created: 2026-11-06T07:37:46Z
updated: 2026-11-06T07:37:46Z
---

# TH-0022 S-0029: close-out's test tier fails on the BSR rate limit, as in TH-0021

On wip/kanban/stories/S-0029-verify-publish-server-main-yml-and-install-sh-after-s-0009-reaches-main.md.

## Entries

### 2026-11-06T07:37:46Z agent-S-0029
S-0029 has both criteria verified and ticked, and everything is committed. The close-out stops at the test tier, so I cannot move it to review.

- The close-out ran at f2b2e47 on a docs-only branch and stopped at the test tier after 40s. scripts/test.sh failed server, client-go, client-python, client-typescript, client-rust and client-csharp. Only client-elixir passed.
- Running scripts/test-client-go.sh in the worktree gives `Failure: resource_exhausted: too many requests` from buf generate. This is the same cause as TH-0021 on S-0031, and it is recorded on I-0007 (count 3). The change is not the cause.

Options, as in TH-0021:
1. (Recommended) Set BUF_TOKEN in flai serve's environment, taking the value from the CI secret BSR_AUTH_TOKEN, then I ask for the close-out again. One answer on TH-0021 covers both stories.
2. Accept S-0029 without a green whole-suite close-out. Its diff is two docs lines plus issue records, and its evidence is the GitHub runs, releases and install runs in the narrative's log.

Reply with 1 or 2 here, or answer TH-0021 and tell me it applies. On 1 I ask for the close-out again and move the story to review if it passes. On 2 I move the story to review with the last verdict noted in the narrative.
