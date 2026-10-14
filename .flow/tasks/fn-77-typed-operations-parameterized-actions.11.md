---
satisfies: [R2, R5, R8, R9]
---
# fn-77-typed-operations-parameterized-actions.11 Verify complete typed-operation compatibility and bounded qualification

## Description
Verify complete typed-operation compatibility and bounded qualification for the referenced parent requirements.

**Size:** M
**Files:** model/README.md; model/ARCHITECTURE.md; model/Umpire/ARCHITECTURE.md; model/Umpire/**/Tests*; model/Temporal/Feature/Nexus3/**; tools/umpire/cmd/umpire-gen-lean-api/**; common/testing/testpilot/**/*_test.go; tests/testpilot_async_nexus_case_test.go
**Touches:** [model/README.md, model/ARCHITECTURE.md, model/Umpire/ARCHITECTURE.md, model/Umpire/**/Tests*, model/Temporal/Feature/Nexus3/**, tools/umpire/cmd/umpire-gen-lean-api/**, common/testing/testpilot/**/*_test.go, tests/testpilot_async_nexus_case_test.go]

### Approach
- Compare final source/fixture bytes and complete raw transitive axiom sets to task1 original captures using explicit renamed/generated declaration mapping; missing/truncated evidence fails qualification.
- Run schema mutation and deterministic complete regeneration/staleness controls, preserving old canonical fixtures when semantics are unchanged; verify newly versioned identities change only on declared semantic edits.
- Exercise tenfold finite request variation, payload/collection sizes and live captures plus independent sequential/concurrent Runs; assert bounded atomic rejection and unchanged finite-versus-runtime claim scope.
- Run both real examples and all affected existing regression gates with named outcomes; retain environmental/inherited failures explicitly instead of treating skipped checks as passing.
- Update checked authoring walkthroughs and supported-form documentation, including float/recursive/enum policies, independent clauses/coverage, symbolic binding and deferred cancellation. Keep raw schema metadata distinct from exact executable values.

### Investigation targets
**Required:**
- model/README.md:90 — authoring walkthrough owner.
- model/Umpire/ARCHITECTURE.md:183 — lowering ownership.
- Makefile:1019 — complete API generation; line1035 explicit fixture rewrite.
- Makefile:1164 — current complete regression recipe.
- common/testing/testpilot/scoped_facade_test.go:189 — isolation/load pattern.

### Quick commands
`go test -count=1 -tags test_dep ./tools/umpire/cmd/umpire-gen-lean-api ./common/testing/testpilot/... ./tests/testcore/testpilot/...`
`go test -race -tags test_dep ./common/testing/testpilot/...`
`go test -json -count=1 -tags 'test_dep integration' ./tests -run '^(TestTestpilotTypedUnaryCase|TestTestpilotTypedNexusOperationsCase|TestTestpilotAsyncNexusCase|TestTestpilotAsyncNexusCaseMissingRemoteEndpoint)$' > /tmp/fn77-task11-live.jsonl && python3 -c 'import json,sys; e=[json.loads(x) for x in open(sys.argv[1])]; names=sys.argv[2:]; assert all(any(v.get("Test")==n and v.get("Action")=="run" for v in e) and any(v.get("Test")==n and v.get("Action")=="pass" for v in e) for n in names); assert not any(v.get("Action") in ("skip","fail") for v in e)' /tmp/fn77-task11-live.jsonl TestTestpilotTypedUnaryCase TestTestpilotTypedNexusOperationsCase TestTestpilotAsyncNexusCase TestTestpilotAsyncNexusCaseMissingRemoteEndpoint`
`make umpire-build-model`
`make lint-model`
`make umpire-check-regression`
`make lint-code GOLANGCI_LINT_FIX=false`

### Execution constraints
Read the full parent spec. Preserve existing comments and unrelated dirty source. New paths listed here are proposed owners; reuse an established equivalent before creating one. Run Lean jobs serially; fn76 lint recovery is complete. Preserve fn75 semantic import isolation and fn78 obligation/evidence authority. No operation cancellation, new dependency/toolchain, or implicit fixture promotion. Do not stage, commit, or push; the user owns commits. Capture task-local before/after trust for changed load-bearing declarations; task11 additionally compares to the original task1 substrate.

