---
satisfies: [R19, R13, R14, R26, R28]
---
# fn-210-rust-prototype-kettle-on-a-trait-engine.8 Slice 1: lift Kettle to TLA+ and agree with TLC

## Description
**Feedback loop (R30):** keep the inner loop at 3 s or less and `make model2-check` at 30 s or less, warm; paste `make model2-time` output into the done summary; a breach is fixed here or excepted by the owner. TLC runs on the Kettle fixture set in the default gate only if the gate stays within budget; otherwise one batched TLC run, or behind `UMPIRE_BACKENDS=require`.

**Placement (owner 2026-10-18):** `umpire-tlc` lives at `model2/umpire/backends/tlc/`; `umpire-lift` at `model2/umpire/lift/`.

**Consumer (R29):** register a TLC consumer owning `<Model>.tla`, `<Model>.cfg` and `tlc.txt`; add those files to Kettle and the shared fixtures TLC can express.

**Conventions 2026-10-18 (R25, R26, R28):** TLA identifiers are the author's Rust names; the TLC-dump decoder parses values into states and re-spells them through `Spell` for the agreement; the lifted tree stays an internal enum (debug dump only, never a golden).

**Layout (R22):** lift cases go in `umpire/lift/tests/fixtures/<name>/`; Kettle's `Kettle.tla`, `Kettle.cfg` and `tlc.txt` go in `umpire/tests/fixtures/kettle/expected/`.

New crates `umpire/lift` (tokens recorded by `#[machine]` to a tree for the R13 subset; TLA+ module and config emitter) and `umpire/backends/tlc` (pinned `tla2tools.jar` runner on the mise JDK, `-deadlock` on, `-dump` streamed, `-dumpTrace json` on violation). Agreement test: TLC's reachable set equals the checker's 283 states (decoded through the shared state spelling), invariant and postcondition verdicts agree, `boils` liveness agrees under weak fairness on internal steps. Goldens: `tests/fixtures/kettle/expected/Kettle.tla`, `Kettle.cfg`, `tlc-agreement.txt`. Record the jar's version and checksum; add `umpire-lift`/`umpire-tlc` to the admitted list only with what they need.

**Files:** model2/umpire/lift/, model2/umpire/backends/tlc/, model2/Cargo.toml (members), model2/umpire/tests/fixtures/kettle/expected/, Makefile (TLC jar fetch into model2/build/).

**Quick commands:** `mise exec -- cargo test -p umpire-lift -p umpire-tlc -p samples`
## Acceptance
- `Kettle.tla` golden emitted; TLC run reports 283 distinct states and no violation.
- Agreement golden lists zero differences; a planted extra transition produces a listed difference.
- An out-of-subset expression refuses emission naming it (trybuild or insta golden).

## Done summary
Moved to fn-212 (TLA+ lifting and the TLC backend), which first decides whether the second opinion is worth the machinery (fn-210 R36).

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: