---
id: T-0041
type: task
nature: improvement
title: "install.sh: install the newest server/v* release"
status: done
parent: S-0009
owner: alex
created: 2026-10-27T22:26:01Z
updated: 2026-10-28T05:55:48Z
transitions:
  - to: ready
    at: 2026-10-28T05:55:34Z
    by: agent-S-0009
  - to: in-progress
    at: 2026-10-28T05:55:34Z
    by: agent-S-0009
  - to: done
    at: 2026-10-28T05:55:48Z
    by: agent-S-0009
stream: S-0009
tags: [server, ci]
touches: [install.sh]
usage:
  source: log
  seconds: 14
  estimated: true
  priced_by: harness
  models:
    - model: claude-sonnet-5-5
      input: 1
      output: 310
      cache_read: 51749
      cache_write: 2711
      cost: 0.0191
---
# T-0041 install.sh: install the newest server/v* release

## Work

Replace the `releases/latest` lookup in `install.sh`, which can now return a client's release, with the newest `server/v*` release.

- List releases with `https://api.github.com/repos/bytepunx/signet/releases?per_page=100`, keep the tags that start `server/v` and are not pre-releases or drafts, and take the first, which the API returns newest first.
- Build the download and checksum URLs from the full tag, URL-encoding the slash (`server%2Fv0.12.0`), and print the version without the prefix.
- Honour an optional `SIGNET_VERSION` environment variable (a `server/vX.Y.Z` tag or `vX.Y.Z`) that skips the lookup, and keep the existing platform detection, checksum verification, and install steps unchanged.
- Parse the JSON with `grep` and `sed` as the script does today, so it keeps depending on nothing beyond `curl`.
- Test it against the existing GitHub Releases, whose tags `v0.2.0` to `v0.11.0` have no `server/` prefix: the script must fail with a clear message until the first `server/v*` release exists, and must not fall back to a client's release.

## Done when

- `install.sh` selects the newest `server/v*` release and ignores every other component's releases.
- Running it on linux and darwin (amd64 or arm64 hosts or a container) resolves the right asset URLs, shown by a dry run or a `SIGNET_VERSION` run against a test release.
- `shellcheck install.sh` is clean if shellcheck is available, and `flai check --strict` is clean.

## Notes