### Environment note
`go test` in this checkout needs `CC=/usr/bin/cc`: mise's lean4 clang shadows the toolchain and cgo
fails with `stddef.h not found`. Every Go gate on this branch needs it.

### Follow-ups inherited from task .10's review
Two P3s were left open by .10 and land here if its acceptance reaches them:
- `TypedNexusProfile` and the live-binding setup duplicate their async-nexus counterparts and want
  one shared builder.
- The Link admits any *declared* identity, because a correlation operand cannot name the scope key.
  A crossed-but-declared identity is separated by the field requirement and by the per-identity
  runtime rules instead; both answers are pinned by `#guard`.

Task .9's `violated`-state P3 below was offered to .10 and declined there, so it is this task's.

### Follow-up inherited from task .9
The runtime rule carries only `pending` and `satisfied`, so a started event recording a different
workflow type leaves the rule inconclusive where the model Property distinguishes violated from
inconclusive. Adding a `violated` state plus a tampered-fixture live assertion is a real R6
improvement; task .9's review raised it as a non-blocking P3 and assigned it here or to .10.

### Known findings introduced by this spec
RESOLVED during this task. The baseline is **169**, all in generated
`Temporal/API/{Types,Proto}.lean`; `Umpire.Lint` itself is clean. Two earlier figures in this note
were both wrong: "exactly 2" omitted the generated-API findings, and "171" then counted this spec's
own two as if they were permanent. They were neither inherited nor permanent - both are fixed:
`CheckedRpc` carries `set_option genInjectivity false` (its other two fields are proofs, so simp
closed the generated lemma, and nothing consumed it) and `PropertyFieldProjection` lost a
`deriving Repr` that had no consumer. Retained for history, the two findings were:
- `unusedArguments` on `Umpire.instReprPropertyFieldProjection` (`model/Umpire/Property/Evaluation.lean:470`)
  — from task .5's `deriving Repr`; no consumer of that instance exists.
- `simpNF` on `Umpire.Operation.CheckedRpc.mk.injEq` (`model/Umpire/Operation.lean:58`) — from task .1;
  the structure's only non-proof field makes the generated lemma simp-provable.

## Acceptance
- [ ] Original before/after raw trust inventories have no unapproved expansion or missing entries; unchanged canonical/fixture/identity bytes are preserved and new schema/parameter changes follow checked versioning.
- [ ] Tenfold variation/payload/collection/capture tests reject within declared bounds without cross-Run leaks, resets or broader exhaustive claims.
- [ ] Both real Driver examples and affected generator/semantic/codec/evaluator/functional/staleness gates have exact terminal receipts; any baseline failure is identified and compared rather than called clean.
- [ ] Documentation and executable walkthroughs report actual exact/unsupported forms, both examples, source clause coverage and finite/runtime scope; Known Gaps waive no requested requirement.

## Done summary
The whole typed-operation surface is compared to the substrate task .1 froze before the first edit,
and both authored examples are qualified on the real Driver with every affected gate given an exact
terminal receipt.

### Original trust and byte comparison

Five raw transitive axiom inventories were recaptured at HEAD with task .1's own capture drivers and
parser and compared to its originals: the semantic selection (9,481 -> 10,608), the scoped
supplement (12,611 -> 13,624), the generated API surface (407 -> 3,612), the generated types (2,909
-> 3,597), and the operation selection task .1 left behind (1,336 -> 1,337). The comparison is
`/tmp/fn77-task11-trust-comparison.json`.

**No unapproved expansion.** Every retained declaration that gained axioms gained only `propext`,
`Classical.choice` and `Quot.sound`. No `sorry`, no custom axiom, and no `native_decide` trust edge
is introduced: the two `native_decide` axioms reachable from `Umpire.Query.Tests.JointConflicts`
belong to the byte-identical `Umpire.Planning.Tests.Fixtures`, and only their compiler-assigned
ordinals shifted, so the comparison normalizes that suffix rather than reporting a false growth.

**No missing entry is unexplained.** 113 disappearances are Lean-generated auxiliaries whose names
embed hygiene counters; the 15 declared ones all carry an explicit mapping — twelve private Case
helpers that moved from `Temporal.Feature.Nexus3.Testpilot` to the shared
`Temporal.Testpilot.CaseSupport`, `CheckedPropertyPredicateInput.input` which moved from
`Umpire.Property.Check` to `Umpire.Property.Evaluation` with the typed field operands, and
`CheckedRpc.mk.inj`/`injEq`, which are no longer generated (below).

