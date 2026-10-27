---
id: S-0015
type: story
nature: feature
title: "C# client: release workflow on client-csharp/v* tags with NuGet trusted publishing"
status: backlog
parent: E-0002
owner: arobson
created: 2026-10-27T19:47:50Z
updated: 2026-10-27T19:48:21Z
transitions: []
tags: [client-csharp]
topics: [release, github]
touches: [".github/workflows/release-client-csharp.yml", client-csharp/SignetClient.csproj, client-csharp/README.md]
after: [S-0007]
---
# S-0015 C# client: release workflow on client-csharp/v* tags with NuGet trusted publishing

## Goal

A push of `client-csharp/vX.Y.Z` builds and tests `Signet.Client`, packs it with the version from the tag, publishes to NuGet with the OIDC login the old workflow used, and creates the GitHub Release. The project file names this repository.

## Acceptance criteria
- [ ] `release-client-csharp.yml` triggers on `client-csharp/v*` only, passes `-p:Version=X.Y.Z` to `dotnet pack`, publishes via `NuGet/login` and `dotnet nuget push`, and creates the GitHub Release with notes from the tag annotation
- [ ] `SignetClient.csproj` repository and project URLs name `bytepunx/signet` and no longer hard-code a version that would be overridden
- [ ] `client-csharp/README.md` names the tag form, the workflow, and the trusted publisher

## Tasks

## Notes

Reference: signet-clients' `publish-csharp.yml` on GitHub.
