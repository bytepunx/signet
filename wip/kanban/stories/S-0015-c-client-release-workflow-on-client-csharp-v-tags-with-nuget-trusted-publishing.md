---
id: S-0015
type: story
nature: feature
title: "C# client: release workflow on client-csharp/v* tags with NuGet trusted publishing"
status: in-progress
parent: E-0002
owner: arobson
created: 2026-10-27T19:47:50Z
updated: 2026-11-06T07:53:57Z
transitions:
  - to: ready
    at: 2026-11-06T07:50:07Z
    by: alex
  - to: in-progress
    at: 2026-11-06T07:53:57Z
    by: agent-S-0015
tags: [client-csharp]
topics: [release, github]
touches: [".github/workflows/release-client-csharp.yml", client-csharp/SignetClient.csproj, client-csharp/README.md]
after: [S-0007]
agent:
  harness: claude-code
  model: claude-sonnet-5-5
usage:
  source: log
  seconds: 47
  estimated: true
  priced_by: estimate
  turns:
    - day: 2026-11-06
      ceremony: 1
      work: 8
  models:
    - model: claude-sonnet-5-5
      input: 22
      output: 129
      cache_read: 450254
      cache_write: 53750
      cost: 0.1678
forecast:
  duration: 2h
  delivery: 2026-11-06T14:29:00Z
  basis: "Its own forecast of 2h; 3rd in the pull order with an in-progress limit of 2, behind S-0012, S-0014, S-0020 and S-0011."
  by: flai
  at: 2026-11-06T07:53:46Z
---
# S-0015 C# client: release workflow on client-csharp/v* tags with NuGet trusted publishing

## Goal

A push of `client-csharp/vX.Y.Z` builds and tests `Signet.Client`, packs it with the version from the tag, publishes to NuGet with the OIDC login the old workflow used, and creates the GitHub Release. The project file names this repository.

## Acceptance criteria
- [ ] `release-client-csharp.yml` triggers on `client-csharp/v*` only, passes `-p:Version=X.Y.Z` to `dotnet pack`, publishes via `NuGet/login` and `dotnet nuget push`, and creates the GitHub Release with notes from the tag annotation
- [ ] `SignetClient.csproj` repository and project URLs name `bytepunx/signet` and no longer hard-code a version that would be overridden
- [ ] `client-csharp/README.md` names the tag form, the workflow, and the trusted publisher

## Tasks
- T-0066 Point SignetClient.csproj at bytepunx/signet and drop the hard-coded version
- T-0069 Write release-client-csharp.yml: test, pack from the tag, NuGet OIDC publish, GitHub Release
- T-0072 Document the client-csharp/v tag form, the workflow, and the trusted publisher in the C# README

## Notes

Reference: signet-clients' `publish-csharp.yml` on GitHub.

### Planning

Touches, by source:

- Declared: `.github/workflows/release-client-csharp.yml`, `client-csharp/SignetClient.csproj`, `client-csharp/README.md`. All kept, each a file; T-0069, T-0066 and T-0072 name one each.
- Design: `design/system/release.md` (the C# row and "Versions in packages") and `docs/operators/release.md` already say what the workflow, the trusted publisher and the `NUGET_USER` secret are, so neither needs an edit here.
- Co-change: `flai touches suggest` listed `client-csharp/CHANGELOG.md` (4 of 11 commits) and C# source files. None was taken: ADR-0001 retires `CHANGELOG.md` files in favour of the GitHub Release, and the source files have nothing to do with releasing. No folder touch is kept.

Overlaps to know of, not `after` reasons: S-0020 edits `client-csharp/README.md` (the "Regenerating stubs" section) and S-0017 its old-repository links; T-0072 keeps to the Install section. S-0006 retires release-please files and does not touch these.

After: S-0007 only, because T-0069 runs `scripts/test-client-csharp.sh`, which S-0007 creates. S-0008 (the NuGet trusted publisher and `NUGET_USER`) is needed only to publish for real, not to write the workflow, so no `after`. S-0020 is not needed: the scripts own stub generation.

Forecast: flai gave 12m (median of three done stories times size 6). Adjusted to 2h: the workflow derives the version, builds release notes from the annotation and the commits since the previous tag, and its pack has to be checked in Docker. The delivery moves to 2026-10-28T12:42:00Z with it.

No test tag is run: a `client-csharp/v0.0.0-test` push would publish a package to nuget.org, which can be unlisted but not deleted. The first real tag proves the workflow; the first release of this component is `flai release --pending`'s to cut once S-0008 is done.

Cost of delay: not set. `flai cod` stopped because neither the story nor E-0002 has the operator's inputs; asked on the plan thread.
