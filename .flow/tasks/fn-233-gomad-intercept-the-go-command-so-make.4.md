---
satisfies: [R6, R7]
---
# fn-233-gomad-intercept-the-go-command-so-make.4 Temporal Makefile through the wrapper and operator documentation

## Description
**Touches:** tools/gomad/docs/wasm/CLI.md, tools/gomad/docs/README.md, gomad.toml at the Temporal module root, tools/gomad/testdata/parity/** (make fixture).
Run gomad -- make functional-test TEST_RACE_FLAG=off PERSISTENCE_TYPE=sql PERSISTENCE_DRIVER=sqlite against the unchanged Makefile: verify generator/linter go run host passthrough in separate invocations (functional-test itself directly calls go test); the functional packages run through run-guest, make exit status is propagated, and TEST_RACE_FLAG=on is refused with the one-line message instead of being dropped. Document the wrapper, shim classification, refusal list, gomad.toml, interception limits (bare go on PATH only; absolute compiler paths are not intercepted), the evidence-origin rule and that intercepted evidence is not qualification evidence. Claim support only for tests the sweep and one-box specs have proven.

## Acceptance
Use a supported registered test selection for the positive smoke case and retain the exact Make variable/selector expansion. A broad-suite run proves routing and truthful classifications, not support for every test. The default invocation without TEST_RACE_FLAG=off must refuse -race before build.

gomad -- make functional-test TEST_RACE_FLAG=off PERSISTENCE_TYPE=sql PERSISTENCE_DRIVER=sqlite exits with make status on the unchanged Makefile with host generators and guest tests confirmed in the fixture capture; race flag refusal visible; CLI.md and README document every item in R7; no support claim beyond proven tests; document diff check passes.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
