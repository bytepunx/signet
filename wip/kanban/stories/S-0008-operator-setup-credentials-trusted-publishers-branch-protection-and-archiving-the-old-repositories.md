---
id: S-0008
type: story
nature: improvement
title: "Operator setup: credentials, trusted publishers, branch protection, and archiving the old repositories"
status: ready
parent: E-0002
owner: arobson
created: 2026-10-27T19:47:49Z
updated: 2026-10-27T23:01:46Z
transitions:
  - to: ready
    at: 2026-10-27T22:54:41Z
    by: alex
tags: []
topics: [release, github]
touches: [docs/operators/release.md, scripts/README.md, scripts/set-release-secrets.sh, scripts/protect-main.sh, scripts/archive-old-repos.sh]
after: [S-0005, S-0007]
agent:
  harness: claude-code
  model: claude-sonnet-5-5
forecast:
  duration: 2h
  delivery: 2026-10-29T07:40:00Z
  basis: "Its own forecast of 2h; 1st in the pull order with an in-progress limit of 2, behind S-0007."
  by: flai
  at: 2026-10-27T23:01:46Z
---
# S-0008 Operator setup: credentials, trusted publishers, branch protection, and archiving the old repositories

## Goal

The `bytepunx/signet` repository can publish every component: the Actions secrets the release workflows need exist (`BUF_TOKEN`, `DOCKERHUB_USERNAME`, `DOCKERHUB_TOKEN`, `CARGO_REGISTRY_TOKEN`, `NUGET_USER`, `HEX_API_KEY`, and whatever the client stories settle on), PyPI, NuGet, and npm trusted publishing are registered against this repository and the new workflow filenames, `main` is protected with the CI jobs as required checks, and `signet-clients`, `signet-proto`, and `signet-admin` are archived on GitHub with a README pointing here.

## Acceptance criteria
- [ ] `docs/operators/release.md` holds the checklist of secrets and trusted publishers per component, each with where it is registered and which workflow file it names
- [ ] Every item on that checklist is done by the operator and ticked in the story notes with the date
- [ ] Branch protection on `main` requires the CI jobs and the system-flow check
- [ ] The three old repositories are archived and their READMEs point to this repository

## Tasks
- T-0024 Script: set and check the repository's release secrets
- T-0025 Script: protect main with the CI jobs as required checks
- T-0027 Script: point the old repositories here and archive them
- T-0030 Runbook: operator checklist of secrets, trusted publishers, protection, and archiving

## Notes

The agent writes the checklist and the commands (`gh secret set`, `gh api .../branches/main/protection`, `gh repo archive`); creating credentials and archiving repositories is the operator's (safety.md).

### Planning

Touches, by where each came from:

- `docs/operators/release.md`: declared. S-0005 creates it and S-0007 adds the job names, so this story waits for both.
- `scripts/set-release-secrets.sh`, `scripts/protect-main.sh`, `scripts/archive-old-repos.sh`: design. The goal names `gh secret set`, `gh api .../branches/main/protection`, and `gh repo archive`, and the tooling convention puts command sequences in `scripts/`.
- `scripts/README.md`: layout. It indexes the scripts folder, so new scripts are listed there. S-0007 also touches it, which is why this story waits for S-0007.
- No folder touches are kept. `flai touches suggest` found no co-change history for the declared path.

After: `S-0005` was declared. `S-0007` is added because `protect-main.sh` must name the job names its accepted `ci.yml` defines.

Forecast: 2h, not the 1h default. The default assumes no history and a size of 5; this story has four tasks and writes three shell scripts that need a dry run and no secret echo. Delivery is the agent's part only. The operator's steps (criteria 2 to 4) take as long as the operator takes and sit outside the forecast.

Cost of delay: not set. The story has no inputs and neither has E-0002; the plan thread asks for them.

Criteria 2 to 4 are done by the operator, not by a task. The story's agent hands over the checklist and commands and ticks each criterion only when the operator reports it done.
