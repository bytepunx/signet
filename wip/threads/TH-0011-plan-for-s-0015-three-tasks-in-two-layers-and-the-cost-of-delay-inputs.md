---
id: TH-0011
title: "Plan for S-0015: three tasks in two layers, and the cost of delay inputs"
anchor:
  path: wip/kanban/stories/S-0015-c-client-release-workflow-on-client-csharp-v-tags-with-nuget-trusted-publishing.md
  item: S-0015
status: open
participants: [planner-S-0015]
created: 2026-10-27T22:53:23Z
updated: 2026-10-27T22:53:23Z
---

# TH-0011 Plan for S-0015: three tasks in two layers, and the cost of delay inputs

On wip/kanban/stories/S-0015-c-client-release-workflow-on-client-csharp-v-tags-with-nuget-trusted-publishing.md.

## Entries

### 2026-10-27T22:53:23Z planner-S-0015
Question first: S-0015 has no cost of delay inputs, and neither has E-0002, so `flai cod S-0015` refuses. I recommend time_lost_per_cycle 2h, with no revenue and no penalty: until the C# client can release from here, each NuGet release is cut by hand from the old repository. Alternatives: a penalty_per_week if a consumer waits on a NuGet release, or 0 if it can wait for S-0008. Set the inputs with `flai edit S-0015 --time-lost-per-cycle 2h` (or on E-0002 to cover all its stories) and I will work out the value on the next run.

Plan (touches and figures are in the story's Notes, under Planning):

Layer 1, no `after` between them, no shared files:
- T-0066 SignetClient.csproj: repository and project URLs name bytepunx/signet, the hard-coded `<Version>` goes, the description stops naming signet-proto.
- T-0069 Write release-client-csharp.yml: tag trigger `client-csharp/v*`, version from the tag, tests via scripts/test-client-csharp.sh, `dotnet pack -p:Version`, NuGet/login OIDC, `dotnet nuget push`, GitHub Release from the tag annotation and the commits since the previous tag.

Layer 2:
- T-0072 README Install section: tag form, workflow, trusted publisher (waits for both, so the text names what exists).

Story `after`: S-0007 only, for scripts/test-client-csharp.sh, which T-0069 calls.

Assumptions:
- The release job tests through scripts/test-client-csharp.sh (Docker, stubs included), then packs with setup-dotnet 8.0.x, as ADR-0001's model says.
- Removing `<Version>` is the reading of "no longer hard-codes a version"; local builds then get the SDK default 1.0.0.
- A version with a `-` suffix makes a pre-release GitHub Release; I added that for safety.
- The workflow needs the `NUGET_USER` secret and the nuget.org policy for `bytepunx/signet` and `release-client-csharp.yml`, which S-0008 sets up; no `after` on S-0008, since writing the workflow does not need them.
- No test-tag run task: pushing `client-csharp/v0.0.0-test` would publish a package to nuget.org, which can be unlisted but not deleted. The pack is checked in Docker in T-0066 instead; the workflow itself is proven by the first real tag.
- Stub generation may need a BSR token on the runner (the old workflow passed BSR_AUTH_TOKEN; the runbook lists BUF_TOKEN). T-0069 follows whatever scripts/buf-generate.sh documents; S-0020 may remove the need.
- client-csharp/CHANGELOG.md is left alone, per ADR-0001.

Forecast adjusted from flai's 12m to 2h, delivery 2026-10-28T12:42:00Z. I propose no split, merge, or drop.
