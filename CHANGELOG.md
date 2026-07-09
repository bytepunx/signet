# Changelog

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
