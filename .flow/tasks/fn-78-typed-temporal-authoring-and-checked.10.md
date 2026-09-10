---
satisfies: [R6, R7, R8, R9]
---
# fn-78-typed-temporal-authoring-and-checked.10 Qualify generic scoped authoring without cancellation

## Description
Deliver reusable bounded temporal syntax and non-cancellation qualification formerly bundled in task 8. Resolve typed trigger/response, correlation key, clock, natural bound, endpoint, and source-local diagnostics through the same checked clauses. Use generic multi-operation fixtures for scoped Contract parity, and preserve the existing Nexus success Prepare/Run integration. Update model/README.md, model/ARCHITECTURE.md, model/Umpire/ARCHITECTURE.md and Nexus/Success/Integration.md to describe the generic delivery and explicitly deferred cancellation. Do not implement a cancellation Target, evidence adapter, capability, or Case.

Repair stale regression-gate build-file references exposed by required validation against the current Lean build configuration.

**Size:** L
**Touches:** [Makefile, common/testing/testpilot/scoped_facade_test.go, common/testing/testpilot/testdata/case-runtime-conformance/scoped.json, model/Umpire/Case/Tests/**, model/Umpire/Property/**, model/Temporal/Feature/Nexus/Success/**, model/Temporal/Tool/Testpilot.lean, tests/testcore/testpilot/**, tests/testpilot_async_nexus_case_test.go, model/README.md, model/ARCHITECTURE.md, model/Umpire/ARCHITECTURE.md]

## Acceptance
- [ ] Readable bounded temporal notation and typed constructors have identical checked canonical meaning and fingerprints.
- [ ] Compile-failure tests reject wrong contexts, references, keys, clocks, scopes, bounds, raw evidence, command effects, and unsupported formulas at the author expression.
- [ ] Authors supply semantic choices without serialization/proof/monitor plumbing.
- [ ] A deterministic admitted non-cancellation Case exercises scoped obligations through the public Prepare/Run facade; existing Nexus success/rejection integration remains green.
- [ ] Generic violating, incomplete, wrong-correlation, stopped/lost execution, and cleanup-failure fixtures preserve verdict/disposition and prior proof.
- [ ] Repeated/concurrent Runs and tenfold evidence/obligation loads demonstrate isolation and bounded failure.
- [ ] Required model builds/lints, regression/staleness gates, focused Go tests with test_dep, and existing success integration with test_dep integration pass or record verified inherited failures.
- [ ] Architecture and authoring documentation matches the generic delivery and retains cancellation as explicitly deferred to fn-79.


## Done summary
Implemented readable bounded temporal notation over the existing checked scoped clauses, owner-aware source-local diagnostics, and executable non-cancellation scoped Case qualification through public Prepare/Run. Architecture and authoring documentation describes the generic delivery and explicitly defers Nexus operation cancellation to fn-79.

Independent review: SHIP, no findings or uncovered R-IDs; receipt /tmp/fn78-task10-impl-review.json. Commits=[]; all source changes remain uncommitted for the user.

Baseline: green via task7 conductor handoff; inherited full Go lint red1284. Final normalized comparison against task7 has introduced=[] and removed=[].

Coverage:
- R6: correlated_response% lowers directly to PropertyCorrelatedClause, preserving checked canonical JSON and fingerprints. All semantic choices remain typed and explicit. Contextual keywords preserve existing identifiers. TemporalAuthoring compile guards cover wrong references/context, scope/key, clocks/endpoints, natural/overflow bounds, unsupported formulas, raw Projection.Event, and Testpilot Instruction effects. The empty-key guard pins clause owner, author path, line62, and columns78–105; related-reference precedence is preserved before owner fallback.
- R7: the existing checked scoped corpus now emits an additive runnableCase, whose ordinary RPC response projection supplies typed evidence through the real scheduler/recorder/facade. Twelve complete deterministic scenarios cover zero/inclusive/late deadlines, repeated triggers, interleaved operations, self-loops, deliberate closure, causal buffering, duplicate/poll stuttering. Controlled wrong-correlation, lost/stopped source, and cleanup-after-violation fixtures assert verdict/disposition and exact prior proof support. Existing live Nexus success and missing-remote-endpoint tests have explicit run/pass receipts, no skip.
- R8: two rounds of ten concurrent mixed satisfied/inconclusive Runs have distinct IDs and isolated immutable verdict state. Two-versus-twenty evidence/obligation/buffer/work loads fail boundedly and inconclusively under the declared ceilings; insufficient capture count or bytes rejects during Prepare. Lost/stopped execution under a final trace ending does not invent a semantic deadline. The fixture has one attempt per instruction and adds no implicit redispatch.
- R9: all14 original scoped cases, evidence sequences and expected answers remain unchanged, and legacy functional/non-scoped artifacts remain byte-for-byte unchanged. Full model/regression/staleness, affected Go, race, and integration gates pass. No new runtime semantics, wire schema, custom/compiler-trust axiom, native_decide, or dependency was introduced. Existing scoped proof audits remain covered by model build/lint.

Verification: exact commands, logs and terminal exit files are in /tmp/fn78-task10-evidence.json. Full model build513 jobs passed. The regression live gate accepted its exact inherited nine failure identities (TestUmpire2 suite and seven children, plus the Umpire3 process-crash test); it did not report a fully green historical live suite. The targeted Nexus success/rejection tests passed without skips. Full make lint-code remains inherited red1284 and therefore its subsequent go-vet recipe step is not reached. The one introduced revive finding was fixed; no unrelated lint debt was changed.

A task-caused full-build failure exposed the new eventually token reserving an existing local identifier. The notation now uses contextual keywords and bounded-precedence operands; the unchanged GuardedTemporal regression and new authoring tests pass together (109 jobs), followed by the final full513-job build. The red/green diagnostic-owner regression is separately logged. The inherited final regression grep referenced absent model/lakefile.toml; only that path was corrected to lakefile.lean, preserving all prior Makefile changes and comments. The full regression command was rerun after this repair and the mixed-outcome public isolation test. No existing comments or cancellation draft content was reverted.

Review substrate: /tmp/fn78-task10-paths.txt (12 exact paths), /tmp/fn78-task10-scoped.patch (task-only delta), /tmp/fn78-task10-frozen-hashes.json. Pre-task baseline: /tmp/fn78-task10-baseline-files, initial.patch, initial-status.txt and base-commit. The conductor's independent review base is recorded in /tmp/fn78-task10-review-base. No further source edits are planned.

Limits: the generic source is a controlled qualification fixture, not a production Implementation Link or cancellation adapter. Small finite tables and bounded loads establish correctness/fail-closed behavior, not production scalability. Ordinary parameterized authoring uses the existing Except checker path; compile-time property% requires closed inputs.

stage: impl-review - ran (SHIP; model: gpt-6-astra at medium)
stage: plan-sync - skipped(config: planSync.enabled != true)
Tracker sync: n/a (bridge inactive)
## Evidence
- Commits:
- Tests: cd model && mise exec -- lake build Umpire.Property.Tests.TemporalAuthoring Umpire.Property.Tests.GuardedTemporal, LEAN_NUM_THREADS=1 make umpire-build-model, LEAN_NUM_THREADS=1 make lint-model, TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) LEAN_NUM_THREADS=1 make umpire-check-regression, TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) make umpire-gen-case-runtime-conformance, TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) go test -tags test_dep ./common/testing/testpilot/... ./tests/testcore/testpilot/... ./tools/umpire/cmd/umpire-gen-case-runtime-conformance, TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) go test -race -tags test_dep ./common/testing/testpilot -run '^TestScoped', TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) go test -json -count=1 -tags 'test_dep integration' ./tests -run '^TestTestpilotAsyncNexusCase(MissingRemoteEndpoint)?$', TMPDIR=/private/tmp CC=$(xcrun -f clang) SDKROOT=$(xcrun --sdk macosx --show-sdk-path) make lint-code GOLANGCI_LINT_FIX=false, python3 /tmp/fn78-task10-verify-snapshot.py
- PRs: