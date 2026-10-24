---
satisfies: [R1, R2]
---
# fn-237-activity-timeouts-heartbeat-and.1 Optional handler facts through native adapters and independent TLC replay

## Description
Moved from fn-227.9 on 2026-10-21.

Deliver the bounded fact-result interface in the tracked deadline-fact contract, independently of activity policy. This is facts checkpoint F, allocated as fn-237-activity-timeouts-heartbeat-and.1 (formerly fn-227-parameterized-models-declared-worlds.9). fn-227 approved its three amendments in `bbf0835fbf`; earlier contract notes calling those amendments pending are superseded. Source anchors predate the completed native work and must be reanchored at execution.

**Size:** M. One return/branch-result adaptation across five existing production owners; existing enum/sequence decoding and Debug replay already carry facts. Compile, source, shared-fixture and replay controls close that same contract. No collector, TLS recorder, fact catalog or general result framework.
**Dependencies:** After fn-227.7 (cross-spec prerequisite, satisfied through this spec's dependency on fn-227). Its typed selected-World/source binding is exercised by F's declared-World fixture. No dependency on fn-227.3 or O (fn-237.2); serial scheduling handles overlapping source ownership.
**Files:** `model/umpire/dsl/macros/src/machine.rs`; `model/umpire/lift/src/{shape,tree,emit}.rs`; `model/umpire/lift/src/body/effects.rs`; new macro controls under `model/umpire/dsl/macros/tests/{pass,ui,fixtures}/`; `model/umpire/lift/tests/source.rs` and new owned source controls; new shared fixture `model/umpire/tests/fixtures/optional_facts/{model.rs,tests.rs,expected/**}`; replay tests in `model/umpire/testing/src/backend.rs`; `model/README.md`.
**Touches:** [model/umpire/dsl/macros/**, model/umpire/lift/**, model/umpire/tests/fixtures/optional_facts/**, model/umpire/testing/src/backend.rs, model/README.md, model/temporal/hosts/go/umpire_temporal_guest.wasm]

### Approach

1. Pin existing unit-handler expansions and no-fact source artifacts. Admit exactly Option<F> for one local nongeneric nonempty fieldless Debug enum. Select the machine Fact and opaque return constraint from that carrier; reject inconsistent/unsupported carriers using existing located diagnostics.
2. Capture the authored optional result inside the existing unit outcome/stay evaluation closure and construct zero-or-one facts only when the step is emitted. Keep state-change/reject/stay enabledness and reject-after-mutation rules. Unit effect helpers remain unit; no facts enter the outcome TLS frame. Direct reject! in an optional handler retains its Rust return-type failure.
3. Independently admit only None, Some(F::Variant), or exhaustive tail conditional trees over those leaves after supported effects. Lower branch facts in the same SSA environment as state updates. Reuse existing conditional/sequence expressions and Seq<Enum>; preserve exact old Seq<Unit> metadata and no-fact emission. Ordinary edges assign an empty sequence every time.
4. Add generic A/B/C/D fixture traces and false claims that force replay across all four values. Preserve authored first-matching guards. Test source-only fact substitution with native rows held constant. Reuse current typed decode and native Debug bridge; native rows supply no source branch meaning.

### Investigation targets

Required, reanchor before coding:

- `.flow/artifacts/fn-227-parameterized-models-declared-worlds/deadline-fact-contract.md` for the complete bounded grammar and current production/test owners.
- `model/umpire/dsl/macros/src/machine.rs` for handler return admission and all reject/stay adapter combinations.
- `model/umpire/lift/src/body/effects.rs` for source conditional SSA and helper boundaries.
- `model/umpire/lift/src/emit.rs` for real-edge fact assignment and observer-only preservation.
- `model/umpire/testing/src/backend.rs` for exact fact replay and observer controls.

### Verification

Quick from `model/`: `mise exec -- cargo test -p umpire-dsl-macros -p umpire-lift -p umpire-testing -p umpire-tests`; repeat admitted shared-fixture tests with `UMPIRE_BACKENDS=require`. Run scoped formatting and strict clippy. First capture the pre-change optional-result/source refusals, then positive and mutation-negative results.

Full from repository root: `make model-check`, `UMPIRE_BACKENDS=require make model-check`, `make model-time`. If linked guest bytes change, run `make umpire-guest`, then the embedded slice/gap checks. Compare all pre-existing snapshots, including metadata, to the pinned original manifest; no activity or other old golden updates. Retain actual counts, replay traces and soft-budget misses.

### Ordering note (2026-10-24)

Starts after fn-244.1, the serialized layout freeze, so it works in the new crate paths (for example `author/dsl/macros` and `backends/tla`).

### Paths after fn-244.1 (2026-10-24)

`model/umpire/dsl/macros` is `model/umpire/author/dsl/macros`; `model/umpire/lift` is `model/umpire/backends/tla` (crate `umpire-tla`); `model/umpire/tests/fixtures/<name>` is `model/umpire/dev/fixtures/fixtures/<name>` (crate `umpire-fixtures`, formerly `umpire-tests`); `model/umpire/testing` is `model/umpire/dev/testing`; `model/umpire/backends/tlc` is unchanged. Quick command crates: `-p umpire-dsl-macros -p umpire-tla -p umpire-tlc -p umpire-testing -p umpire-fixtures` as applicable.

### Execution reanchor and resolved implementation choices (2026-10-24)

fn-243.4's closure is committed in umpire before admission. The optional-fact contract remains independent of Activity policy and stored optional values. Required current owners include front declaration/SSA, IR, author macro generation, the typed Interprets bridge, the native interpreter, both main and safety TLA emitters, names traversal, and existing replay/schema consumers; the older five-owner estimate is not an exemption. Optional value task .2 remains serial because those owners overlap.

Freeze the shared typed fact interface before parallel source lanes. Prefer one local Debug-only fieldless enum, Unit for legacy fact metadata, and typed Seq<Fact> branch expressions. No Clone/Copy/Finite requirement, TLS fact recorder, generic Option IR or new evaluator. Preserve existing outcome/fact independence: a completed Some after an unchanged rejecting unit helper retains that fact; rejection-after-mutation still fails. Evaluate authored tail conditions even when the unchanged handler is disabled, preserving Rust errors. Preserve all tracked old expansions and artifacts exactly; necessary hidden embedded IR constructor fields may be added. Canonical Option paths must be resolved by identity, not final-segment spelling.

The task owner is the sole committer and owns integration/gates. After a compiling shared-interface checkpoint, disjoint lanes may own source/machine admission; typed bridge/interpreter; symbolic emitters; and new fixture/replay controls. Use isolated task-named worktrees for concurrent source writers, reconcile all lanes before generation and staging, and remove retired worktrees normally after preserving evidence. Read-only evidence checks may start earlier. The conductor retains MILESTONES.md.
## Acceptance
- [ ] Public/private and unit/declared-World machines mix unit and optional handlers correctly. Unsupported returns, payload/discriminant/alias/nested carriers and mixed carriers fail the appropriate compile/source controls.
- [ ] Native changed-None, unchanged-Some, ordinary empty, stay, rejection and invalid-mutation controls preserve enabledness and exact fact cardinality.
- [ ] Independent source facts follow the authored SSA branch; source-only swaps alter emitted facts; unsupported returns/helpers refuse at their source; no previous fact leaks into an ordinary edge.
- [ ] Required TLC executes all four value traces and false-claim replay. Wrong valid enum, omitted/duplicate/extra ordinary-edge facts fail replay; unknown values fail decoding.
- [ ] Old unit expansions and all pre-existing artifacts/metadata remain byte-identical. New fixture goldens, focused/full gates and actual timing evidence are retained.


## Done summary
Optional handler facts now pass through native adapters, independent SSA lowering, typed interpretation and both TLA emitters. The new generic fixture exercises four forced fact witnesses and clearing; all 799 old artifacts and 15 frozen originals remain byte-identical.

Tier:session jev-unavailable(no_key), explicitAGENTS implementer6.1medium/reviewer6.1high; samefamilydisclose.
stage: impl-review - ran [2026-10-24T11:46:40Z..2026-10-24T11:52:28Z] (SHIP; three fresh codex:gpt-6.1-sol:high draws, zero findings)
baseline: green (focused six-crate command, exit 0 before edits)

Full gates: make model-check exit 0, 401.5006 s; UMPIRE_BACKENDS=require make model-check exit 0, 561.2351 s; make model-time exit 0, 361.2700 s wall. Timed gate 352.224 s exceeds the 60 s soft target; inner loop 8.910 s meets 10 s. The individual test-stage timing 197.:00s is inconclusive and retained verbatim.

Native fixture: 7 states, 31 edges. TLC: 57 distinct observer states, 248 generated; five counterexamples replayed. Controls cover all adapter combinations, Debug-only facts, native enabledness/rejection, SSA and canonical constructor refusals, source/IR fact-only mutants, exact replay vector forgeries and observer preservation. Required TLC executes both selected Worlds. Safety emission was controlled; live Apalache remains deferred.

Correctness audit: one Should Fix, resolved with a red-to-green canonical-path control. Standards audit: zero findings. Primary review: all three draws SHIP. Reviewer and implementers are from the same family under the explicit routing choice.

Evidence: .flow/artifacts/fn-237-activity-timeouts-heartbeat-and/ (execution record, raw quality reports, TLC witnesses, gate attempts, timing and manifests). Linked guest bytes are unchanged. All four writer worktrees were removed normally after verified task-owned archives. No Activity policy, optional-value support, old golden, canary pin or MILESTONES change. The broader legacy Go tooling baseline remains inherited red outside the active model gate.

GATE_SKIPPED:model-check:green-receipt 5be80b30 - baseline reused from prior post-gate pass
GATE_SKIPPED:model-check-required:green-receipt 5be80b30 - baseline reused from prior post-gate pass
GATE_SKIPPED:model-time:green-receipt 51dc99f5 - baseline reused from prior post-gate pass

Tracker sync: n/a (bridge inactive)

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: a7e9bfcad8ba399d1a24e5b1f4e9c8aeb78e4de8, 84dfc0a328b6ab6f28a7232c5ea6695a7def687f, d47ad5a4d6a01ebad93ac9e8d3adcba8c820ebfb, ba14a34a430999ee21eede8f3a04b8891b3fbb66, 5be80b307fb9ddc2c46eb5a37905ee42ac1f23f3, 51dc99f53cd6c4e470160664a24c54a1903b8a1e
- Tests: make model-check, UMPIRE_BACKENDS=require make model-check, make model-time, cd model && mise exec -- cargo fmt --all --check, cd model && mise exec -- cargo clippy -p umpire-dsl -p umpire-dsl-macros -p umpire-front -p umpire-checker -p umpire-tla -p umpire-tlc -p umpire-testing -p umpire-fixtures --all-targets --all-features -- -D warnings, cd model && mise exec -- cargo test -p umpire-tla --test optional_facts, cd model && UMPIRE_BACKENDS=require mise exec -- cargo test -p umpire-testing --lib, python3 .flow/tmp/fn2371-evidence-audit.py final, GATE_SKIPPED:model-check:green-receipt 5be80b30 - baseline reused from prior post-gate pass, GATE_SKIPPED:model-check-required:green-receipt 5be80b30 - baseline reused from prior post-gate pass, GATE_SKIPPED:model-time:green-receipt 51dc99f5 - baseline reused from prior post-gate pass
- PRs: