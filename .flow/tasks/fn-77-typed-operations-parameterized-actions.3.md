---
satisfies: [R2, R3, R9]
---
# fn-77-typed-operations-parameterized-actions.3 Check typed schema field references and value access

## Description
Check typed schema field references and value access for the referenced parent requirements.

**Size:** M
**Files:** model/Umpire/Value/**; model/Umpire/Operation/**; tools/umpire/cmd/umpire-gen-lean-api/**; model/Temporal/API/**
**Touches:** [model/Umpire/Value/**, model/Umpire/Operation/**, tools/umpire/cmd/umpire-gen-lean-api/**, model/Temporal/API/**]

### Approach
- Expose generated typed field references for the full declared selected-operation schema, retaining field-number/containing-schema identity rather than Lean names.
- Build checked nested access, bounded repeated index/cardinality, map lookup, presence and oneof selection over task2 values. Access admission tracks descriptor-specific availability and types.
- Keep implicit scalar defaults distinct from explicit-presence fields; a selected oneof or established presence is required before consuming dependent values. Unsupported structural forms remain discoverable but reject requested evaluation.
- Prove admitted field-path denotation against task2 concrete values; add source-local negative references, wrong schema/oneof and out-of-range selection fixtures.

### Investigation targets
**Required:**
- tools/umpire/cmd/umpire-gen-lean-api/model.go:255 — full descriptor field metadata.
- common/testing/testpilot/internal/ir/path.go:54 — bound descriptor paths.
- common/testing/testpilot/internal/ir/path.go:151 — descriptor-presence restriction.
- model/Umpire/Property/Language.lean:61 — existing predicate contexts.

### Quick commands
`go test -count=1 -tags test_dep ./tools/umpire/cmd/umpire-gen-lean-api ./common/testing/testpilot/internal/ir`
`cd model && mise exec -- lake build Umpire.Property.Tests Testpilot.Tests`

`cd model && mise exec -- lake build Umpire.Value.FieldTests`

### Execution constraints
Read the full parent spec. Preserve existing comments and unrelated dirty source. New paths listed here are proposed owners; reuse an established equivalent before creating one. Run Lean jobs serially; fn76 lint recovery is complete. Preserve fn75 semantic import isolation and fn78 obligation/evidence authority. No operation cancellation, new dependency/toolchain, or implicit fixture promotion. Do not stage, commit, or push; the user owns commits. Capture task-local before/after trust for changed load-bearing declarations; task11 additionally compares to the original task1 substrate.

Create and wire the proposed test module Umpire.Value.FieldTests into its existing aggregate test root; if an existing equivalent is reused, record its exact name and run it explicitly. The new-module Quick command applies after creation; baseline existing roots before editing instead of treating an absent module as an environmental failure.

## Acceptance
- [ ] All declared selected-schema fields have stable structural references; exact executable support/unsupported diagnostics are explicit rather than sample-field whitelists.
- [ ] Nested, repeated, map, scalar presence and oneof access enforce schema types and availability, including missing and boundary cases.
- [ ] Checked access correspondence proves returned values/presence match admitted concrete field semantics.
- [ ] Negative source-location tests and unchanged literal/portable tests pass through normal owner roots.

## Done summary
# fn77 task3 implementation handover

Task `fn-77-typed-operations-parameterized-actions.3` is implemented and source-frozen for conductor review. Native status remains `in_progress`. No Flow mutations, review dispatch, child agents, CLI bridges, staging, commits, pushes, branch changes, worktrees, cancellation work, or other-spec implementation occurred. `commits=[]`; `prs=[]`.


### Delivered interface and semantics

`Umpire.Value.Field` supplies complete generated-schema field discovery, checked structural references, and schema-indexed cursors over actual task2 `Checked` values. Every reference retains the selected `RpcOwner`, payload-indexed witness, request/response side and containing schema. Reference membership is proved against the selected schema graph. `references_complete` proves that discovery returns every field, including unsupported and recursive forms. Generated `Temporal.API.fields` and `fieldReference` expose the same interface; there is no field-name whitelist or independently maintained schema.

`Cursor.typed` carries a `TypedPath` derivation from the selected schema; `Cursor.denotes` carries a `Denotes` derivation from its actual admitted payload. Primitive denotation constructors describe field-number lookup, exact proper lists, ordered indices, exact map keys and presence. `sequence_sound` proves the bounded parser produces those proper-list semantics. Access constructs these derivations from the concrete parser results, rather than storing an equality between an evaluator and itself. `Cursor.correspondence`, `length_correspondence`, and `scalar_correspondence` expose universal relationships; the scalar theorem additionally proves exact descriptor-type agreement. The origin is an actual `Checked` value, retaining its admission/resource proofs and original owner/witness indices.

`root`, `field`, `refine`, `establish`, `select`, `present`, `index`, `lookup`, `cardinality`/`length`, and `scalarValue`/`scalar` are the access interface. Refinement uses kernel equality of structural types/cardinality/availability and cannot establish presence. Optional fields and keyed lookups require `establish`; oneof members require `select` of the correct group before dependent use. Implicit scalars read task2's canonical descriptor defaults and reject invented optional-presence tests. Present defaults remain distinct from absence where descriptors permit it. Repeated order, bounds and cardinality are exact; cardinality is a Lean natural rather than a narrowed protobuf integer. Typed maps distinguish missing keys from malformed/wrong-kind/out-of-range keys. Lookup charges a comparison work ceiling and exact key byte/work ceilings. Errors retain full author path/line/column/provenance and structural field coordinates.

Byte content and integer kind/range remain exact. Unknown OPEN enum int32 values are retained. Float and special-form metadata stays discoverable while requested concrete access rejects explicitly. Recursive schema discovery has no concrete-depth cutoff; task2 admission rejects exhausted concrete traversal atomically, before a cursor can exist. No fabricated subtree, digest comparison, source-text type authority, or Temporal authority inferred from an arbitrary generic owner is introduced.

### Tests and verification

`Umpire.Value.FieldTests` is wired into `model/UmpireTests.lean`; this single aggregate import is the minimal change beyond task Touches. Generic tests cover defaults/presence, dependent availability, nested recursion, repeated empty/N/N+1 bounds, maps with text/bool/signed/unsigned keys, key kind/range/byte/work rejection, oneof selection, malformed inputs, unsupported float/group access, and exact source coordinates.

The existing generator fixture checks exercise actual generated references and task2 admitted values: concrete bytes, nested OPEN enum 99, optional present-default text, selected and unselected oneof branches, keyed nested messages, ordered repeated signed integers and cardinality. Compile-negative fixtures reject wrong owner (even with copied schema text), wrong side, wrong containing schema, optional scalar consumption and oneof consumption without selection. Existing operation/payload forgery fixtures remain unchanged. Fixture regeneration uses existing owner Make routes; ordinary tests do not rewrite fixtures.

Four isolated mutants all compiled and were caught by semantic assertions: field-number comparison, map-key comparison, key-range admission and lookup work bound. Temporary mutant modules used distinct namespaces and imported existing dependencies; no caches or build trees were copied or modified for mutation testing. Exact child commands, exits, durations and logs are in `/tmp/fn77-task3-mutants.json`.

Fresh final gates passed: tagged generator/internal-IR Go Quick; complete owner regeneration; `make umpire-check-lean-api` including actual Temporal.API and fixture compilation; and `lake build Umpire.Property.Tests Testpilot.Tests Umpire.Value.FieldTests UmpireTests` (301 jobs). Existing Quick roots were baselined before edits; the new FieldTests root was first invoked after creation. Pinned mise, per-command xcrun CC/SDKROOT, TMPDIR=/private/tmp, LEAN_NUM_THREADS=1 and the delivered task2 Go cache workaround were used. Build commands ran serially.

Nonfixing `make lint-code GOLANGCI_LINT_FIX=false` exits 2. Its 1,284 diagnostic occurrences exactly equal task2's latest accepted log as a multiset including file, line, column and message: zero added or removed. Separate Make go-vet was not reached. This is an inherited lint failure, not a global lint or live-clean claim. The exact comparison is `/tmp/fn77-task3-lint-comparison.json`.

Actual development failures are preserved in the command journal: expected missing Field/generated-reference red checks; constructor-arity and inferred-error-type test mistakes; proof elaboration/name-shadowing and map-attach completeness failures; test pipe/guard diagnostic syntax; one generated fixture attempt while the failed preceding compile had removed Field.olean; and two mutation-harness setup/assertion failures. All were corrected before final gates. The first mutation harness used an isolated Umpire import root that hid Value.olean; the second correctly detected a mutation but expected the wrong diagnostic marker. The third completed all four controls. No timeout caused a command restart, no editor process was killed, and no permission widening occurred.

### Trust and preservation

The genuine pre-edit captures contain 4,735 core/Temporal declarations and 390 fixture declarations. Final captures contain 5,126 and 394 respectively. All existing complete declaration statements and raw transitive axiom arrays are identical; no removal or axiom growth occurred. Every new declaration is mapped to a captured existing analogue whose assumptions contain its inventory. New assumptions are limited to the inherited `propext`, `Quot.sound`, `Classical.choice` boundary; there are no custom/compiler-trust axioms or placeholders. These captures contain complete declaration statements/types, not definition bodies. Strict parsing verifies every OWNER/TRUST/DECLARATION record, matching selection/completion counts and no elisions. See `/tmp/fn77-task3-trust-comparison.json` and `/tmp/fn77-task3-trust-parser-completeness.json`, with the four raw logs and parsed JSON files.

`/tmp/fn77-task3-baseline` records pre-edit HEAD/index/staged diff, 7,684 source hashes, 50 original source-byte copies and the two new-module absences. All originals verify. Every changed existing file has captured original bytes. All unrelated source, original comments, HEAD and staged entries are preserved. The task1 immutable baseline verifies (715 internal files plus four external raw captures); the four task2 dependency artifacts are unchanged. Later comparisons are explicitly post-edit verifications, never reconstructed pre-edit executions.

Removing only the new generated import and two access wrappers reproduces original Temporal/basic API facade bytes exactly; the empty-service facade changes only by that import. All descriptor metadata, complete schema closure/identity, input.pb, original fixture protos and descriptive Proto/Types are byte-identical to the task3 baseline. `/tmp/fn77-task3-preservation.json` records these checks. The nine-path task-only patch was applied to isolated captured original files and every resulting final hash verified.

### Handoff and scope

Task4 retains the existing owner/witness-indexed value contract; this task adds no Action/domain semantics. Task5 can consume typed cursors and their scalar/cardinality/presence denotations, with explicit establishment before dependent reads; it owns Boolean branch refinement and Property expressions. Task7 owns portable Go alignment, including task2's OPEN-enum mismatch; no Go runtime semantics changed here. Task8 owns serialization/lowering of these structural coordinates and whole-Case coverage correspondence. None of task3's field/access semantics is deferred to those owners. Whole-spec model/lint/regression/live/load gates remain task11 by scope; no live or full cross-language qualification is claimed.

Frozen handover: `/tmp/fn77-task3-evidence.json`, `/tmp/fn77-task3-changed-paths.txt`, `/tmp/fn77-task3-frozen-hashes.json`, `/tmp/fn77-task3.patch`. Conductor owns review, lifecycle and Git. No source edits or jobs follow handover.

stage: impl-review - ran; SHIP with no findings (model: gpt-6-astra at medium)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: GOMAXPROCS=1 GOFLAGS=-p=1 GOCACHE=/private/tmp/fn77-task2-go-cache GOCACHEPROG="python3 /private/tmp/fn77-task2-cache.py" mise exec -- go test -count=1 -tags test_dep ./tools/umpire/cmd/umpire-gen-lean-api ./common/testing/testpilot/internal/ir, GOMAXPROCS=1 GOFLAGS=-p=1 GOCACHE=/private/tmp/fn77-task2-go-cache GOCACHEPROG="python3 /private/tmp/fn77-task2-cache.py" mise exec -- make umpire-check-lean-api, mise exec -- lake build Umpire.Property.Tests Testpilot.Tests Umpire.Value.FieldTests UmpireTests, GOMAXPROCS=1 GOFLAGS=-p=1 GOCACHE=/private/tmp/fn77-task2-go-cache GOCACHEPROG="python3 /private/tmp/fn77-task2-cache.py" mise exec -- make lint-code GOLANGCI_LINT_FIX=false
- PRs: