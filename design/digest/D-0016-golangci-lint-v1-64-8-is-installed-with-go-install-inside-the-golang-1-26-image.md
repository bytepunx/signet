---
id: D-0016
title: "golangci-lint v1.64.8 is installed with go install inside the golang:1.26 image, not run from the golangci-lint v1 image"
kind: decision
sources: [S-0007, I-0003]
topics: [ci, tooling]
paths: [scripts/test-server.sh, scripts/test-client-go.sh]
date: 2026-10-28
status: active
absorbed_by: ""
asked_again: []
cited: []
---

# D-0016 golangci-lint v1.64.8 is installed with go install inside the golang:1.26 image, not run from the golangci-lint v1 image

Decision in S-0007 (wip/archive/agents/S-0007.md): "golangci-lint v1.64.8 is installed with `go install` inside the golang:1.26 image, not run from the golangci/golangci-lint v1 image: a prebuilt v1 binary is built with an older Go and refuses a go.mod that targets 1.26, and that image could not be pulled anyway (I-0003)."

Later stories that change the Go version or the lint step keep the install-in-image shape.
