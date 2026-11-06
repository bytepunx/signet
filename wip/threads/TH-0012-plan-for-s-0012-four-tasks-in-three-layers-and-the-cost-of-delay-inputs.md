---
id: TH-0012
title: "Plan for S-0012: four tasks in three layers, and the cost of delay inputs"
anchor:
  path: wip/kanban/stories/S-0012-python-client-release-workflow-on-client-python-v-tags-with-pypi-trusted-publishing.md
  item: S-0012
status: resolved
participants: [planner-S-0012, alex]
created: 2026-10-27T22:53:24Z
updated: 2026-11-06T07:54:47Z
---

# TH-0012 Plan for S-0012: four tasks in three layers, and the cost of delay inputs

On wip/kanban/stories/S-0012-python-client-release-workflow-on-client-python-v-tags-with-pypi-trusted-publishing.md.

## Entries

### 2026-10-27T22:53:24Z planner-S-0012
Question first: S-0012 has no cost of delay inputs, and neither has E-0002, so `flai cod S-0012` refuses. I recommend time_lost_per_cycle 1h, with no revenue and no penalty: until this ships, a Python release is built and uploaded by hand and no GitHub Release exists for it. Alternatives: 0 if releases can wait for the server and proto workflows, or a penalty_per_week if a consumer waits on a package that names this repository. Set the inputs with `flai edit S-0012 --time-lost-per-cycle 1h` (or on E-0002 to cover all its stories) and I will work out the value on the next run.

Plan (touches and figures are in the story's Notes, under Planning):

Layer 1, no `after` between them, no shared files:
- T-0059 pyproject.toml: placeholder version 0.0.0, [project.urls] naming bytepunx/signet, description no longer naming signet-proto.
- T-0060 scripts/set-client-python-version.sh: strips the client-python/v prefix, validates a PEP 440 version, rewrites the version line.

Layer 2:
- T-0064 release-client-python.yml: jobs test, build, publish (OIDC, id-token write only), github-release. Waits for T-0059 and T-0060, whose placeholder and script it uses.

Layer 3:
- T-0067 client-python/README.md: tag form, workflow, trusted publisher. Waits for T-0064 so it names a workflow that exists.

Story `after`: S-0007 only, for scripts/test-client-python.sh, which T-0064 calls.

Assumptions:
- Version comes from an explicit rewrite, not hatch-vcs: eight components share the tag namespace and hatch-vcs would need custom tag matching and full-history checkouts. The criterion allows either.
- Only X.Y.Z with an optional a, b, or rc suffix is accepted, since PyPI rejects other pre-release forms after the build. A tag like client-python/v0.0.0-test fails in the version step.
- No safe test publish: criterion 2 limits the trigger to client-python/v* tags, so there is no dry-run trigger and any matching tag uploads to the real PyPI. The tasks check the wheel and the release notes locally in Docker and a scratch clone. The first live run is the first release (client-python/v0.5.0 baseline from S-0006, then the next flai bump). Say if you want a TestPyPI rehearsal instead; that would be an added story.
- The trusted publisher has no environment, as with the old publish-python.yml; the job declares none.
- Stubs are gitignored, so the build job runs buf generate and checks the wheel holds signet/v1, admin/v1, and signet_client. If S-0020 lands first, the workflow follows buf.gen.yaml as merged.
- CHANGELOG.md is not touched; ADR-0001 retires it.
- Folder touches: none.
- Forecast 2h (flai's 14m ignored workflow work); delivery 2026-10-28T12:02Z.

I propose no split, merge, or drop.

### 2026-11-06T07:54:47Z alex
Resolved: S-0012 was accepted
