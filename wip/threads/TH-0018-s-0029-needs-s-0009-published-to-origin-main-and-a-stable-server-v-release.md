---
id: TH-0018
title: "S-0029 needs S-0009 published to origin/main and a stable server/v* release"
anchor:
  path: wip/kanban/stories/S-0029-verify-publish-server-main-yml-and-install-sh-after-s-0009-reaches-main.md
  item: S-0029
status: resolved
participants: [agent-S-0029, alex]
created: 2026-11-05T22:46:11Z
updated: 2026-11-06T06:57:31Z
---

# TH-0018 S-0029 needs S-0009 published to origin/main and a stable server/v* release

On wip/kanban/stories/S-0029-verify-publish-server-main-yml-and-install-sh-after-s-0009-reaches-main.md.

## Entries

### 2026-11-05T22:46:11Z agent-S-0029
Neither criterion of S-0029 can be checked yet, and both blockers are yours to clear.

State found:
- origin/main is at 9b60c6c. Local main is 115 commits ahead, so publish-server-main.yml and release-server.yml (S-0009) are not on the remote. The only release-server run so far was the prerelease tag server/v0.0.0-test from a branch.
- `gh release list` shows only v0.x releases from the old layout. There is no stable server/vX.Y.Z release, so install.sh has nothing to resolve.

What I need from you:
1. Publish the accepted work (git fetch, then flai release --pending). I will not push to origin/main myself.
2. Cut a stable server/vX.Y.Z release. This is your call: the notes on S-0029 say so.

What I will do once both are done:
- Criterion 1: look at the publish-server-main run for the publishing push, then push a change touching only non-server paths and confirm no run starts. For this step I need your go-ahead to push to main, or you can do it and tell me.
- Criterion 2: run install.sh on linux and darwin against the repo, with SIGNET_VERSION unset, and confirm it picks the newest server/v* release and ignores other components' releases.
- Fix either workflow or script if it fails.

I have blocked S-0029 until this is answered. Reply here when the push and release are done, or if you want it handled differently.

### 2026-11-05T23:44:43Z alex
as of now, there is no longer the CI machinery in git to cut a release, how would I do that?

### 2026-11-05T23:45:10Z agent-S-0029
The CI machinery for cutting a release is still in git. It is the tag push plus flai, not a release-please PR. The release steps are in docs/operators/release.md.

How it works:
- `.github/workflows/release-server.yml` fires on a pushed `server/v*` tag. It tests, publishes the CLI binaries, images and Helm chart, and creates the GitHub Release.
- `flai release --pending` writes the annotated `<component>/vX.Y.Z` tags and pushes the branch and the tags together.
- Release-please is retired (S-0006). Its config exists only in the old `origin/main` layout.

One catch: the baseline tags are not in this clone yet. `git tag -l 'server/*'` is empty, and so is the remote. A dry run today plans `server 0.0.0 -> 0.0.1`, so a release now would be server/v0.0.1 instead of continuing from v0.11.0. The same dry run also plans `client-elixir 0.0.0 -> 0.1.0`.

Steps, from a checkout of `main` with push rights:
1. Create the baseline tags. docs/operators/release.md, "Push the baseline tags", has the exact `git tag -a` commands. For the server it is `git tag -a server/v0.11.0 db3b802 -m "Baseline: continues signet v0.11.0"`. Push them with `git push origin <tags>`. They start no workflow.
2. `git fetch`.
3. `flai release --pending --dry-run`. The server should now plan 0.11.0 -> 0.11.1.
4. `flai release --pending`. It pushes `main`, which also runs `publish-server-main.yml` for the server changes, and tags `server/v0.11.1`, which runs `release-server.yml`.

Alternative: if you would rather skip the baselines and want a specific number, a tag you push by hand also starts the workflow, for example `git tag -a server/v0.12.0 -m "..." && git push origin server/v0.12.0`. The workflow only needs the tag to match `server/vX.Y.Z`. flai will then count from it.

Before you push, check the secrets with `scripts/set-release-secrets.sh check`. `release-server.yml` and `publish-server-main.yml` need `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN`. The release workflow's test job also needs `BSR_AUTH_TOKEN` (I-0006).

Once `server/vX.Y.Z` is released, tell me. I will then check the `publish-server-main` run, push a non-server-only change for the path-filter check if you say so, and run `install.sh` on linux and darwin.

Source: docs/operators/release.md § Publish

### 2026-11-06T06:57:31Z alex
Resolved.
