---
satisfies: [R1]
---
# fn-211-apalache-backend-for-umpire-in-rust.1 Typed Apalache safety relation, checked arithmetic and postcondition history

## Description
Implement R1 and the parent Backend implementation contracts for emitter compatibility, checked arithmetic and postcondition history. The approved task is a source-derived Apalache safety projection with Snowcat types, not comments alone. Preserve the existing TLC relation and answers. No source transition may be inferred from a native explored graph.

**Size:** L. **Dependencies:** fn-210, fn-212 and fn-227 must be closed and integrated on umpire before admission; no worktree is needed for the single writer.
**Touches:** [model/umpire/lift/**, model/umpire/testing/**, model/umpire/backends/**, model/umpire/tests/**, model/temporal/feature/activity/expected/**, model/scripts/**, model/Cargo.toml, model/Cargo.lock, Makefile]
These are the existing emitter/proof surfaces plus narrowly necessary pinned-tool admission and shared process support. Do not change authored production Models, Go runtime/protocol, guest policy or third-party dependencies. Source/front/IR extensions beyond existing admitted expressions are not expected here; report a concrete need before widening implementation scope. Task .2 finalizes reusable runner lifecycle, .3 owns general typed ITF replay, .4 owns the fixture consumer, and .5 owns symbolic Worlds.

### Approach

1. Reanchor the parent contracts and completed fn-227 named-local IR, bindings, complete starts and exact action identities. Record baseline source/guest hashes and artifact inventory before edits. Read actual lift metadata and TLC runner helpers; choose the smallest explicit safety-mode interface, preserving TLC by default. Do not add Apalache-specific annotations to authored Models.
2. Emit Snowcat annotations from existing Rust/source types for every emitted variable, constant and operator, including generated monitor/history helpers and locals. Unsupported mappings are located refusals naming the type. An explicitly selected Apalache module has typed Init and Next with lifted real actions plus tagged stutter preserving state and monitors; it contains neither ENABLED nor imperative TLC Assert/Complete semantics. TLC completion and liveness remain their existing route.
3. Translate evaluated expressions as typed value plus validity, preserving Rust evaluation order, short-circuit guards, untaken branches and named-local sharing. Total fallback values cannot hide an evaluated error. Latch a distinct arithmetic-error monitor for evaluated overflow, zero division/remainder and signed MIN/-1, including starts and claims. Pin successor overflow, intermediate overflow whose final mathematical value is in range, zero division, MIN/-1 and guarded unevaluated errors. Arithmetic refusal must be distinguishable from authored claim failure and tool NotRun.
4. Emit one-step history for postconditions: complete previous state, exact concrete action identity and outcome/facts as used, has_step=false with typed defaults in Init, and true only for real steps. Stutter preserves history. Exact after filters apply; add a failing postcondition, filtered negative control and unrelated-action control. Metadata distinguishes invariants, postconditions and arithmetic checks and supports later declared monitor projection and replay.
5. Verify the current official release and selected release CLI/source before choosing and checksum-pinning the artifact. Run actual pinned Apalache on Kettle and the required arithmetic/postcondition controls during this task. Tool admission may begin here but must use approved workspace build-root containment and reuse/extract existing TLC process patterns instead of copying orchestration. Do not rely on unversioned docs or treat a refused/timeout/malformed check as green. Keep deterministic output free of times, absolute paths and native-graph substitution. Preserve actual command outputs and enough typed offending-action/prefix evidence for .3 replay integration.
6. Verify focused tests and strict lint, then the complete required model-time once after fixes/reviews settle. Preserve native/TLC answers and unrelated artifacts exactly; justify only annotation/safety-mode outputs. Actual guest/production changes require corresponding live slice and gaps; exact unchanged guest and production hashes permit explicit reuse of retained .5 proof, not a fabricated gate receipt. No fresh cross-OS claim.

### Verification

Quick from model/: `mise exec -- cargo test -p umpire-lift --tests`. After changes run focused safety/annotation tests, actual pinned-tool controls, formatting and strict clippy for changed crates. Retain red/green controls that discriminate the required semantics. Final from repository root: `UMPIRE_BACKENDS=require make model-time` (one invocation subsumes default/full/required checks and timing; retain all 17 stages and soft-target misses). Use an outer timeout at least 2400 seconds. Do not run duplicate full gates solely for Flow evidence commits.

### Required evidence

Exact selected release/version/checksum and actual tool invocations, located unsupported-type refusal, fully typed emitted fixtures, discriminating arithmetic/history/filter controls, unchanged TLC results, native/guest inventory, reviewed final source checkpoint and terminal full-gate output. Missing or non-replayable evidence is incomplete rather than success. No new user input is needed for routine choices within these contracts.
## Acceptance
- [ ] Every emitted variable, constant and operator has Snowcat types derived from the admitted source; unsupported mapping refuses with its type and location.
- [ ] Explicit Apalache safety Init/Next use independently lifted actions plus tagged preserving stutter, without ENABLED/TLC Assert/Complete; the TLC route and results remain unchanged.
- [ ] Evaluated arithmetic faults latch a distinct reachable failure; red/green controls cover successor and intermediate overflow, zero division/remainder, MIN/-1 and guarded unevaluated errors, including starts/claims.
- [ ] Typed one-step history checks postconditions under exact after-action filters; failing, filtered and unrelated-action controls discriminate the behavior, and stutter preserves history.
- [ ] Actual checksum-pinned Apalache checks Kettle and the required arithmetic/history controls from workspace-owned build scratch; source/property metadata and offending prefix/action evidence support task .3 typed replay.
- [ ] Focused tests, strict lint and one final required model-time pass; retained native/TLC artifacts and guest/production proof justify every changed output and any explicit live-evidence reuse. Timings retain soft-target misses.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
