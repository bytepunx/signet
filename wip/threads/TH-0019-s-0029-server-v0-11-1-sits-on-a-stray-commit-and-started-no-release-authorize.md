---
id: TH-0019
title: "S-0029: server/v0.11.1 sits on a stray commit and started no release; authorize retag and a push for the path-filter check"
anchor:
  path: wip/kanban/stories/S-0029-verify-publish-server-main-yml-and-install-sh-after-s-0009-reaches-main.md
  item: S-0029
status: open
participants: [agent-S-0029]
created: 2026-11-06T06:58:24Z
updated: 2026-11-06T06:58:24Z
---

# TH-0019 S-0029: server/v0.11.1 sits on a stray commit and started no release; authorize retag and a push for the path-filter check

On wip/kanban/stories/S-0029-verify-publish-server-main-yml-and-install-sh-after-s-0009-reaches-main.md.

## Entries

### 2026-11-06T06:58:24Z agent-S-0029
The publish is done, but server/v0.11.1 did not release. I need two decisions before I can verify either criterion.

Found, after `git fetch --tags`:
- origin/main is at 970e598, so S-0009's workflows are published.
- `server/v0.11.1` (annotated tag dd8c982, 2026-11-05 18:52) points at 130a92b "chore: commit wip". That commit is on no branch, and `release-server.yml` does not exist in it.
- No "Release server" run exists for `server/v0.11.1`. The only runs are the two `server/v0.0.0-test` ones. `gh release list` has no `server/v*` release, so `install.sh` still has nothing to resolve.

Criterion 1:
- The positive half is already observed. The pushes to c341f0a and 970e598 both changed `server/` files. "Publish server main" ran for each, c341f0a succeeded, and the 970e598 run was still in progress when I looked.
- The negative half is not. Every push so far touched `server/`, so I have not seen one that correctly starts no run.

Decisions:
1. Retag. I recommend deleting `server/v0.11.1` on the remote and locally, then tagging a commit on main that contains `release-server.yml` (970e598 or later) and pushing it. That starts the release workflow. This deletes a published tag, so it is yours, or I do it only if you say so. Alternative: leave 0.11.1 alone and push `server/v0.11.2` on main's head. That costs a version number but moves nothing.
2. Path-filter check. May I push a docs-only commit that touches no `server/` path, such as a note in this story's narrative, to main? Alternative: skip the extra push. The next publish after S-0029 is accepted will touch only `wip/`, and I confirm no run starts then. That means criterion 1 is ticked only after acceptance, so I would leave it unticked at review and say why in the notes.

Reply with your choice for each. Once a stable `server/v*` release exists and its workflow has succeeded, I will run `install.sh` on darwin and on linux through a container, tick criterion 2, and move the story to review.
