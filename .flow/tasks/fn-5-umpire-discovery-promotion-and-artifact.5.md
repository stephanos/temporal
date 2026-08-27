---
satisfies: [R4, R7]
---
# fn-5-umpire-discovery-promotion-and-artifact.5 Bind and expose Temporal promotion proposals

## Description
Bind reusable in-memory promotion proposals to deterministic compiling Temporal source and expose the effect-thin proposal executable for R4/R7.

**Size:** M
**Files:** `model/Temporal/Tool/PromotionBinding.lean`, `model/Temporal/Tool/Promote.lean`, `model/Temporal/Tool/PromoteTests.lean`, `model/TemporalModelTests.lean`, `model/lakefile.toml`, `tools/umpire/integration/promotion_source_test.go`
**Touches:** [model/Temporal/Tool/PromotionBinding.lean, model/Temporal/Tool/Promote.lean, model/Temporal/Tool/PromoteTests.lean, model/TemporalModelTests.lean, model/lakefile.toml, tools/umpire/integration/promotion_source_test.go]

### Approach

- Define a checked Temporal `PromotionCandidateBinding` with explicit fresh promoted IDs, required module imports, the typed original Query/run/target/kernel values, and accepted Query, run, target, and planner-kernel constant names; include binding Behavior Fingerprint in the proposal/source envelope without changing the reusable catalog Behavior Fingerprint.
- Seal `CompiledPromotionSource` construction behind candidate-module elaboration of the exact emitted declaration against those imports and typed constants. The production registry and CLI accept only this token, so failed elaboration prevents candidate registration/build and cannot produce status-0 source output.
- Expose `temporal-model-promote <candidate-identity>` only for complete in-memory candidates registered by Temporal, returning canonical `umpire-promotion-proposal/v2` JSON with lineage-linked promoted identities and deterministic Lean source.
- Keep pure binding/render/compile functions separate from IO; enforce one-LF stdout and structured stderr.
- In a tagged Go integration test, obtain the exact already-elaborated bytes from the production command, write only to `t.TempDir`, and invoke `mise exec -- lake env lean <temp-file>` from the model root so the structural gate is independently defended.
- Never accept arbitrary source, artifact JSON, runtime evidence, or filesystem output paths.

### Investigation targets

**Required:**
- `model/Temporal/Tool/Inspect.lean:23-88` — effect-thin executable pattern.
- `model/Temporal/Tool/InspectTests.lean:10-71` — exact result tests.
- `model/lakefile.toml:1-20` — current executable registration.
- `model/Temporal/Feature/Nexus/CallerClosure.lean:510-658` — one candidate registry source.
- `model/Umpire/Examples/Switch.lean:307-611` — reusable candidate source.

### Quick command

`cd model && mise exec -- lake build Temporal.Tool.PromoteTests temporal-model-promote && cd .. && go test -count=1 -tags test_dep ./tools/umpire/integration -run TestPromotionSourceCompiles`

## Acceptance
- [ ] Candidate bindings validate fresh promoted IDs, required typed values, imports, and accepted constants, have stable Definition IDs, and remain outside reusable `Umpire` catalog semantics.
- [ ] Only successfully elaborated exact source bytes can produce a `CompiledPromotionSource` or enter the closed production registry; invalid source makes model build/registration fail before CLI emission.
- [ ] Promote output is deterministic with exact stream and exit behavior; unknown/incomplete candidates and invalid bindings return structured errors.
- [ ] Promotion output binds original lineage identities, new checked proposal identities, source query, Behavior Fingerprints, binding Behavior Fingerprint, and deterministic Lean source.
- [ ] The production-rendered bytes compile through `lake env lean` from a test-owned temporary file; missing imports and stale/unqualified constants fail.
- [ ] CLI code remains effect-thin and calls the reusable promotion API.
- [ ] No command reads raw artifacts, runtime evidence, or user-authored Lean.
- [ ] Focused and aggregate Lean tests pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