**Bytes.** 204 of the 208 non-tracker fixtures task .1 froze are byte-identical at HEAD, including
every case-runtime-conformance artifact and the async-nexus, get-system-info and synthetic Cases.
The four that changed are this spec's own generator-fixture surface, each rewritten through its
Makefile owner (`/tmp/fn77-task11-byte-preservation.json`).

### The generated API surface was stale, and is not any more

Task .7 added `ScopedCaptureDeclaration` and `ScopedCaptureRef` to `contract.proto` without
regenerating `model/Temporal/API{,/Types}.lean`. R8 requires a schema change to trigger
regeneration, so the owned surface was regenerated through `make umpire-gen-lean-api`; two
consecutive runs produce identical bytes. Because every method's `RpcSchema` carries the generator's
whole descriptor input closure, this moved both typed Cases' Behavior Fingerprints — which is what
task .12's version-2 closure digest exists to do, and nothing outside those two regenerated
artifacts pins the old values. **Follow-up, not built here:** that shared `schemaInputs` list makes
one operation's identity sensitive to any proto change anywhere in the generator's input closure.
It is conservative rather than unsound, but a per-method input closure would be tighter.

Controls, each red then green: a mutated generator fixture fails `TestBasicFixture`; a mutated
`typed-unary-case.json` fails `make umpire-check-case-runtime-conformance`.

### Bounded qualification

`Tests/TypedUnary.lean` now admits ten request variations as exactly ten distinct Action instances
under the same `.sampled` claim two carried, keeps the authored Case at its own two samples, and
separates the three answers a load can get: inside the declared runtime bounds, `outOfScope` past
them, and a value-layer resource rejection when the checker's own collection ceiling is the one
exhausted. Rejection is atomic — nine admissible variations plus one the ceiling refuses yield no
domain at all, and so does a repeated variation.

`tests/testcore/testpilot/typed_unary_artifact_test.go` drives the unchanged typed unary Case bytes
through the public `Prepare`/`Run` facade on a scripted driver: ten histories differing only in the
workflow type the started event records (one satisfied, nine violated), two histories that never
establish the field (inconclusive, not violated), four load cases that probe exactly one declared
bound each — at and one past the Program's path-fanout ceiling, and past its response-byte budget,
each asserting the bound it is not probing stays unexceeded — and twenty sequential and concurrent
Runs across two drivers, every Run answering from its own evidence with its own run id.

### The inherited follow-ups

- **Task .9's `violated` state is built.** The typed unary rule now carries a `violated` terminal
  state and a `reject-recorded-workflow-type` transition, so an event that establishes the compared
  field and disagrees is a violation rather than an absence. It is sound there because the oneof
  selector admits only the one started event a history carries. The ten-variation test above is the
  tampered-fixture assertion.
- **Task .10's crossed-completion P3 is a recorded Known Gap, not a rule.** A completed Nexus event
  carries no operation identity, so a reference to another scheduled event is indistinguishable from
  the sibling operation's own completion; separating them needs a correlation condition ACT-4 makes
  an Implementation Link obligation. The Case now says so in its provenance.
- **Task .10's duplication P3 is built.** `tests/testcore/testpilot/profile.go` gives the three Case
  Profiles one workflow-service role builder and one assembler, and `tests/testpilot_live_case_test.go`
  gives the three live tests one binding builder that registers every resource cleanup under the one
  timeout its caller chose.

### Gates, and the baselines they are measured against

`umpire-build-model`, `umpire-check-lean-api`, `umpire-check-regression`, the focused and `-race` Go
suites, `go vet ./...` and the four-test live gate are all exit 0. Two are red, both identified
rather than called clean:

- `make lint-model` is at **169** findings, all `simpNF` and `unusedArguments` on generated
  `Temporal/API/{Types,Proto}.lean` declarations, and `Umpire.Lint` is clean. The recorded baseline
  was 171: the two extra were this spec's own and are both resolved here. `CheckedRpc` is declared
  under `set_option genInjectivity false` (both its other fields are proofs, so the lemma reduced to
  equality of the one data field and simp closed it; nothing used it), and `PropertyFieldProjection`
  lost a `deriving Repr` no consumer had.
