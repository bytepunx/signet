---
id: S-0005
type: story
nature: improvement
title: Rebuild CI as root workflows with one job per sub-project
status: backlog
parent: E-0001
owner: alex
created: 2026-10-24T22:33:36Z
updated: 2026-10-24T22:33:36Z
transitions: []
tags: []
after: [S-0002, S-0004]
---
# S-0005 Rebuild CI as root workflows with one job per sub-project

## Goal

GitHub Actions run from `.github/workflows/` at the repository root and cover everything the three source repositories' CI covered, filtered by changed paths, so a pull request is checked for every project it touches and the workflows under `server/.github` and `proto/.github` are gone.

## Acceptance criteria
- [ ] A root CI workflow runs, on push to `main` and on pull requests, path-filtered jobs for: server test, lint, build, Docker build, and the Helm lint and template assertions from `server/.github/workflows/ci.yml`; proto `buf lint` and `buf breaking` against the last `proto/v*` tag; and Go, Python, TypeScript, Rust, and C# client build and test jobs matching the five jobs of the old signet-clients `ci.yml`.
- [ ] A generated-code drift check runs `make proto` and fails when committed stubs differ, replacing the auto-commit step of the old server workflow.
- [ ] The existing `system-flow-check.yml` still runs and passes alongside the new workflows.
- [ ] `server/.github/` and `proto/.github/` are removed; `FUNDING.yml` moves to the root `.github/` if the operator keeps it.
- [ ] `.github/pull_request_template.md` exists at the root as `design/conventions/git.md` requires.
- [ ] The old `smoke-images.yml` (per-language echo example images) is either carried over as a root workflow or recorded in S-0006's notes as release-time work.
- [ ] Every job is green on a pull request that touches each project, and the run links are in the story notes.

## Tasks

## Notes

The old signet-clients root files are not in git history here; fetch them from `https://github.com/bytepunx/signet-clients/tree/main/.github/workflows`. `proto-bump.yml` (nightly bump of the BSR pin) becomes unnecessary once S-0002 lands.
