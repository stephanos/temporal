# Root pre-review source observations

These are conductor checks during task-23 implementation, not an independent
review verdict, a formal SHIP receipt, or native qualification.
HEAD remains `b43aeb5b15d438eebab65f2ce48eecb19e76e55c`.

## Nested default target

Placing the new nested `lint-code` target before `generate` changed Make's
default action. A database query with an explicit nonexistent target reported
`.DEFAULT_GOAL := lint-code`; the same query against the HEAD Makefile reported
`generate`. The worker retained a behavioral RED in `make-default-red.log` and
moved the new target below `generate`. Its first frozen contract suite passed,
including the new default-target regression.

## Tagged root integration package

The first actual frozen fast-lint run returned Make status 2, with golangci
loading status 7. It selected 109 root packages, then failed because
`tools/gomad3integration` has only `integration_test.go`, requiring build tag
`gomad3_integration`. Root Make's existing `gomad3-integration-test` target
already supplies `test_dep,gomad3_integration`.

Counting `IgnoredGoFiles` as inventoried source does not make this package
loadable with the ordinary lint tags. This is a remaining routing defect, not
a lint-rule finding or permission to omit the integration package. The bounded
correction keeps this root-owned package in a scope with its existing tag and
preserves the comparison revision and ordinary packages' existing tags/rules.
The worker is adding a tagged-only real Git/Go-list/Make regression and a new
corrective evidence set; the original frozen receipts remain immutable.

First-freeze identities:

- `final-root-fast.log`: SHA-256
  `29673c499537983188c87cc6a7583800fcf32f0a614acc0884e3013f2e9ad68f`.
- `final-root-fast.receipt.json`: SHA-256
  `691424bfac8d5ec558f99d49cf7b00e4233e9914e60d73278670a527c90bcadd`.
- `source-frozen.sha256`: SHA-256
  `d7c235be3de43405908509e7ba232bdc5c6a8771b830035f2c5253e10909fb4c`.

The first script ended with status 1. Its helper lint, helper vet, contracts,
Make ownership and generated validation commands each returned 0; the actual
Gomad lint gate returned 2 with 1,300 current-rule findings, and mixedbrain's
configured lint/vet gate returned 0. The retained source-after log checked all
15 declared source inputs; tools-after checked all three binaries. Those checks
bind that first freeze only, not later corrected helper source.

Current Gomad lint failures remain visible and separately require ownership.
The root-tag correction cannot close those findings, native qualification,
aggregate first-baseline preservation, or the full milestone goal.
