---
id: MS-0008
title: "S-0017 holds S-0011 on overlap alone: may S-0011 start beside it?"
from: S-0011
to: S-0017
about: [client-go/README.md]
status: closed
participants: [flai, agent-S-0017, alex]
created: 2026-11-06T08:17:45Z
updated: 2026-11-06T08:22:48Z
---

# MS-0008 S-0017 holds S-0011 on overlap alone: may S-0011 start beside it?

Between S-0011 and S-0017, about `client-go/README.md`.

## Entries

### 2026-11-06T08:17:45Z flai S-0011
S-0017 holds S-0011 on overlap alone: may S-0011 start beside it?

S-0011, "Go client: module path for the mono-repo and release workflow on client-go/v* tags", is ready, and its claim overlaps S-0017's on `client-go/README.md`. It has no agent until it starts, so flai asks for it (ADR-0134).

Its goal:

> The Go client is `github.com/bytepunx/signet/client-go`, resolvable with `go get github.com/bytepunx/signet/client-go@vX.Y.Z` because flai's `client-go/vX.Y.Z` tag is exactly the tag form Go requires for a module in a subdirectory. `release-client-go.yml` runs on that tag: regenerates stubs, vets, tests, asks `proxy.golang.org` to index the version, and creates the GitHub Release. The README's install and publish sections describe the new path and tags.

Answer in one of three ways:

1. Narrow your touches with `flai touches S-0017 --remove <path>`, or on the task that names the path, if S-0017 will not change it: the hold clears when the claims no longer overlap.
2. Share the paths with `flai message share MS-0008 --paths <path> "<split>"`, or the MCP tool `message_share`, saying who changes what, if the two stories can change them apart: the overlap on them no longer holds, and S-0011 can start.
3. Reply with `message_reply` saying why the hold stands: S-0011 waits until S-0017 moves to review, is cancelled, or is sent back.

### 2026-11-06T08:17:49Z agent-S-0017 S-0017 (settled)
Yes, S-0011 may start beside S-0017. My only change to client-go/README.md is committed (46de46a): the line 4 link to the signet-proto repository became a link to the mono-repo's proto/ folder, and the Go client link fixed in other READMEs. S-0017 makes no further edits there.

### 2026-11-06T08:22:48Z alex
Closed: S-0017 was accepted
