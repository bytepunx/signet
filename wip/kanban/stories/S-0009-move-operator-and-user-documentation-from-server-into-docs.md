---
id: S-0009
type: story
nature: improvement
title: Move operator and user documentation from server/ into docs/
status: ready
parent: E-0001
owner: alex
created: 2026-10-24T22:33:36Z
updated: 2026-10-24T22:33:48Z
transitions:
  - to: ready
    at: 2026-10-24T22:33:48Z
    by: claude
tags: []
---
# S-0009 Move operator and user documentation from server/ into docs/

## Goal

Users and operators find every guide under `docs/<audience>/`: getting started, CLI reference, configuration index, policies, the secrets workflow, the restart lock, installation, SOPS and GitHub webhook setup, and the Helm chart reference, so `server/docs/` and `server/deploy/docs/` are gone and the `docs/` placeholders are real.

## Acceptance criteria
- [ ] `server/docs/getting-started.md`, `cli.md`, `secrets-workflow.md`, `restart-lock.md`, and `policies.md` live under `docs/users/` or `docs/operators/` by audience, with front matter, and the two index files link them.
- [ ] `server/docs/installation.md`, `configuration.md`, `server/deploy/docs/sops-setup.md`, and `github-webhook-setup.md` live under `docs/operators/`, with the configuration index covering every flag, `SIGNET_*` variable, stored setting, and secret as `design/conventions/documentation.md` requires.
- [ ] `docs/operators/` has runbooks for install, upgrade, deletion, backup, restore, and migration, each at least stating the procedure or that it is not yet defined and why.
- [ ] `server/deploy/helm/signet/README.md` stays with the chart (it ships in the package) and `docs/operators/` links to it rather than copying it.
- [ ] `server/docs/` and `server/deploy/docs/` no longer exist; `server/README.md` links to `docs/` instead.
- [ ] Every relative link in the moved files resolves; `make lint-md` passes.

## Tasks

## Notes

`docs/README.md` already defines the three audiences. Paths users already have bookmarked (GitHub URLs to `docs/*.md` in the old repo) will break once the old repository is archived; S-0011 adds the pointer.
