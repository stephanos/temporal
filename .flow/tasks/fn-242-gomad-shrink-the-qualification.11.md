---
satisfies: [R2]
---
# fn-242-gomad-shrink-the-qualification.11 Independent 96-stage plan and shared cycle contracts

**Touches:** tools/gomad/rust/gomad/src/cycle/mod.rs, tools/gomad/rust/gomad/src/cycle/plan/**, tools/gomad/rust/gomad/src/cycle/model.rs, tools/gomad/rust/gomad/src/cycle/scheduler/mod.rs, tools/gomad/rust/gomad/src/cycle/process/mod.rs, tools/gomad/rust/gomad/src/cycle/config/mod.rs, tools/gomad/rust/gomad/src/cycle/custody/mod.rs, tools/gomad/rust/gomad/src/cycle/build/mod.rs, tools/gomad/rust/gomad/src/cycle/admission/mod.rs, tools/gomad/rust/gomad/src/cycle/journal/mod.rs, tools/gomad/rust/gomad/src/lib.rs, tools/gomad/rust/gomad/src/cycle/preparation/inputs/mod.rs, tools/gomad/rust/gomad/src/cycle/preparation/mod.rs, tools/gomad/rust/gomad/src/cycle/preparation/bindings/mod.rs, tools/gomad/rust/gomad/src/cycle/preparation/composition/mod.rs, tools/gomad/rust/gomad/src/cycle/readers/command/mod.rs, tools/gomad/rust/gomad/src/cycle/readers/mod.rs, tools/gomad/rust/gomad/src/cycle/readers/runtime/mod.rs, tools/gomad/rust/gomad/src/cycle/readers/workload/mod.rs, tools/gomad/rust/gomad/src/cycle/readers/inspection/mod.rs, tools/gomad/rust/gomad/src/cycle/readers/live/mod.rs, tools/gomad/rust/gomad/src/cycle/readers/core_codec/mod.rs, tools/gomad/rust/gomad/src/cycle/readers/core_guests/mod.rs, tools/gomad/rust/gomad/src/cycle/readers/core_inspection/mod.rs, tools/gomad/rust/gomad/src/cycle/readers/core_retained/mod.rs, tools/gomad/rust/gomad/src/cycle/readers/core_world/mod.rs, tools/gomad/rust/gomad/src/cycle/readers/core_lifecycle/mod.rs, tools/gomad/rust/gomad/src/cycle/readers/cells/mod.rs, tools/gomad/rust/gomad/src/cycle/final_readers/admission/mod.rs, tools/gomad/rust/gomad/src/cycle/final_readers/mod.rs, tools/gomad/rust/gomad/src/cycle/final_readers/profiles/mod.rs, tools/gomad/rust/gomad/src/cycle/final_readers/profile_reference/mod.rs, tools/gomad/rust/gomad/src/cycle/final_readers/retained/mod.rs, tools/gomad/rust/gomad/src/cycle/final_readers/native/mod.rs, tools/gomad/rust/gomad/src/cycle/final_readers/installed/mod.rs, tools/gomad/rust/gomad/src/cycle/final_readers/clusters/mod.rs, tools/gomad/rust/gomad/src/cycle/final_readers/terminal/mod.rs, tools/gomad/rust/gomad/src/cycle/final_readers/public/mod.rs, tools/gomad/rust/gomad/src/cycle/final_readers/reach_support/mod.rs, tools/gomad/rust/gomad/src/cycle/producers/static_adapter/mod.rs, tools/gomad/rust/gomad/src/cycle/producers/mod.rs, tools/gomad/rust/gomad/src/cycle/producers/primary/mod.rs, tools/gomad/rust/gomad/src/cycle/producers/build/mod.rs, tools/gomad/rust/gomad/src/cycle/producers/workload/mod.rs, tools/gomad/rust/gomad/src/cycle/producers/installed/mod.rs, tools/gomad/rust/gomad/src/cycle/producers/reference/mod.rs, tools/gomad/rust/gomad/src/cycle/producers/sqlite_terminal/mod.rs, tools/gomad/rust/gomad/src/cycle/producers/identity/mod.rs, tools/gomad/rust/gomad/src/cycle/final_readers/reference/mod.rs, tools/gomad/rust/gomad/src/cycle/final_readers/sqlite/mod.rs, tools/gomad/rust/gomad/src/cycle/readers/integration/mod.rs, tools/gomad/rust/gomad/src/cycle/routes/mod.rs

## Description
independent stage plan and shared cycle contracts

Depends on: .1. Produces the shared contract types and independent production stage-plan derivation consumed by every successor. Original .2-owned IDs: 3, all pending, exact list in partition.json.

Description: derive the actual Rust plan from fixed original stage semantics and candidate configuration. Preserve all 96 labels, original predecessor edges, bounds, repetitions, resources, locks, scratch, isolation and unchanged leaf argv byte for byte. Map orchestration substitutions only by .1's explicit original endpoint/argument ledger. Keep the stage table strictly an independent test oracle; production must not read or include that JSON. Repo-root-dependent bytes require explicit bound path substitution, never arbitrary normalization that hides changed leaves. Composition of fresh inputs remains .9; actual endpoint implementations remain their assigned owners.


Proof ownership: `.flow/artifacts/fn242-cycle-proof-ownership.json`, group `cycle-foundation`; inventory source binding and exact ID list remain authoritative. Keep original .2 aggregate acceptance; this task produces evidence, not qualification credit. Reuse existing validation and fold same-boundary bookkeeping only with a named surviving guarantee and executed catching/control proof.

Establish compileable type-only child-module ownership roots and register them serially. No callable success placeholder. Each successor must be able to compile/run its own production tests without editing shared registration; later newly introduced roots are foundation/conductor registration work before its worker starts.

Serially register the complete child ownership tree declared by `module_registration` in `.flow/artifacts/fn242-port-proof-ownership.json`, including preparation/readers/final_readers/producers parent roots and routes. These are compileable module/type seams, never callable success placeholders. Later task tests must compile through the real production module graph.

## Acceptance
- [ ] The production plan is derived independently of .flow/artifacts/fn242-stage-table.json and matches every original contract under explicit original/candidate root bindings.
- [ ] Catch changed leaf argument, label, resource/bound/dependency/repetition and undeclared orchestration substitution with executed mutations and valid control. Preserve schema/parser/invariant errors required by the 96-stage table, not only the three named source checks.
- [ ] Serial plan adds predecessor edges in original order while retaining original resource evidence; parallel plan retains original edges; both select identical stage subprocess argv.
- [ ] Public CLI remains aggregate-owned; internal plan/config types compile and expose only sibling requirements. No endpoint producer success is invented.
- [ ] Execute source-bound proof for its exact inventory IDs and preserve golden source/version/hash and path-substitution rules in proof output.

## Done summary
Implemented the independent 96-stage Rust plan, shared contracts and production leaf-command authority. Exact original wrapper/stage contracts and all 235 fixed recipes on 88 rows match under both modes and two explicit binding sets (940 recipe comparisons); exact eight dynamic and 32 runtime expansion contracts remain explicit. Actual leaf timeout mutation leaves the wrapper unchanged and is refused. Three owned guard countermutations and the actual leaf-source countermutation each fail the intended selected test; restored final source passes nine focused tests, strict Gomad library lint and formatting. Original golden/inventory and original evidence remain immutable. All ownership module roots compile in the production graph; no callable success placeholder or endpoint success is claimed.

Formal same-receipt Codex re-review returned SHIP, zero blocking findings. Integrated source equals reviewed source. R2 foundation coverage only; actual endpoints, cutover/global R1 audit and full qualification remain successor-owned. Full attempts remain 0 used / 2 available.

Tier: session jev-unavailable(no_key), project implementer pin retained.
stage: impl-review - ran (model: gpt-6.1-sol high; same family as writer; initial 3-draw NEEDS_WORK, same-receipt fix re-review SHIP)

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 860e004f2c, af38a745a9, b87cc8f329
- Tests: cargo +1.99.1 test --locked -p gomad --lib cycle::plan:: -- --nocapture --test-threads=2 (exit 0), cargo +1.99.1 clippy --locked -p gomad --lib --no-deps -- -D warnings (exit 0), cargo +1.99.1 fmt --all -- --check (exit 0)
- PRs: