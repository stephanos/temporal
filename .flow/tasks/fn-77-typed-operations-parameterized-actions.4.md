---
satisfies: [R1, R5, R8, R9]
---
# fn-77-typed-operations-parameterized-actions.4 Integrate parameterized Actions with finite and runtime domains

## Description
Integrate parameterized Actions with finite and runtime domains for the referenced parent requirements.

**Size:** M
**Files:** model/Umpire/Operation/**; model/Umpire/Core.lean; model/Umpire/Target/**; model/Umpire/Query/**; model/Umpire/Planning/**; model/Umpire/Artifact/Planning.lean; model/Umpire/Artifact/Types.lean
**Touches:** [model/Umpire/Operation/**, model/Umpire/Core.lean, model/Umpire/Target/**, model/Umpire/Query/**, model/Umpire/Planning/**, model/Umpire/Artifact/Planning.lean, model/Umpire/Artifact/Types.lean]

### Approach
- Bind checked operation templates to exact admitted argument instances; use generic Target/FiniteMachine semantics so prior state and arguments determine allowed alternatives, never caller-selected outcomes.
- Add the smallest checked versioned bridge needed by ordinary ModelValue consumers. Preserve old Atom/literal bytes when no extension is used; parameter equality is typed canonical equality, not display-string comparison.
- Extend existing finite adapters to enumerate exactly explicit instance domains with soundness/completeness. Record fixed, sampled and abstracted dimensions; reject unsupported stronger abstraction claims.
- Represent separately declared runtime admissibility and resource bounds. A runtime value outside finite samples either satisfies this admission or rejects out-of-scope without changing the finite verification claim.
- Update instance/domain identity and receipt encoding only through named versioned semantics; add exact replay and incomplete-search tests.

### Investigation targets
**Required:**
- model/Shared/SemanticData.lean:8 — current Atom representation.
- model/Umpire/Core.lean:379 — authoritative generic TransitionKernel.
- model/Umpire/Target/Semantics.lean:15 — private checked Target.
- model/Umpire/Target/FiniteMachine.lean — existing finite authority.
- model/Umpire/Planning/Engine.lean — candidate traversal and validity receipts.

### Quick commands
`cd model && mise exec -- lake build Umpire.TargetTests Umpire.Query.Tests Umpire.Planning.Tests Umpire.Artifact.Tests.Codecs`

`cd model && mise exec -- lake build Umpire.Target.Tests.Parameterized`

### Execution constraints
Read the full parent spec. Preserve existing comments and unrelated dirty source. New paths listed here are proposed owners; reuse an established equivalent before creating one. Run Lean jobs serially; fn76 lint recovery is complete. Preserve fn75 semantic import isolation and fn78 obligation/evidence authority. No operation cancellation, new dependency/toolchain, or implicit fixture promotion. Do not stage, commit, or push; the user owns commits. Capture task-local before/after trust for changed load-bearing declarations; task11 additionally compares to the original task1 substrate.

Create and wire the proposed test module Umpire.Target.Tests.Parameterized into its existing aggregate test root; if an existing equivalent is reused, record its exact name and run it explicitly. The new-module Quick command applies after creation; baseline existing roots before editing instead of treating an absent module as an environmental failure.

## Acceptance
- [ ] Finite enumeration equals authored parameter domains and retains Target-owned nondeterministic/error alternatives.
- [ ] Runtime out-of-sample admission is explicit and cannot broaden finite completeness; unsupported abstractions and exhausted search remain distinct.
- [ ] Parameterized semantic edits change intended identities; absent extensions preserve old canonical bytes, fingerprints and replay behavior.
- [ ] Bridge round-trip/denotation and finite proof obligations compile without new trust; focused domain/planning tests pass.

## Done summary
# fn77 task4 implementation handoff

Task: fn-77-typed-operations-parameterized-actions.4. Native status remains **in_progress**. Only task4 source was changed. No staging, commits, push, branch/worktree changes, Flow mutation, review, or done action. `commits=[]`; `prs=[]`.


### Implementation

- `Umpire.Operation.Action`: checked stable authored Action template retains task1 generated RPC owner, Request/Response witness and Failure type. An instance contains only task2 `Value.Checked` request arguments. Canonical request-tree equality proves typed instance equality; no outcome selector exists.
- `Umpire.Operation.Canonical`: version 1 exact structural schema/value serialization, including full request/response schema graphs, descriptors, schema inputs, field shapes, presence/defaults and streaming flags. Instance keys use exact tree bytes in a catalog-compatible alphabet, never display output or copied method strings. Generated spelling alone is not connectivity.
- `ParameterDomain.check` admits exactly authored vectors and rejects duplicate canonical arguments, unsupported abstractions and invalid fixed domains. Fixed means one whole-request vector; sampled means only the listed vectors. Runtime scope separately declares samples-only or schema bounds; caller work/resource limits are independent. Schema-admitted out-of-sample values never enter the finite catalog or its bridge decoder.
- `Umpire.Target.Parameterized` checks exact catalog agreement and reuses the existing validated FiniteTable, FiniteMachine, TransitionKernel and checked Target. `catalog_exact`, `actionDomain_iff`, and `step_iff` establish exact finite membership and every prior-state/argument-selected row alternative. The checked Atom bridge's `decode_sound` and `decode_encode` concern its actual lookup, returning the original typed request, rather than a copied equality certificate.
- The versioned `umpire-parameter-domain/v1` metadata receipt includes fixed/sampled scope, exact samples and runtime bounds. Instance edits change their canonical keys; domain/schema/runtime semantic edits affect Target behavior identity. Reordering the authored sample list preserves semantic fingerprint. Work-budget changes do not change instance identity.
- Added `Umpire.Target.Tests.Parameterized` to `Umpire.TargetTests`; planning/replay tests live in their existing Planning test owner and aggregate. This respects the Target-to-Planning import boundary.

### Compatibility and evidence

Existing Atom, ModelValue, DefinitionId, canonical Artifact codecs, Query, Planning and generic Target/FiniteMachine source bytes are unchanged. The extension is selected by importing/binding the new template/domain adapter. Legacy literal and canonical-byte goldens, Definition identity, migration compatibility and replay tests run through the normal roots. The only edits to pre-existing task4 source are appended test imports; all original lines/comments survive exactly.

The new bridge is domain-relative: it requires a checked selected template/domain, and decodes only its finite admitted values. It does not infer an owner from an arbitrary Atom or accept a display label as a typed value. Successful bridge denotation is exercised by decoding an Atom, constructing the task3 checked field reference/cursor and reading the exact bytes. Wrong owner, Request/Response types and payload side fail elaboration; wrong schema/type, DefinitionId, version and display-only keys reject.

Native existing Quick roots were baselined before module creation. Final Quick, explicit Parameterized root, full model build and a real generated `WorkflowService.startWorkflowExecution` binding are recorded with actual exits in the command journal. No generator or Go source changed. Formatting is checked with `git diff --check`; Lean built-in lint checks the seven affected sources with warnings treated as errors.

Meaningful negative controls cover omitted alternatives, broadened sample-only runtime scope, ignored semantic bounds, unsupported abstraction admission, erased parameter identity and erased schema identity. Each compile-valid mutant fails semantic guards, while the control compiles. The versioned-domain receipt also has a genuine failing assertion before its implementation change. Initial missing-module/setup/elaboration failures and a malformed first outcome-mutant harness are retained as development failures, not presented as semantic counterexamples. Subsequent error-class guards explicitly distinguish unsupported abstraction, runtime resource failure and out-of-scope from planner incomplete search. Planning budget 1 reports incomplete limit-reached; budget 100 verifies only the finite domain. Exact successful replay passes and forged result/state replay rejects.

### Trust

Complete pre/post production owner-closure inventories contain **6,367 → 6,692 declarations**. All 6,367 prior declaration types, ownership mappings and transitive axiom arrays match exactly; none were removed. All 325 additions have exhaustive per-declaration analogue mappings. No new custom, sorry or compiler-trust axiom appears; additions use subsets of the existing `propext`, `Classical.choice`, `Quot.sound` boundaries. The inventory includes full declaration types and transitive axiom arrays, not definition bodies. No task4 test uses `native_decide` as a proof.

The original task1 frozen baseline (715 files and four external raw captures), dependency summaries/evidence/frozen manifests/final review receipts, task-local originals/absences, staged entries and source hashes are retained and verified. Patch reconstruction uses only the seven affected source files, without a worktree or build cache.

### Gates and limitations

- `make lint-code GOLANGCI_LINT_FIX=false` exits 2 with **exactly the same multiset of 1,284 diagnostics** as the latest accepted task3 baseline. No diagnostics added or removed. Its subsequent `go vet` recipe is not reached. No global Go-lint cleanliness is claimed.
- The model import graph gate passes after keeping Planning tests in their proper owner. Batteries model lint still reports the unchanged task1 `CheckedRpc.mk.injEq` simp-normal-form diagnostic. Full built-in lint exits 1 for the unchanged task2 `Value.Encoding.encodeNat.eq_1` looping-simp warning.
- Both model diagnostics were reproduced from the captured originals after task4 edits: the reconstructed original root explicitly excludes all task4 modules and verifies 215 imported first-party source hashes; direct original codec compilation reproduces the identical warning. This is post-edit reconstruction, **not** a pre-edit model-lint run or a prior accepted waiver. No global model-lint cleanliness is claimed.
- Final native gates are serialized by a single runner. An earlier candidate full-build launch overlapped an unfinished candidate Quick invocation; both exited 0, but those candidate receipts are not the serial verification claim. The final serial reruns are the authoritative completion evidence. No second cache/build tree was created.

### Supported domain and Known Gaps

This adapter supports task1 unary RPC templates and exact task2-supported bounded request values. Each finite dimension is an explicitly authored whole-request vector. It does not claim schema exhaustiveness, factorized per-field products, symbolic abstraction, or preservation of an abstraction relation; all `.abstracted` claims reject. Task2 exclusions (including unsupported concrete floating-point evaluation/groups/extensions) remain unchanged. Samples-only runtime rejects out-of-sample values; declared schema scope may admit them within semantic and resource bounds, without broadening finite proof.

The finite checked decoder is not an arbitrary runtime payload decoder. Runtime values use `admitRuntime` and the task2 checked codec. Canonical schema keys can be large; finite identity checking is pairwise and intended for explicit bounded domains. There is no universal cross-domain Atom ownership claim. No operation cancellation, live execution qualification, new Query language, portable protobuf lowering or Go runtime integration is introduced.

### Downstream handoff

- **Task5:** consume `ActionInstance.arguments` and task3 field/schema cursors under the retained template/witness. Effects remain Target-owned; never synthesize a selected result from the Action.
- **Task6:** capture exact typed values and selected generated schema/owner. Use canonical instance identity for concrete bindings and the versioned domain receipt for coverage/runtime semantics; display text is not equality. Preserve out-of-sample versus finite membership distinction.
- **Task8:** lower the retained generated declaration and checked arguments with exhaustive supported-domain handling. Keep the authored DefinitionId separate from selected generated connectivity. Carry the versioned domain/scope receipt, reject unsupported values/abstractions, and preserve exact replay plus Target-owned outcome alternatives. This task supplies no portable lowering proof.

Source is frozen for conductor review. No SHIP/done verdict is made; no jobs are run after the final freeze.

### Final native exits

- `mise exec -- lake build Umpire.TargetTests Umpire.Query.Tests Umpire.Planning.Tests Umpire.Artifact.Tests.Codecs`: exit 0.
- `mise exec -- lake build Umpire.Target.Tests.Parameterized`: exit 0.
- `mise exec -- make umpire-build-model`: exit 0.
- `mise exec -- lake env lean /tmp/fn77-task4-generated-action.lean`: exit 0.
- `GOMAXPROCS=1 GOFLAGS=-p=1 GOCACHE=/private/tmp/fn77-task2-go-cache GOCACHEPROG="python3 /private/tmp/fn77-task2-cache.py" mise exec -- make lint-model`: exit 2.
- `GOMAXPROCS=1 GOFLAGS=-p=1 GOCACHE=/private/tmp/fn77-task2-go-cache GOCACHEPROG="python3 /private/tmp/fn77-task2-cache.py" mise exec -- make lint-code GOLANGCI_LINT_FIX=false`: exit 2.
- `git diff --check`: exit 0.

Seven focused built-in lint invocations: exit 0 each. Full built-in lint: exit 1 (the reproduced inherited byte-codec warning).

stage: impl-review - ran; SHIP with no findings (model: gpt-5.6-sol at high)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: