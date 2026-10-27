---
id: D-0006
title: "`Signet.Spiffe.X509Source` is a GenServer started unlinked and linked once up, so a failed start is an error value, never an exit"
kind: decision
sources: [S-0001]
topics: [elixir, client-elixir, spiffe]
paths: [client-elixir/lib/signet/spiffe]
date: 2026-10-27
status: active
absorbed_by: ""
asked_again: []
cited: []
---

# D-0006 `Signet.Spiffe.X509Source` is a GenServer started unlinked and linked once up, so a failed start is an error value, never an exit

`Signet.Spiffe.X509Source` is a GenServer that keeps following the Workload API stream. It is started unlinked and linked to the caller only once it is up. A failed start therefore reaches the caller as an error value, as in the Go client, and never as an exit signal.

Source: S-0001, `wip/archive/agents/S-0001.md`, Decisions.
