---
satisfies: [R13, R14]
---
# fn-232-gomad-rewrite-the-host-side-in-rust.4 Behavioral differential harness and corpus

Touches: tools/gomad/scripts/behavioral-parity, tools/gomad/testdata/parity/{targets,seeds,plans}/, CI job

## Description
Frozen Go baseline owner/input index: /home/agent/wt/evidence/fn232/go-baseline-1d7b37f00a/README.md. Preserve its independently copied launcher/helper, source archive and receipts/tool pins for fresh equivalence; it supplies no qualification claim.
Build the harness that runs the Go binary and the Rust binary on fresh executions of a fixed corpus (targets, seeds, profiles, scenarios, exploration plans) and compares per verb: outcome class and termination, failure grouping (partition equality), replay match/divergence ordinal and reason on each binary's own artifacts, accepted minimization reductions by decision identity, qualification verdict per cell. Report the first semantic difference with a path. Fail on a missing corpus entry. Provide CI job source wiring until R12, without dispatching CI. The required local comparison gate is blocking. Preserve/inventory the frozen Go baseline under fn-232 ownership; every actual verb task adds its own fresh execution/parity and golden lane.

## Acceptance
The harness and the golden corpus share one corpus directory; the foundation comparator is exercised on fixed semantic examples and intentionally different outputs without requiring the not-yet-implemented inspect verb; task .5 owns the first real inspect lane; a deliberately perturbed Rust output is reported with its path; a missing corpus entry fails the run; the corpus regenerates under an environment flag with a byte-identical second pass.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
