---
id: T-0080
type: task
nature: remediation
title: Point the client and proto READMEs at the mono-repo's proto/ folder
status: backlog
parent: S-0017
owner: alex
created: 2026-10-27T22:52:55Z
updated: 2026-10-27T22:52:55Z
transitions: []
stream: S-0017
tags: [docs]
touches: [proto/README.md, client-csharp/README.md, client-elixir/README.md, client-erlang/README.md, client-go/README.md, client-python/README.md, client-rust/README.md, client-typescript/README.md]
---
# T-0080 Point the client and proto READMEs at the mono-repo's proto/ folder

## Work

Each client README opens with a link to `github.com/bytepunx/signet-proto` as the place the schema lives, and `proto/README.md` is titled `signet-proto` and links `signet-clients`. In each file, change prose that describes a separate repository to describe the folder:

- `proto/README.md`: retitle to `proto`, say it is the schema folder of the signet repository published to the BSR as `buf.build/bytepunx/signet-proto`, and replace the `signet-clients` link with the `client-*` folders as relative links.
- Each of the seven client READMEs: the schema "lives in `proto/`" (relative link `../proto/README.md`) and is published to the BSR as `buf.build/bytepunx/signet-proto`. Keep the sentence about pinning if it is still true on this branch.
- Leave alone what belongs to other stories and to history: the `bytepunx/signet-clients#NN` issue references, which name issues in the archived repository; the Go module path `github.com/bytepunx/signet-clients/go` and its install line (S-0011); and any `buf.gen.yaml` input text (S-0020). Record in this task's Notes each README line left because of one of these.
- Do not touch the install or release sections of `client-elixir/README.md` and `client-erlang/README.md`: S-0006 edits them.

## Done when

- `grep -n "github.com/bytepunx/signet-proto" proto/README.md client-*/README.md` finds nothing.
- `grep -n "signet-clients" proto/README.md` finds nothing.
- Every remaining `signet-clients` mention in the seven client READMEs is an issue reference or the Go module path, and this task's Notes say which.
- `flai check --strict` and the markdown lint pass.

## Notes

Layer 1: it shares no path with another task. Its touches overlap S-0006 on `client-elixir/README.md` and `client-erlang/README.md`; the pull order settles that.
