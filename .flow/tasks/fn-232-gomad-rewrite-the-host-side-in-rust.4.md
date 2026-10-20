---
satisfies: [R13]
---
# fn-232-gomad-rewrite-the-host-side-in-rust.4 Behavioral differential harness and corpus

Touches: tools/gomad/scripts/behavioral-parity, tools/gomad/testdata/parity/{targets,seeds,plans}/, CI job

## Description
Build the harness that runs the Go binary and the Rust binary on fresh executions of a fixed corpus (targets, seeds, profiles, scenarios, exploration plans) and compares per verb: outcome class and termination, failure grouping (partition equality), replay match/divergence ordinal and reason on each binary's own artifacts, accepted minimization reductions by decision identity, qualification verdict per cell. Report the first semantic difference with a path. Fail on a missing corpus entry. Wire into CI as a non-blocking job until R12.

## Acceptance
The harness and the golden corpus share one corpus directory; the harness runs end to end with inspect only (later verbs are added by their tasks); a deliberately perturbed Rust output is reported with its path; a missing corpus entry fails the run; the corpus regenerates under an environment flag with a byte-identical second pass.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
