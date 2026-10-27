---
id: T-0030
type: task
nature: improvement
title: "Runbook: operator checklist of secrets, trusted publishers, protection, and archiving"
status: backlog
parent: S-0008
owner: alex
created: 2026-10-27T22:25:34Z
updated: 2026-10-27T22:25:34Z
transitions: []
stream: S-0008
tags: [release, github, docs]
touches: [docs/operators/release.md, scripts/README.md]
after: [T-0024, T-0025, T-0027]
---
# T-0030 Runbook: operator checklist of secrets, trusted publishers, protection, and archiving

## Work

Extend `docs/operators/release.md`, as S-0005 and S-0007 left it on main, with a setup checklist, and list the three scripts in `scripts/README.md`. It waits for the three script tasks because it names their flags and the job names `protect-main.sh` reads.

- A checklist table with a checkbox per item: every secret (`BUF_TOKEN`, `DOCKERHUB_USERNAME`, `DOCKERHUB_TOKEN`, `CARGO_REGISTRY_TOKEN`, `NUGET_USER`, `HEX_API_KEY`, and any the client stories added) with where it is created, the scope it needs, and the command that sets it (`scripts/set-release-secrets.sh set`, values from the environment); every trusted publisher (PyPI, npm, NuGet) with the registry page, the repository `bytepunx/signet`, and the exact workflow filename `release-client-<language>.yml`, as registered on the registry's side since no script can do it.
- The date-and-tick convention: the operator ticks each item in S-0008's notes with the date when done.
- Branch protection: the required checks, `scripts/protect-main.sh` with and without `--apply`, why administrators are not enforced and no review is required, and that path filters in `ci.yml` must skip at job level so a required check always reports.
- Archiving: the order (publish a release from this repository first, confirm it, then `scripts/archive-old-repos.sh --apply`), what the pointer README says, and how to undo an archive from the repository settings.
- Keep the file's front matter and its other sections as they are; bump `updated`.

## Done when

- `docs/operators/release.md` has the checklist with a row for each secret and trusted publisher the workflow stories settled on, each naming where it is registered and which workflow file it names.
- The branch protection and archiving sections name the scripts and their flags exactly as the scripts take them.
- `scripts/README.md` lists the three scripts.
- `flai check --strict` and the markdown lint are clean.

## Notes

The operator's own steps (criteria 2 to 4 of S-0008) happen after this task and are not a task: the story's agent hands the operator the checklist and the commands, and ticks the criteria only as the operator reports each done.
