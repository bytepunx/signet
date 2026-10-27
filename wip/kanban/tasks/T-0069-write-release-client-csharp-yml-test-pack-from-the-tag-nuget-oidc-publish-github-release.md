---
id: T-0069
type: task
nature: feature
title: "Write release-client-csharp.yml: test, pack from the tag, NuGet OIDC publish, GitHub Release"
status: backlog
parent: S-0015
owner: alex
created: 2026-10-27T22:52:35Z
updated: 2026-10-27T22:52:35Z
transitions: []
stream: S-0015
tags: [client-csharp, release, github]
touches: [".github/workflows/release-client-csharp.yml"]
---
# T-0069 Write release-client-csharp.yml: test, pack from the tag, NuGet OIDC publish, GitHub Release

## Work

Add `.github/workflows/release-client-csharp.yml`, modelled on signet-clients' `publish-csharp.yml` (read it with `gh api repos/bytepunx/signet-clients/contents/.github/workflows/publish-csharp.yml`) and on `design/system/release.md`:

- Trigger: `on: push: tags: ['client-csharp/v*']` and nothing else. Workflow-level `permissions: contents: read`; the one job gets `contents: write` (GitHub Release) and `id-token: write` (NuGet OIDC).
- Checkout with `fetch-depth: 0` and `fetch-tags: true`, so the tag annotation and the previous `client-csharp/v*` tag are available.
- Version: strip the `client-csharp/v` prefix from `GITHUB_REF_NAME` into `VERSION`, fail if the result is not `X.Y.Z` with an optional `-suffix`, and export it to `GITHUB_ENV`. A suffix makes the GitHub Release a pre-release.
- Test: run `scripts/test-client-csharp.sh` (S-0007 adds it; it generates the stubs and runs `dotnet test` in Docker), so the release tests exactly what CI tests.
- Pack: `actions/setup-dotnet` with `8.0.x`, then `dotnet pack client-csharp/SignetClient.csproj --configuration Release --output ./nupkg -p:Version="$VERSION"`. Pack the project, not the solution, since the solution holds the test and example projects. Pass any BSR token the stub generation needs the way `scripts/buf-generate.sh` documents.
- Publish: `NuGet/login@v1` with `user: ${{ secrets.NUGET_USER }}`, then `dotnet nuget push "./nupkg/*.nupkg" --api-key "${{ steps.login.outputs.NUGET_API_KEY }}" --source https://api.nuget.org/v3/index.json --skip-duplicate`. No long-lived NuGet key is stored. The trusted publisher on nuget.org is registered for `bytepunx/signet` and this exact filename (S-0008 does that).
- GitHub Release: create it with `gh release create "$GITHUB_REF_NAME"`, titled by the tag, with `--notes-file` built from the tag annotation (`git tag -l --format='%(contents)' "$GITHUB_REF_NAME"`) followed by `git log --oneline` of the commits since the previous `client-csharp/v*` tag, limited to `client-csharp/`; `--prerelease` for a suffixed version; attach the `.nupkg`. Put the notes building in a script under `scripts/` only if it is longer than a few lines; do not touch other components' files.
- Commit nothing to `main`: no version bump, no changelog.

It does not wait for the csproj task: the workflow overrides the version on the command line whatever the file says.

## Done when

- The file triggers on `client-csharp/v*` tags only, and `actionlint` (or `yq` for syntax, if `actionlint` is not available in Docker) reports nothing.
- It passes `-p:Version=X.Y.Z` derived from the tag to `dotnet pack`, publishes through `NuGet/login` and `dotnet nuget push`, and creates the GitHub Release from the tag annotation with the `.nupkg` attached.
- It stores no NuGet API key and commits nothing.
- A comment at the top names the trusted publisher requirement: repository `bytepunx/signet`, this workflow filename, the `NUGET_USER` secret.

## Notes

Layer 1 of the plan.
