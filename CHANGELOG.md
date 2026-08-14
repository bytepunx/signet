# Changelog

## [0.10.2](https://github.com/bytepunx/signet/compare/v0.10.1...v0.10.2) (2026-08-14)


### Bug Fixes

* **store:** actually create the audit_log retention TTL ([#51](https://github.com/bytepunx/signet/issues/51)) ([8f635bb](https://github.com/bytepunx/signet/commit/8f635bb29b42c04effae10d046481217d2523a74))

## [0.10.1](https://github.com/bytepunx/signet/compare/v0.10.0...v0.10.1) (2026-08-14)


### Bug Fixes

* **gitops:** remove delete-by-absence, add explicit Delete RPCs ([#46](https://github.com/bytepunx/signet/issues/46)) ([4061d29](https://github.com/bytepunx/signet/commit/4061d2955db5c71807ba1fd9ac58ce096075ac96))
* **gitops:** stop git sync from silently reverting PatchServiceConfig writes ([#49](https://github.com/bytepunx/signet/issues/49)) ([f9ead75](https://github.com/bytepunx/signet/commit/f9ead75d7d5c7d356ca9790a043868b15d4694ec))

## [0.10.0](https://github.com/bytepunx/signet/compare/v0.9.0...v0.10.0) (2026-08-13)


### Features

* **gitops:** add PatchServiceConfig for atomic config patching ([5b8be2d](https://github.com/bytepunx/signet/commit/5b8be2d81b792bb3c1e337c949d143a6204f00f1))


### Bug Fixes

* **chart:** require admin.tls when admin.clusterAccess is enabled ([#42](https://github.com/bytepunx/signet/issues/42)) ([3fc5658](https://github.com/bytepunx/signet/commit/3fc56588d274b4dbfc09a426dc009149bd7a686b))
* **chart:** restrict admin-tls Secret volume to 0400 ([b81fab1](https://github.com/bytepunx/signet/commit/b81fab1464950e90139c6a92ed8d0f66032767e6))

## [0.9.0](https://github.com/bytepunx/signet/compare/v0.8.0...v0.9.0) (2026-08-12)


### Features

* **audit:** record every GitOps write in the audit log ([#36](https://github.com/bytepunx/signet/issues/36)) ([92edac0](https://github.com/bytepunx/signet/commit/92edac0444765144958cfca2e807e43241bae203))

## [0.8.0](https://github.com/bytepunx/signet/compare/v0.7.0...v0.8.0) (2026-08-12)


### Features

* **gitops:** let a workload push GitOps secrets scoped to its own identity ([#32](https://github.com/bytepunx/signet/issues/32)) ([1effb69](https://github.com/bytepunx/signet/commit/1effb69f71e4240bc60ff3a846c3099c0d500f3b))
* **server:** terminate real TLS on the admin listener ([#31](https://github.com/bytepunx/signet/issues/31)) ([478aa60](https://github.com/bytepunx/signet/commit/478aa60b7fd3ee55118a3f0bd9812dcefdec5a16))


### Bug Fixes

* **gitops:** surface files skipped for path-depth mismatch during sync ([#30](https://github.com/bytepunx/signet/issues/30)) ([f6e6a23](https://github.com/bytepunx/signet/commit/f6e6a234de33728586910b4f0e994589d492357c))
* **proto:** pin remote codegen plugin versions; auto-fix drift in CI ([#34](https://github.com/bytepunx/signet/issues/34)) ([96032c0](https://github.com/bytepunx/signet/commit/96032c03a518eca3ce3d35cbb40c6bd077412c3d))

## [0.7.0](https://github.com/bytepunx/signet/compare/v0.6.1...v0.7.0) (2026-08-08)


### Features

* **chart:** add admin.clusterAccess flag for in-cluster admin gRPC access ([#20](https://github.com/bytepunx/signet/issues/20)) ([0f43c8b](https://github.com/bytepunx/signet/commit/0f43c8bf9fd3ba2db5d6b9c9d21df7a1446f508e)), closes [#19](https://github.com/bytepunx/signet/issues/19)

## [0.6.1](https://github.com/bytepunx/signet/compare/v0.6.0...v0.6.1) (2026-07-21)


### Bug Fixes

* **store:** restart-lock acquisition was broken against real CockroachDB ([#17](https://github.com/bytepunx/signet/issues/17)) ([6216d59](https://github.com/bytepunx/signet/commit/6216d59365809f95f8cb126482a649e58d8525a4))

## [0.6.0](https://github.com/bytepunx/signet/compare/v0.5.1...v0.6.0) (2026-07-19)


### Features

* **gitops:** detect and delete secrets/configs removed from a repo ([#15](https://github.com/bytepunx/signet/issues/15)) ([a36725a](https://github.com/bytepunx/signet/commit/a36725a7fd18e895bc9e854b3a9c80138403b151))

## [0.5.1](https://github.com/bytepunx/signet/compare/v0.5.0...v0.5.1) (2026-07-19)


### Bug Fixes

* **auth:** fix SPIFFE ID extraction from real go-spiffe mTLS connections ([#13](https://github.com/bytepunx/signet/issues/13)) ([d2e8431](https://github.com/bytepunx/signet/commit/d2e843149d8743bb319dd3c3fb0e7f28591afa1c))
* **gitops:** make repo sync over SSH actually work ([#12](https://github.com/bytepunx/signet/issues/12)) ([ac5ff60](https://github.com/bytepunx/signet/commit/ac5ff60a668adce8b49c4742a272b71d96ce4945))

## [0.5.0](https://github.com/bytepunx/signet/compare/v0.4.0...v0.5.0) (2026-07-18)


### Features

* add policy management: CreatePolicy/ListPolicies/DeletePolicy ([#9](https://github.com/bytepunx/signet/issues/9)) ([28195b8](https://github.com/bytepunx/signet/commit/28195b859545fea1943c84761bb7baaf58d3aff8))

## [0.4.0](https://github.com/bytepunx/signet/compare/v0.3.0...v0.4.0) (2026-07-12)


### Features

* **cli:** add signet secret set/rm for goal-oriented secret authoring ([012df1f](https://github.com/bytepunx/signet/commit/012df1f8e2a87a0449f09331d870bdd018fba380))
* **proto:** externalize schema to bytepunx/signet-proto for independent client versioning ([280efa4](https://github.com/bytepunx/signet/commit/280efa42c064392ba500b128f92fcc85bb063613))


### Bug Fixes

* **gitops:** decrypt SOPS data key directly instead of via local keyservice ([daf864e](https://github.com/bytepunx/signet/commit/daf864e1b7a8a29442734f7af251f5c4b6fd264e))

## [0.3.0](https://github.com/bytepunx/signet/compare/v0.2.3...v0.3.0) (2026-07-09)


### Features

* **cli:** add reusable destructive-operation confirmation prompt ([166e49c](https://github.com/bytepunx/signet/commit/166e49ccf4cc0aefa9f6ea1190da0dacb6528612))
* **crypto,api:** add KEK tier, AAD-bound encryption, and key-check verification ([b4397d1](https://github.com/bytepunx/signet/commit/b4397d1a2b61985294b75e314dab1c9f9dcc5b39))


### Bug Fixes

* **api:** tighten bundle path-traversal check (L-6) ([e0d847b](https://github.com/bytepunx/signet/commit/e0d847b8440a0924209494315103e094df0d772d))
* **auth:** authorize admin API calls, not just authenticate them (C-1) ([ba336de](https://github.com/bytepunx/signet/commit/ba336de8953a82ea1341168d421ba70f75e4f9d6))
* **auth:** validate SPIFFE trust domain and match three-segment policies ([5c103fa](https://github.com/bytepunx/signet/commit/5c103fa653735b14caaa799c6d36df53fdd3bc6e))
* **cli:** require TLS for non-loopback admin connections (H-6) ([3b3f51e](https://github.com/bytepunx/signet/commit/3b3f51e3e288bd9a94a5b2066f1ab1bd6edce17f))
* **helm:** add adminSubjects and auditFailClosed to values.schema.json ([300594b](https://github.com/bytepunx/signet/commit/300594b5b82fd8d685757295fdca5af44cbdc83b))
* **server:** recover from panics in streaming admin RPCs (M-5) ([4d9652e](https://github.com/bytepunx/signet/commit/4d9652e66519f2aa7cce726c957577a5e24d506e))
* **signetd:** harden config validation and add admin/audit knobs ([dc34dc5](https://github.com/bytepunx/signet/commit/dc34dc55784a9d5cbed7dc2374faaa0c6845de81))
* **unseal:** make GF(2^8) multiplication branchless (L-3) ([38335a5](https://github.com/bytepunx/signet/commit/38335a55335bd1c59ff111f0b986ff0354072f72))

## [0.2.3](https://github.com/bytepunx/signet/compare/v0.2.2...v0.2.3) (2026-07-05)


### Bug Fixes

* prefix docker image semver tags with v to match helm chart appVersion ([0f57554](https://github.com/bytepunx/signet/commit/0f575546113a7e94c43c8fb728252d31b8855d46))

## [0.2.2](https://github.com/bytepunx/signet/compare/v0.2.1...v0.2.2) (2026-07-01)


### Bug Fixes

* resolve all golangci-lint findings ([296d977](https://github.com/bytepunx/signet/commit/296d97761048b9dc12d9f182d1f7c202bab6619b))

## [0.2.1](https://github.com/bytepunx/signet/compare/v0.2.0...v0.2.1) (2026-07-01)


### Bug Fixes

* commit generated proto stubs instead of generating them in CI ([940eaca](https://github.com/bytepunx/signet/commit/940eaca9b8eb1be8cacab4453e9e8b883092a5a6))

## [0.2.0](https://github.com/bytepunx/signet/compare/v0.1.0...v0.2.0) (2026-06-28)


### Features

* initial checkin ([f959f24](https://github.com/bytepunx/signet/commit/f959f24eed754fa0c664ca3881859de467c776e9))


### Bug Fixes

* commit generated proto stubs and align CI with Makefile ([256dad7](https://github.com/bytepunx/signet/commit/256dad724e6d8233f92c053c472fd9939a9b4b6e))
* remove version field from golangci-lint config ([a872d21](https://github.com/bytepunx/signet/commit/a872d217daf9f3aef3c5c512d65e6f640b8b9a4d))
* run golangci-lint through make lint so proto stubs are generated first ([271369f](https://github.com/bytepunx/signet/commit/271369fdfb1d341b4aa9ef2f18f64714d7cba7a8))
