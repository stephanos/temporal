---
satisfies: [R6, R7]
---
# fn-233-gomad-intercept-the-go-command-so-make.4 Temporal Makefile through the wrapper and operator documentation

## Description
Touches: tools/gomad/internal/backends/wasm/CLI.md, tools/gomad/README.md, gomad.toml at the Temporal module root, golden corpus (make fixture).
Run gomad -- make functional-test against the unchanged Makefile: go run of lintcode, protogen and test-runner produce host binaries, the functional packages run through run-guest, make exit status is propagated, and TEST_RACE_FLAG=1 is refused with the one-line message instead of being dropped. Document the wrapper, shim classification, refusal list, gomad.toml, interception limits (bare go on PATH only; absolute compiler paths are not intercepted), the evidence-origin rule and that intercepted evidence is not qualification evidence. Claim support only for tests the sweep and one-box specs have proven.

## Acceptance
gomad -- make functional-test exits with make status on the unchanged Makefile with host generators and guest tests confirmed in the fixture capture; race flag refusal visible; CLI.md and README document every item in R7; no support claim beyond proven tests; document diff check passes.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
