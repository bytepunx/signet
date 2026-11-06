---
id: D-0030
title: "Old `bytepunx/signet-clients#NN` issue references in client code and tests stay: they name history, as ADR-0001 keeps `CHANGELOG.md`"
kind: decision
sources: [S-0011]
topics: [release, client-go, client-python, client-rust, client-typescript, client-csharp, client-elixir]
paths: [client-go/client_test.go, client-go/sops_encrypt.go, design/adrs/0001-the-mono-repo-releases-from-flai-tags-per-component-with-github-actions.md]
date: 2026-11-06
status: active
absorbed_by: ""
asked_again: []
cited: []
---

# D-0030 Old `bytepunx/signet-clients#NN` issue references in client code and tests stay: they name history, as ADR-0001 keeps `CHANGELOG.md`

When S-0011 renamed the Go module and retired the three-repository layout, `client_test.go` and `sops_encrypt.go` kept their `bytepunx/signet-clients#NN` issue references. They name history, not the current layout, so a rename of paths and install instructions does not rewrite them. ADR-0001 likewise keeps `CHANGELOG.md` as it was. A later client story that sweeps for the old repository name should leave such issue references and changelog entries alone.

Source: S-0011, `wip/archive/agents/S-0011.md`, Decisions.
