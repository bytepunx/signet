---
id: T-0066
type: task
nature: improvement
title: Point SignetClient.csproj at bytepunx/signet and drop the hard-coded version
status: done
parent: S-0015
owner: alex
created: 2026-10-27T22:52:29Z
updated: 2026-11-06T07:54:42Z
transitions:
  - to: ready
    at: 2026-11-06T07:54:34Z
    by: agent-S-0015
  - to: in-progress
    at: 2026-11-06T07:54:34Z
    by: agent-S-0015
  - to: done
    at: 2026-11-06T07:54:42Z
    by: agent-S-0015
stream: S-0015
tags: [client-csharp, release]
touches: [client-csharp/SignetClient.csproj]
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
# T-0066 Point SignetClient.csproj at bytepunx/signet and drop the hard-coded version

## Work

Edit `client-csharp/SignetClient.csproj`:

- Set `RepositoryUrl` to `https://github.com/bytepunx/signet`, add `RepositoryType` `git`, and add `PackageProjectUrl` `https://github.com/bytepunx/signet/tree/main/client-csharp`, so the NuGet page links to this repository and the component's folder.
- Remove the `<Version>0.6.0</Version>` line. ADR-0001 says no version file is bumped in a commit and the release workflow overrides the value with `dotnet pack -p:Version=X.Y.Z`; a checked-in 0.6.0 would only be a stale claim. Add an XML comment where it was, saying the release workflow sets the version from the `client-csharp/vX.Y.Z` tag.
- Reword `Description` so it no longer says the client is generated from `bytepunx/signet-proto` (the stubs come from the `proto/` folder of this repository, or the BSR for now); keep it one sentence.

This task does not wait for any other: the workflow task and this one touch different files.

## Done when

- `RepositoryUrl` and `PackageProjectUrl` name `bytepunx/signet`; the file contains no `signet-clients` or `signet-proto` URL and no `<Version>` element.
- With the stubs generated (`scripts/buf-generate.sh client-csharp`, or `buf generate` in `client-csharp/`), `dotnet pack SignetClient.csproj -c Release -p:Version=1.2.3` in the `mcr.microsoft.com/dotnet/sdk:8.0` image produces `Signet.Client.1.2.3.nupkg`, and its `.nuspec` names the repository URL above.
- `dotnet build SignetClient.sln` still succeeds.

## Notes

Layer 1 of the plan.
