---
satisfies: [R6]
---
# fn-210-rust-prototype-kettle-on-a-trait-engine.19 Claims as check! statements; bare #[within(..)]

## Description
**Owner decision (2026-10-18):** invariants and postconditions are methods with no return type whose body makes `check!` statements, as Scala's `check`; a claim holds on a state or edge iff no check failed. `check!(e)` expands to `if !(e) { ::umpire_dsl::claim::fail(stringify!(e)) }` and records the failure (expression text) in a per-evaluation recorder that the generated predicate reads after calling the method; `unknown!("reason")` records a `Hole` the same way. No panics and no unwinding, since the wasm guest is `panic=abort`; a claim method stays an ordinary method callable from a unit test (the recorder is simply empty there). Counterexamples name the failing check: `fails_over_at_boiling: check!(self.beeping) failed at Model { .. }`. Also (owner 2026-10-19): there is no `#[within(..)]` attribute at all; a progress claim names its assumption in its expression, `.assuming(Self::never_switched_off())`, and `#[machine]` registers the `#[assumptions]` block's names for reports. `#[after(..)]` keeps its variant path.

```rust
#[invariants]
impl Model {
    fn beeps_only_when_off(&self) {
        if self.beeping {
            check!(!self.heating_on);
        }
    }
}

#[postconditions]
impl Model {
    fn warms_only_when_on(&self, before: &Self) {
        if !before.heating_on {
            check!(self.temperature <= before.temperature);
        }
    }
    #[after(Heater::Failover)]
    fn fails_over_at_boiling(&self) {
        check!(self.temperature == BOILING_TEMPERATURE);
        check!(self.beeping);
    }
}

#[progress]
impl Model {
    fn boils(&self) -> Progress {
        whenever(self.heating_on).eventually(self.beeping).assuming(Self::never_switched_off())
    }
}
```

Do: `check!`/`unknown!` and the recorder in `umpire-dsl` (`claim.rs`); the generated invariant and postcondition predicates in `dsl/macros/src/machine.rs` wrap each call with the recorder and map it to `Verdict`; refuse a claim method with a return type (`invariants and postconditions return nothing; use check!`) and any `#[within(..)]` attribute (`a progress claim names its assumption with .assuming(..)`), each pinned by a trybuild `.stderr` fixture; rewrite Kettle's claims and the `machine` macro fixture; the counterexample text gains the failing check (the only permitted golden change, in `report.txt`/`paths/` of the attribute Kettle and the checker's refusal/hole fixtures that use the attribute surface); `kettle_trait` and its goldens unchanged; DESIGN.md notes the shape and the recorder in one sentence each.

**Touches:** `model/umpire/dsl/**`, `model/umpire/checker/**`, `model/umpire/tests/fixtures/kettle/**`, `model/umpire/tests/fixtures/changed_state/**`, `model/DESIGN.md`

The shared `changed_state` attribute fixture introduced by .18 must adopt the new claim signatures too; its action/step semantics stay unchanged. Other manual trait fixtures remain unchanged. Test the recorder with crate-owned fixtures for failed checks, holes, exact-once evaluation, direct calls and isolation across evaluations.

**Feedback loop (R30):** paste `make model-time` into the done summary.
**Owner-approved assumption consistency:** within one exploration, each progress claim must return the same assumption across reached states. A mismatch makes that progress claim Unknown with a diagnostic. Do not silently use the first value while ignoring mismatches. The rule is per claim and per exploration; different world instances may supply different fixed assumptions.
**Owner-approved legacy compatibility:** retain a legacy `Scope` type alias so the separate trait Kettle and other manual trait sources remain unchanged. The attribute surface exposes `Assumption`, `#[assumptions]` and `.assuming(..)` only. Preserve the legacy trait builder calls needed by the unchanged reference; document this compatibility boundary.
## Acceptance
- Invariant and postcondition methods have no return type and use `check!` / `unknown!`; a claim holds iff no check failed; the recorder never panics or unwinds, and the wasm guest's behaviour is unchanged.
- A `#[within(..)]` attribute is refused naming `.assuming(..)`; a claim method with a return type is refused; both pinned by trybuild `.stderr` fixtures; the attribute surface replaces `Scope`/`#[scopes]` with `Assumption`/`#[assumptions]`; the owner-approved legacy `Scope` alias preserves unchanged manual trait sources.
- Kettle's claims and the `machine` macro fixture use the new form; the only golden change is the failing-check text in counterexamples; `kettle_trait` and its goldens are byte-identical.
- A claim method called directly in a unit test behaves as an ordinary method.
- DESIGN.md records the shape and the recorder; the full local gate passes within R30.
- Each progress claim's assumption stays fixed across reached states in one exploration; a mismatch yields Unknown with a diagnostic.
## Done summary
Implemented fn-210.19: unit check!/unknown! claims record explicit per-evaluation evidence, and ordinary progress bodies return one combined evaluation per real reached state. Assumptions are fixed per claim and exploration; mismatches yield Unknown. Legacy trait sources and public Verdict variants remain compatible.

Checkpoint: 1fc903d6eea7927212ec9e3250d48f496744bb0b
Base: 72eccde7374c82882419c0f38b224e0dd5ea0b1e
Workspace: /Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/fn21019-claims
Status: in_progress; conductor owns integration, review and completion.
Delegated implementation lanes: 4, all terminal; no child commands running. Coordinator alone ran Cargo/Make and committed.
Worker deferred implementation review to the conductor under the parallel-wave contract.

