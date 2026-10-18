---
satisfies: [R19, R13, R14]
---
# fn-210-rust-prototype-kettle-on-a-trait-engine.8 Slice 1: lift Kettle to TLA+ and agree with TLC

## Description
New crates `umpire/lift` (tokens recorded by `#[machine]` to a tree for the R13 subset; TLA+ module and config emitter) and `umpire/backends/tlc` (pinned `tla2tools.jar` runner on the mise JDK, `-deadlock` on, `-dump` streamed, `-dumpTrace json` on violation). Agreement test: TLC's reachable set equals the checker's 283 states (decoded through the shared state spelling), invariant and postcondition verdicts agree, `boils` liveness agrees under weak fairness on internal steps. Goldens: `cases/kettle/expected/Kettle.tla`, `Kettle.cfg`, `tlc-agreement.txt`. Record the jar's version and checksum; add `umpire-lift`/`umpire-tlc` to the admitted list only with what they need.

**Files:** model2/umpire/lift/, model2/umpire/backends/tlc/, model2/Cargo.toml (members), model2/umpire/samples/cases/kettle/expected/, Makefile (TLC jar fetch into model2/build/).

**Quick commands:** `mise exec -- cargo test -p umpire-lift -p umpire-tlc -p samples`
## Acceptance
- `Kettle.tla` golden emitted; TLC run reports 283 distinct states and no violation.
- Agreement golden lists zero differences; a planted extra transition produces a listed difference.
- An out-of-subset expression refuses emission naming it (trybuild or insta golden).

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