- `make lint-code GOLANGCI_LINT_FIX=false` is at exactly the inherited **1,284** issues with the
  same per-linter breakdown, so this task added none. Its second step, `go vet ./...`, which make
  never reaches while golangci-lint is red, was run separately and is clean at zero findings.

`make umpire-check-regression` also caught a real violation this spec introduced: task .12 wrote
"Temporal.API" into a comment in `model/Umpire/Operation/Canonical.lean`, which fn-75's semantic
import isolation forbids in reusable Umpire artifacts. The comment now names the method without the
namespace and the gate is green.

### Documentation

`model/README.md` gains a "Typed operation authoring" walkthrough: how a generated declaration is
referenced and what rejects, the separately declared finite and runtime claims, the exact supported
value forms and the unsupported ones with the diagnostic each gives (floating point, recursion
depth, integer range, closed enums), the two authored examples, symbolic environment binding,
deferred cancellation, and the two Known Gaps. `model/Umpire/ARCHITECTURE.md` gains the lowering and
identity ownership; `model/ARCHITECTURE.md` points at both.

### Concurrent local edits this run swept in that this task did not author

`.flow/tasks/fn-77-...10.md` and `.11.md` were already dirty at Phase 1, and
`.flow/specs/fn-77-....json` is the review-round receipt the impl-review dispatch wrote.

stage: impl-review - ran [round 1 SHIP (claude/claude-fable-5-1, high)]; three P3 findings, all
fixed in f8bcaf5d with the affected gates re-run. Investigating the collection-budget one showed the
bound that actually rejects is the Program's path-fanout ceiling, not the emitted-event limit the
reviewer assumed, so the case now pins that ceiling exactly.
## Evidence
- Commits: 261c0af12665c1a0f6aa9cc551abcb95a2eb3d4f, f8bcaf5deb4d51143236261410a8a690433c95ee
- Tests: make umpire-build-model => exit 0; /tmp/fn77-11-final-build.log, make umpire-check-lean-api => exit 0; /tmp/fn77-11-checkapi.log, make umpire-check-regression => exit 0; /tmp/fn77-11-regression3.log (live failure identities match the inherited exact set), make lint-model => exit 2 at 169 inherited generated findings (Temporal.API.Types simpNF + Temporal.API.Proto unusedArguments); Umpire.Lint clean; /tmp/fn77-11-final-lintmodel2.log, make lint-code GOLANGCI_LINT_FIX=false => exit 2 at the inherited 1284 issues (errcheck 220, exhaustive 5, forbidigo 209, goimports 1, govet 5, revive 732, staticcheck 111, testifylint 1); /tmp/fn77-11-lintcode2.log, go vet -tags disable_grpc_modules,test_dep -vettool=.bin/errortype -style-check=false ./... => exit 0, zero findings; /tmp/fn77-11-govet2.log, go test -count=1 -tags test_dep ./tools/umpire/cmd/umpire-gen-lean-api ./common/testing/testpilot/... ./tests/testcore/testpilot/... => exit 0; /tmp/fn77-11-q1c.log, go test -race -tags test_dep ./common/testing/testpilot/... => exit 0; /tmp/fn77-11-race2.log, go test -json -count=1 -tags 'test_dep integration' ./tests -run '^(TestTestpilotTypedUnaryCase|TestTestpilotTypedNexusOperationsCase|TestTestpilotAsyncNexusCase|TestTestpilotAsyncNexusCaseMissingRemoteEndpoint)$' => exit 0, all four run and pass, none skipped; /tmp/fn77-task11-live.jsonl, raw transitive axiom comparison against the task .1 original substrate => /tmp/fn77-task11-trust-comparison.json (5 captures, 0 unexplained missing, added axioms only propext/Classical.choice/Quot.sound), fixture and canonical byte preservation against task .1's frozen hashes => /tmp/fn77-task11-byte-preservation.json (204 of 208 preserved), deterministic regeneration control: make umpire-gen-lean-api twice => identical sha256 for model/Temporal/API.lean and model/Temporal/API/Types.lean, schema mutation control: mutated generator fixture => TestBasicFixture exit 1, restored => exit 0; /tmp/fn77-11-mutant1.log, staleness mutation control: mutated typed-unary-case.json => make umpire-check-case-runtime-conformance exit 2, restored => exit 0; /tmp/fn77-11-mutant2.log
- PRs:

stage: plan-sync - skipped(config: planSync.enabled != true)
