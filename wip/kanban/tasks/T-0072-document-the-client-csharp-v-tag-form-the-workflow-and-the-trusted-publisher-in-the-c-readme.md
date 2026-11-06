---
id: T-0072
type: task
nature: improvement
title: "Document the client-csharp/v tag form, the workflow, and the trusted publisher in the C# README"
status: in-progress
parent: S-0015
owner: alex
created: 2026-10-27T22:52:43Z
updated: 2026-11-06T07:54:34Z
transitions:
  - to: ready
    at: 2026-11-06T07:54:34Z
    by: agent-S-0015
  - to: in-progress
    at: 2026-11-06T07:54:34Z
    by: agent-S-0015
stream: S-0015
tags: [client-csharp, release, docs]
touches: [client-csharp/README.md]
after: [T-0066, T-0069]
usage:
  source: log
  seconds: 8
  estimated: true
  priced_by: estimate
  models:
    - model: claude-sonnet-5-5
      input: 2
      output: 13
      cache_read: 45333
      cache_write: 183
      cost: 0.0152
---
# T-0072 Document the client-csharp/v tag form, the workflow, and the trusted publisher in the C# README

## Work

Rewrite the publishing paragraph under `## Install` in `client-csharp/README.md`, which still says `publish-csharp.yml` publishes when release-please tags a `csharp-v*` release, and links `../.github/workflows/publish-csharp.yml`. It must now say, in plain sentences:

- The release tag is `client-csharp/vX.Y.Z`, created by `flai release --pending` and never by hand; the package version is the tag with `client-csharp/v` stripped, passed as `-p:Version=X.Y.Z` to `dotnet pack`, and `SignetClient.csproj` carries no version.
- `.github/workflows/release-client-csharp.yml` (link it with the correct relative path from `client-csharp/`) tests, packs, publishes `Signet.Client` to NuGet, and creates the GitHub Release, which is the changelog.
- NuGet publishing uses Trusted Publishing (OIDC, no stored API key): the policy on nuget.org names repository `bytepunx/signet` and workflow `release-client-csharp.yml`, and the `NUGET_USER` secret names the account. Link `docs/operators/release.md` for the setup.

Leave the other sections as they are; S-0020 edits the "Regenerating stubs" section and S-0017 the old-repository links, so keep to the Install section. It waits for the csproj and workflow tasks so the text names what exists.

## Done when

- The README names the tag form `client-csharp/vX.Y.Z`, the file `release-client-csharp.yml`, and the trusted publisher (repository, workflow filename, `NUGET_USER`).
- It no longer mentions `publish-csharp.yml`, release-please, or a `csharp-v*` tag, and its links to the workflow and the runbook resolve.
- `scripts/lint-md.sh` is clean on the file.

## Notes

Layer 2 of the plan.