Baseline: inconclusive command-exit receipt: the initial focused baseline log contains completed passing suites, but its host session handle/exit status was not retained. It is not used as an authoritative green receipt. Subsequent focused runs, standalone workspace tests and final full gates have captured exit 0.
Intended red evidence: missing check!/unknown!/claim API; acceptance of forbidden return-valued claims; missing combined progress API; missing path.failure on recorded check failure.
Final verification: cargo test --workspace (exit 0), clippy --workspace --all-targets --all-features -- -D warnings (exit 0), make model-check (exit 0), and one make model-time after source checkpoint (exit 0).
make model-check included format, native Clippy/docs/tests, unchanged capability negative controls, protobuf freshness, wasm core build/Clippy, release wasm guest, and Go-host parity.
Tests cover check evaluation/control flow, first failure/hole precedence, direct/inactive calls, nested and panic restoration, thread isolation, correct state and second emitted choice, rendering without reevaluation, manual callback evidence isolation, fixed per-claim/per-exploration assumptions, differing worlds, mismatches, dynamic invalid actions and empty-world/manual validation.
New regression tests are modules of the existing fixture binaries, preserving the single fixture-binary layout.
Only behavioral golden additions are failed-check text in changed_state report/path; attribute Kettle goldens need no changes and kettle_trait source/goldens are byte-identical. Macro expansion and requested UI diagnostics were refreshed from actual output. Existing unknown-after and actor-type UI errors retain their intended diagnoses.
No Flow metadata, MILESTONES, archive, manifests, locks, ABI, schema, host source or generated protobuf changes.

Logs copied into root .flow/tmp/: fn21019-baseline.log, fn21019-recorder-red.log, fn21019-macro-red.log, fn21019-checker-red.log, fn21019-evidence-red.log, fn21019-focused.log, fn21019-shared.log, fn21019-workspace.log, fn21019-clippy.log, fn21019-model-check.log, fn21019-model-time.log.

make model-time (one actual warm run after checkpoint):
```text
step (warm)                         time
gate command: UMPIRE_TIME_STEPS=1 make -C .. model-check
timing-controls                   0.016s
capability-controls               0.283s
protobuf-freshness                0.211s
fmt                               0.133s
clippy                            0.130s
docs                              0.122s
tests                             0.778s
wasm32-wasip1                     0.053s
core-wasm-clippy                  0.090s
wasm-guest                        0.104s
go-host                           1.810s
inner source: umpire/checker/src/lib.rs
inner command: mise exec -- cargo test -p umpire-checker --quiet
inner loop (edit+test)            1.719s
gate total                        3.928s  (budget 30s)
inner loop                        1.719s  (budget 3s)
```

Conductor integration: ordinary merge eed24ff6f36b2db4358a6ce986120ce34e0f9a0c. Actual integrated make model-check exited 0 with the approved payload-only cleanup present. Final single integrated make model-time exited 0: full gate 4.005s (30s budget), inner edit/test loop 2.305s (3s budget); full output .flow/tmp/fn210-19-integrated-model-time.log. No timing retry.

stage: impl-review - ran [three concurrent correctness/contracts/integration draws, each SHIP, zero findings]. Actual reviewers gpt-6.1-sol at high, same GPT family as writer. RID a8c22ada47b34e7789ee72ba13596850; receipt /tmp/impl-review-receipt-8f37faba39e2-fn-210-rust-prototype-kettle-on-a-trait-engine.19.json, SHA256 84fdab6b0eca9a05243879a7cd4e3b36b099dd03b12654b80850e119371c670e. Committed review artifact excluded all dirty owner/cleanup paths (empty path intersection). Fresh reviewer Cargo commands were blocked by read-only build locks and are inconclusive; conductor's actual integrated gate is authoritative.
stage: plan-sync - skipped(policy: planSync.enabled=false)
Tracker sync: n/a (bridge inactive).

Post-integration generic-world readiness diagnostic rebuilt exact current library artifacts offline, compiled and ran successfully, and matched three literal expected outputs; logs .flow/tmp/fn210-parameter-readiness-post19*. This is additional engine evidence for .7, not attribute-surface support for deferred world/input/generic features.
## Evidence
- Commits: 1fc903d6eea7927212ec9e3250d48f496744bb0b, eed24ff6f36b2db4358a6ce986120ce34e0f9a0c
- Tests: cd model && CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target TRYBUILD=overwrite INSTA_UPDATE=always mise exec -- cargo test -p umpire-dsl -p umpire-dsl-macros -p umpire-checker, cd model && CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target INSTA_UPDATE=always mise exec -- cargo test -p umpire-tests --test fixtures, cd model && CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target mise exec -- cargo test --workspace, cd model && CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target mise exec -- cargo clippy --workspace --all-targets --all-features -- -D warnings, CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target make model-check, CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target make model-time, Integrated make model-check exited 0 including payload-only cleanup and stock wasm/Go parity, Final single integrated make model-time exited 0: full4.005s inner2.305s; .flow/tmp/fn210-19-integrated-model-time.log, Three gpt-6.1-sol high draws SHIP, zero findings; receipt SHA25684fdab6b0eca9a05243879a7cd4e3b36b099dd03b12654b80850e119371c670e, Post19 generic-world rustc probe and literal expected-output diff exited 0
- PRs: