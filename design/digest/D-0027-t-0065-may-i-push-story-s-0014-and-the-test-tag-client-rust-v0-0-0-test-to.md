---
id: D-0027
title: "T-0065: may I push story/S-0014 and the test tag client-rust/v0.0.0-test to origin: no"
kind: answer
sources: [TH-0023]
topics: [release, github, ci, client-rust, rust, code, all]
paths: [".github/workflows/release-client-rust.yml", client-rust/Cargo.toml, client-rust/README.md]
date: 2026-11-06
status: active
absorbed_by: ""
asked_again: []
cited: [S-0011]
---

# D-0027 T-0065: may I push story/S-0014 and the test tag client-rust/v0.0.0-test to origin: no

The operator, alex, answered TH-0023 at 2026-11-06T08:09:29Z.

What was asked:

```text
T-0061, T-0062 and T-0063 are done and committed on story/S-0014 (workflow, Cargo.toml, README); acceptance criteria 1-3 are ticked from reading the files (the workflow was YAML-parsed only: actionlint is not installed here; cargo metadata parses Cargo.toml). T-0065 needs a real run, which means pushing the branch and an annotated tag client-rust/v0.0.0-test to origin, which starts Actions and creates a pre-release GitHub Release, then deleting both. The version has a pre-release suffix so the run only does cargo publish --dry-run and never touches crates.io. Recommended: yes, go ahead; I…
```

The answer:

```text
no
```

Source: TH-0023.
