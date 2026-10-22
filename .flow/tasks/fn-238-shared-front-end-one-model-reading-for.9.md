---
satisfies: [R2]
---
# fn-238-shared-front-end-one-model-reading-for.9 Derive the IR token printer instead of hand-writing it

## Description
Owner approved 2026-10-22 ("yes"). `dsl/macros/src/lifted.rs` hand-writes the printer that turns the admitted `umpire-ir` tree back into Rust tokens for the embedded Model (277 lines: one `record!` line per struct, explicit arms per enum variant). Every new IR node or field needs a matching printer edit, about 5-15 lines per construct, and body-level and Quint-parity constructs add many nodes.

Add a small proc-macro crate (about 100-150 lines, e.g. `umpire-ir-derive`) with a `#[derive]` that emits the token-printing impl for structs and enums in `umpire-ir`, and delete the hand-written printer. fn-238.4's reviewer estimated about -190 net lines. No third-party dependency beyond what the workspace already uses (`syn`, `quote`, `proc-macro2`). R30's rule that proc-macro crates stay small applies.

Schedule after fn-227 closes or before the next batch of new IR constructs.

### Verification
All goldens byte-identical, including expansion snapshots, and the fn-238 IR dump unchanged. The new crate enters `umpire-dsl`'s lockfile closure, so refresh the guest once in a separate commit and run slice and gaps. Quick then Full gates; report line deltas.


**Size:** M
**Touches:** `model/Cargo.toml`, `model/Cargo.lock`, `model/umpire/ir/**`, `model/umpire/ir-derive/**`, `model/umpire/dsl/macros/**`, `model/DESIGN.md`, `model/temporal/hosts/go/umpire_temporal_guest.wasm` (generated refresh only, separate commit).

### Execution and gates

Run before fn-227.16 introduces new IR constructs, as allowed by the scheduling rule above. Retain the deferred status of fn-238.5 and .6. Reuse the current IR, front-end and DSL module boundaries. The derive must cover structs and enums and preserve the exact printed constructors; handwritten generic container/scalar support may remain shared. Do not update expansion, semantic or IR goldens to accept output changes.

Quick baseline and post-edit checks: `cd model && mise exec -- cargo test -p umpire-ir -p umpire-dsl-macros --lib`; run focused derive/added-field controls and IR/expansion comparisons appropriate to the implementation. Full gate: `UMPIRE_BACKENDS=require make model-check`, once Quick passes and the separate guest-refresh commit is present. Refresh the embedded guest once through `make umpire-guest` in a separate commit, then `make umpire-check-model-slice` and `make umpire-check-model-gaps`. Record source/test line deltas and full/inner-loop timing without hiding soft-target overruns. Required TLC verifies that moving constructor printing preserves backend behavior.
## Acceptance
- [ ] IR token printing is derived; the hand-written printer is gone; adding an IR field needs no printer edit (control).
- [ ] All goldens and the IR dump unchanged; guest refreshed once with slice/gaps green.
- [ ] Line deltas reported (target about -190 net).

## Done summary
IR authors can add fields and enum variants without updating a handwritten constructor printer. `umpire-ir-derive::Print` now derives every admitted IR constructor; shared scalar/container printing lives in `umpire-ir::print`.

The added-field/variant control passed after an initial missing-feature compile failure. Baseline Quick passed before edits but exercised zero library tests, so it established compilation only. Post-edit Quick exercises the new control; expansion fixtures and 26 shared semantic tests passed. All 782 tracked golden/snapshot/trybuild files remain byte-identical. Old and derived constructor tokens match for 26 Models, and reconstruction with the pre-task IR definitions produces the exact same 4,425,591-byte Debug dump (19 admitted Models, seven refusals).

Guest refresh ran once through `make umpire-guest` and is isolated in commit `a0323d4d6e`; the guest shrank 16 bytes. Both actual slice and gap Make targets passed, including their required-test summary checks. The gaps suite took 433.268 s. `UMPIRE_BACKENDS=require make model-time` passed every nested full-model gate stage with required TLC. Full gate 532.677 s exceeds the 60 s soft target; inner loop 9.272 s meets 10 s; outer wall 542.347 s. Attempt 1 is retained as inconclusive (exit 124 at 600.154 s during cold guest compilation after workspace tests). Trybuild closures and guest/host fixtures were recompiled; these measurements do not establish steady-state cost. No source changes followed the timeout.

Physical Rust line deltas are production -60, tests +42, combined -18. This misses the roughly -190 estimate because shared scalar/container implementations remain and the derive plus constructor-path annotations offset deleted registrations and enum arms. Workspace manifests/lockfile add 42 lines; DESIGN adds 16 net lines, for +40 total Model text lines excluding the generated binary and Flow evidence. Detailed counts and verification/review evidence are under `.flow/artifacts/fn-238-shared-front-end-one-model-reading-for/task9-*`.

All three fresh Codex review draws returned SHIP with no findings; the deterministic finalizer recorded SHIP. Their attempted Cargo reruns hit read-only build locks; cached control binaries and independent golden/IR hashes were checked, and the writer's actual full gate is retained. Reviewer and writer use the same model family as disclosed. The review receipt is `.flow/artifacts/fn-238-shared-front-end-one-model-reading-for/task9-review.json`.

GATE_SKIPPED:unittest:green-receipt 11436f2f - post-review verification reused the completed required-TLC gate

fn-238.5 and .6 remain deferred; this task does not close their parent spec. No feature entry route changed.

stage: impl-review - ran [2026-10-22T20:11:46Z..2026-10-22T20:14:57Z] (codex:gpt-6.1-sol:high; SHIP)
Tier: session (jev-unavailable(no_key)); repository implementer override gpt-6.1-sol at medium.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 080fcfbae1210f3fcab84482195be671c3aadfa7, a0323d4d6e3e258515ac3ad9b2062e436dabb9b8, 11436f2f2b29aebb5ee5af2e380cb9145fc4ef5a, 4348bade0233279ac35450400537fb5028eb832a
- Tests: cd model && mise exec -- cargo test -p umpire-ir -p umpire-dsl-macros --lib, cd model && mise exec -- cargo test -p umpire-dsl-macros --test fixtures fixtures, cd model && mise exec -- cargo test -p umpire-tests --test fixtures, cd model && mise exec -- cargo clippy -p umpire-ir -p umpire-ir-derive -p umpire-dsl-macros --all-targets -- -D warnings, mise exec -- cargo run --manifest-path .flow/tmp/fn-238.9/token-probe/Cargo.toml, mise exec -- cargo run --manifest-path .flow/tmp/fn-238.9/baseline-ir/Cargo.toml, make umpire-guest, make umpire-check-model-slice, make umpire-check-model-gaps, GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=f11cecbee9 make lint-code-fast, INCONCLUSIVE: UMPIRE_BACKENDS=require make model-time attempt1, exit124 after600.154s during cold wasm-guest compilation, UMPIRE_BACKENDS=require make model-time, cd model && mise exec -- cargo fmt --all --check, GATE_SKIPPED:unittest:green-receipt 11436f2f - post-review verification reused the completed required-TLC gate
- PRs: