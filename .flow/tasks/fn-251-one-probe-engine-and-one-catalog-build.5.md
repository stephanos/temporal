---
satisfies: [R5, R7]
---
# fn-251-one-probe-engine-and-one-catalog-build.5 Generated annotation dispatch and AnnotationRuntime::with_probes

## Description
Implement **Transport dispatch (B8)**: generate `annotate_<svc>(..)` and add `AnnotationRuntime::with_probes(rt, probe_fn, annotate_fn)`; replace `kettle/transport.rs:13-61`, `temporal/feature/activity/api_transport.rs:19-58`, `temporal/realization/src/annotations.rs:25-40`.

## Acceptance
- [ ] One dispatch path; unknown prefix or method refuses as before; Go host tests pass.
- [ ] Re-accept goldens after reviewing their diffs; `make model-check` passes.


## Done summary
Kettle and Temporal now route probes and annotations through one typed runtime adapter and generated annotation dispatch. The generated macros retain exact RPC refusal rules, OutOfScope explanations, actual wire evidence, Kettle's per-resource limits and Temporal's supplemental WorkerFailure findings.

Task: fn-251-one-probe-engine-and-one-catalog-build.5
Status: in_progress, awaiting conductor integration, review and joined validation.
Tier: session (jev-unavailable(no_key))
Workspace: /Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/fn2515-transport

Generated API:
- `annotate_<svc>!(method, response, bytes, seed, limits_factory)` expands in the catalog owner. The factory receives `(Method, bool, &[u8])` and returns `Result<Context, Refusal>`. It runs only for in-scope RPCs. Kettle decodes Fill before selecting `Limits::for_kettle`; Temporal supplies its default context.
- The generated method enum exposes `parse_rpc(&str)` for bare names and exact `/service/Method` paths, and `out_of_scope()` from the admitted source classification. `dispatch_<svc>_annotations!` is replaced and all in-tree consumers updated.
- `AnnotationRuntime<D, ProbeRequest, ProbeResponse>::with_probes(rt, probe_fn, annotate_fn)` owns Any admission, typed request routing and response packing. The probe callback has signature `fn(&ProbeRequest) -> Result<ProbeResponse, Refusal>`. The annotation callback has signature `fn(&AnnotationRequest) -> Result<AnnotationResponse, Refusal>`. Catalog `AnnotationResult` converts into the shared response. Other methods and commit delegate unchanged.
- The generated API depends on downstream Inspect implementations only when the macro expands. No realization dependency enters the API crate, and no system names enter the generic crates.

Validation:
- Focused baseline passed 185 tests before edits. Final identical five-crate command passed 187 tests, including the new runtime adapter test, Kettle route table, Temporal directions/path controls and the retained annotation/catalog/probe/publication/admission controls. Exact command is in evidence JSON. Logs are `fn2515-baseline.log` and `fn2515-verify.log` beside this summary.
- Generator test first failed on missing `parse_rpc`; runtime adapter test first failed on missing `with_probes`. `fn2515-red.log` and `fn2515-runtime-red.log` preserve those observations.
- Strict scoped all-target/all-feature clippy passed. Formatting and diff whitespace checks passed. Both native guest entrypoints compile with all features and locked dependencies. Logs are `fn2515-clippy.log`, `fn2515-fmt.log` and `fn2515-guests.log`.
- Extra boundary check failed on the inherited `catalog/syntax/tests/admission.rs:7` system-named mirror fixture. Standalone snapshot ownership failed on the inherited registry still naming `temporal/annotate/syntax/tests/snapshots/expansion__typed-catalog.snap`, leaving the moved catalog snapshot without an owner. Both files are unchanged from the base commit. The conductor explicitly owns the later fixture/registry fixes. Logs are `fn2515-boundary-ownership.log` and `fn2515-ownership.log`; these checks are red, not successful task gates.

Ownership and remaining obligations:
- The former API-only typed message selection test moved to `temporal/realization/tests/annotations.rs`, where actual downstream catalog implementations validate request/response direction and OutOfScope behavior. The generator source assertion now pins the generated annotate_message call. No snapshot, golden or protocol/schema bytes changed, and no Go protocol regeneration is needed.
- Existing tests retain supplied and deterministic request IDs, unknown response fields/full response bytes, scoped Fill limits, WorkerFailure supplement findings, probe identities and actual-observation classification. No generic probe engine work from .4 was included.
- The conductor must integrate the root-owned fixture/registry fixes, perform independent review, regenerate current guests and closure manifests as needed, review any resulting goldens, and run the joined `make model-check` including Go hosts. Native guest compilation here does not claim Wasm or Go execution. Full gate and committed guest regeneration were explicitly deferred by dispatch. Gate classification reported FULL because executable Rust paths changed; no full-gate receipt was written.

stage: impl-review - skipped(policy: parallel-wave - conductor owns the gate)

Conductor acceptance: integrated in umpire; actual three-draw gpt-6.1-sol high SHIP review finalized (same-family CLI). Current transport/tooling composition at 8da35e97e4 passed the complete TLC/Apalache-required UMPIRE_APALACHE=require make model-validate, exit0, full1201.646s, inner6.370s. Its model tree remained unchanged; current guest freshness, workspace strict checks/tests, standard wasm/native controls and all three Go hosts passed. The earlier failed attempt lacked only the isolated checkout TLC jar; the pinned installer and exact previously failed real-TLC control passed before the successful retry. Durable receipt: .flow/artifacts/consolidation-20261025/transport-tooling-validation-8da35e97e4/receipt.json. Later independent consolidation changes retain their own final parent gate.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 67ad152c7466ead36072699eee6a50a985fd19a6, 8da35e97e4, 6079dbaddd
- Tests: cd model && timeout 600 mise exec -- cargo test -p umpire-catalog-build -p umpire-runtime -p umpire-temporal-api -p umpire-temporal -p umpire-reference-kettle --tests, cd model && timeout 600 mise exec -- cargo clippy -p umpire-catalog-build -p umpire-catalog -p umpire-runtime -p umpire-temporal-api -p umpire-temporal -p umpire-reference-kettle --all-targets --all-features -- -D warnings, cd model && mise exec -- cargo fmt --all -- --check, git diff --check, cd model && timeout 600 mise exec -- cargo check -p umpire-kettle-guest -p umpire-temporal-guest --all-features --locked, FAILED inherited: cd model && timeout 600 mise exec -- cargo test -p umpire-temporal-tests --test boundary --test snapshot_ownership --locked (exit 101; boundary stopped execution before snapshot_ownership), FAILED inherited: cd model && timeout 600 mise exec -- cargo test -p umpire-temporal-tests --test snapshot_ownership --locked (exit 101; stale registry path), Three-draw actual Codex gpt-6.1-sol high SHIP implementation review, finalized, UMPIRE_APALACHE=require make model-validate (8da35e97e4 unchanged model tree; exit0;1201.646s full,6.370s inner; allTLC/Apalache/workspace/guests/3Go hosts passed)
- PRs: