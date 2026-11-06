---
id: T-0093
type: task
nature: research
title: "Verify install.sh installs the newest server/v* release on linux and darwin"
status: backlog
parent: S-0029
owner: alex
created: 2026-11-06T06:59:16Z
updated: 2026-11-06T06:59:16Z
transitions: []
stream: S-0029
tags: [server]
touches: [install.sh]
after: [T-0091]
---
# T-0093 Verify install.sh installs the newest server/v* release on linux and darwin

## Work

Check criterion 2 against the real release made in T-0091. It waits for T-0091 because `install.sh` resolves its version from the GitHub releases list, and that list has no `server/v*` release until T-0091 is done.

Run `install.sh` with `SIGNET_VERSION` unset on darwin (this host, with `INSTALL_DIR` writable or sudo) and on linux (a container, for amd64 and, if the host runs it, arm64). Confirm each run prints `Version: server/vX.Y.Z` for the newest stable `server/v*` release, verifies the checksum, installs `signet`, and that `signet --version` reports that version.

Confirm it ignores other components' releases: `gh release list` shows `v0.x` releases from the old layout and, once other components release, `client-*/v*` ones. The newest release overall must not be what it picks unless that is the newest `server/v*`. Also confirm that a prerelease tag is skipped and that `SIGNET_VERSION` pins a version in each of its three forms.

Run the script in a scratch directory with a scratch `INSTALL_DIR` or in a container, so that it does not replace a `signet` binary on the operator's PATH. If `install.sh` fails, fix it and run the check again.

## Done when

- On darwin and on linux, `install.sh` with `SIGNET_VERSION` unset installs the newest stable `server/v*` release.
- It is shown to ignore releases of other components and prereleases.
- Criterion 2 is ticked through flai; any fix to `install.sh` is committed.

## Notes
