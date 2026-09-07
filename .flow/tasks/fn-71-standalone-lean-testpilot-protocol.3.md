---
satisfies: [R3, R4, R5]
---
# fn-71-standalone-lean-testpilot-protocol.3 Build context-safe Testpilot authoring and ProtoJSON APIs

## Description
Add the small developer-facing module over generated messages and establish the sole public ProtoJSON policy wrapper. Producers receive concise constructors returning generated values directly, while serialization delegates wholly to `Protobuf.Json`.

**Size:** M
**Files:** `model/Testpilot/Authoring.lean`, `model/Testpilot/ProtoJSON.lean`, `model/Testpilot.lean`, `model/TestpilotTests/Authoring.lean`, `model/TestpilotTests/AuthoringFailures.lean`, `model/TestpilotTests/ProtoJSON.lean`, `.plans/UMPIRE4_SPEC.md`
**Touches:** [`model/Testpilot/Authoring.lean`, `model/Testpilot/ProtoJSON.lean`, `model/Testpilot.lean`, `model/TestpilotTests/**`, `.plans/UMPIRE4_SPEC.md`]

### Approach
- Provide pleasant smart constructors for Cases, Programs, Contracts, values, paths, expressions, instructions, monitors, Run coordinates, limits, Verdicts, and opaque provenance; return generated protocol values rather than parallel records.
- Keep shared combinators statically specialized to Program or Contract context. Expose no broad expression union, raw-JSON constructor, implicit coercion, or normal authoring route through generated oneof ceremony.
- Preserve Nexus3 feature syntax beside Nexus3; the neutral API supplies only reusable Testpilot construction.
- Implement `Testpilot.ProtoJSON.canonical` as a thin typed-error wrapper over `Protobuf.Json`, with centralized print options and generated-pool resolver configuration.
- Test every reference family and recursive combinator, both Run identity encodings, presence/default behavior, integers, floating values, bytes, enums, monitor rules, `Any`, repeated determinism, strict Go decoding, and propagated failures.

### Investigation targets
**Required** (read before coding):
- task 2 generated declarations
- `model/Umpire/Case/Value.lean`
- `model/Umpire/Case/Program.lean`
- `model/Umpire/Case/Contract.lean`
- `model/Temporal/Testpilot/TestpilotProtoJSON.lean`
- Lean-zh/protobuf `Protobuf.Json` public API at the pinned revision

**Optional** (reference as needed):
- `model/Temporal/Feature/Nexus3/Syntax.lean` for the feature-local boundary only
## Acceptance
- [ ] Public helpers cover the complete current Testpilot authoring vocabulary and return generated messages directly.
- [ ] Direct and nested cross-context references fail elaboration through paths, comparisons, negation, all, and any; valid references and both Run identity forms remain constructible.
- [ ] `Testpilot.ProtoJSON.canonical` contains no field-by-field serializer and centralizes explicit print/resolver policy.
- [ ] Representative values, presence, integer limits, bytes, enums, `Any`, expressions, and monitor rules serialize deterministically and strictly decode in Go.
- [ ] Resolver, value, and serialization failures propagate without dropping fields, rules, or values.
## Done summary
Added the producer-neutral `Testpilot.Authoring` facade over generated protocol values and the sole typed `Testpilot.ProtoJSON` policy wrapper over `Protobuf.Json`. Colocated Lean coverage proves the complete current vocabulary and recursive context safety, while the focused fixture gate verifies deterministic serialization, typed failures, and strict Go decoding; the user retains ownership of all uncommitted changes.

stage: impl-review - ran [2026-09-07T05:12Z..2026-09-07T05:15Z] (SHIP; `/tmp/fn71-task3-impl-review.json`)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: baseline: green — make umpire-check-testpilot-protocol, baseline: green — (cd model && mise exec -- lake build Testpilot TestpilotTests), make umpire-check-testpilot-authoring, (cd model && mise exec -- lake build Testpilot TestpilotTests UmpireTests TemporalModelTests temporal-testpilot), (cd model && mise exec -- lake exe modelLintTests), make umpire-check-case-runtime-conformance, CGO_ENABLED=0 TMPDIR=<physical-temp-root> mise exec -- go test -count=1 -p=1 -tags test_dep ./common/testing/testpilot/... ./tools/umpire/cmd/umpire-gen-case-runtime-conformance ./tests/testcore/testpilot/..., make lint-model, gofmt -d tests/testcore/testpilot/protobuf_lean_authoring_test.go, git diff --check -- <task paths>, baseline inherited red: make lint-code GOLANGCI_LINT_FIX=false (1361 pre-existing findings recorded by fn-71.1/.2; not repeated)
- PRs:
