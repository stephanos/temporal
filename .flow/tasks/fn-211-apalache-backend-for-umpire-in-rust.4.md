---
satisfies: [R4, R5]
---
# fn-211-apalache-backend-for-umpire-in-rust.4 Apalache consumer in the fixture runner; Kettle agreement golden

## Description
Owner deferred fn-211 on 2026-10-22: "defer fn-211: Apalache work and move on for now". Do not execute until reactivated. Unfinished consumer source, tests, guest and handoff are preserved at local branch `work/fn2114-deferred`, commit `fee5fc56ae`; its `.flow/artifacts/fn-211.4-deferred/summary.md` records completed checks and missing proof. Raw diagnostics remain in `.flow/tmp/fn-211.4` and ignored `model/build/apalache`. This is an unfinished checkpoint, not task completion; integrate and reanchor it only after reactivation.

Integrate the completed bounded runner and typed replay as an Apalache fixture consumer under parent R4/R5. Task .3 is done at committed closure 023c6c7bc13e2cc65fc880a30c7db8a99fadade2, with final gate e17871d6 independently honored. Reanchor its API before editing; reuse the existing Consumer registry, logical-file ownership, skipped-artifact preservation, Bound/native exploration and generated fixture dispatch instead of creating a second runner.

**Size:** L. **Dependencies:** .3 done and its closure committed on umpire before admission. **Touches:** [model/umpire/testing/**, model/umpire/checker/**, model/umpire/tests/**, model/umpire/backends/apalache/**, model/Cargo.toml, model/Cargo.lock, Makefile]
Checker changes are restricted to reusing its existing exploration with an already checked retained catalog when needed for exact binding/exploration/replay identity; preserve ordinary checker semantics and avoid a second exploration engine. No canonical production Model/protocol/Go changes, new Model authoring syntax, third-party libraries or publication. New small test fixtures may use the existing authoring surface where needed for consumer failure controls. .5 owns symbolic Worlds and inverse World construction.

### Approach

1. Baseline focused registry/ownership and existing decoder/replay controls. Register Apalache ownership unconditionally for logical apalache.txt and apalache/paths/*.txt; retain the native checker's paths/*.txt. Use the repository's existing snapshot/logical-file convention rather than assuming the logical filename is a raw physical file. Select execution by the logical apalache.txt marker; backend paths without their marker are stale/orphaned output and must fail validation.
2. Extend the existing generated fixture hooks/typed Bound route to emit the explicit safety projection for the same concrete selected World, retained complete starts and checked catalog. Retain the same checked catalog for source binding, native exploration and replay rather than re-enumerating it; reuse the existing checker algorithm through a narrow retained-catalog entry point if necessary. Reuse the completed native exploration and .2 checked depth+1 bound, refusing cut/overflow. Do not compare another World's graph or look up states/actions through formatted strings. Keep source emission independent of native edges and preserve all TLC/native artifacts.
3. Execute one bounded tool check per declared safety property in source order, including invariants, postconditions and dedicated arithmetic/step checks as exposed by the actual completed API. No progress/liveness through Apalache. Decode/replay every violation with .3, recheck its specific claim, and compare the bounded answer to the native claim result at that same bound. A Kettle !beeping violation is an agreeing negative control, not a tool failure. Render deterministic property/name/bound/verdict/agreement and typed replay path outputs; retain raw diagnostics separately, never timestamps/absolute paths in goldens.
4. In default mode, validate ownership and preserve every committed Apalache payload without download, JVM launch or execution/agreement claims. Required mode executes and compares every selected output, including committed violation paths. No skipped route may erase backend payloads, claim a pass or hide unowned/duplicate output. Register ownership before execution on either route. Preserve checker paths and reject duplicate ownership, stale paths and extra backend files deliberately.
5. Required NotRun, timeout, unsupported source, malformed/witness-rejected trace, incomplete native exploration or disagreement fails the fixture independently of snapshot equality. Give useful deterministic disagreement diagnostics; golden update cannot turn any such failure into success. Test both normal compare and update modes without blessing failed artifacts.
6. Add Kettle's main agreement golden and replayed !beeping path. Add selected concrete-World coverage for parameterized_world and action_inputs with their exact starts/input identities. Include invariant, failing postcondition and after-filtered negative consumer controls using small existing-surface fixtures or existing admitted hooks; prove unrelated actions do not trigger an after-filter. Preserve prior source/native/TLC goldens byte for byte; new backend outputs are exclusively owned.
7. Exercise a committed violating trace through default skip and required comparison. Use fake/spied tools for zero launch/download default, missing/wrong tool, malformed output and failure propagation; required actual pinned-tool checks prove complete Kettle bounded agreement at the native-derived bound (currently63), typed replay, postconditions and inherited concrete World/input fixtures. Do not shorten the bound to obtain a passing smoke proof.

### Verification

Focused registry/ownership/replay suites in default and required modes, strict changed-crate clippy and actual selected fixture checks. One final timeout3600 env UMPIRE_BACKENDS=require make model-time after implementation review and root's correctness/standards audits settle; it subsumes duplicate default/required full checks and timing. Retain all existing gate stages, actual timings/soft misses and source-scoped guest/live proof. Relevant checker/DSL/runtime production-source changes require a canonical guest rebuild and fresh actual slice/gaps proof. Exact unchanged runtime/guest closure permits explicit proof reuse; no synthetic execution/receipt or new cross-OS claim.
## Acceptance
- [ ] Apalache owns logical apalache.txt and apalache/paths/*.txt exclusively; native checker paths remain separate. Ownership is registered and validated before either execution or skip, and orphan/duplicate/stale/unowned outputs fail.
- [ ] Default mode preserves committed Apalache artifacts with zero tool download/launch and explicit skip semantics; required mode executes and compares every selected main/path output.
- [ ] Kettle invariants and postconditions agree at the completed native-derived bound, and !beeping yields a typed replayed violation path. Concrete parameterized World/input and after-filter controls also agree.
- [ ] Every required refusal, incomplete result, malformed/non-replayable witness or disagreement fails independently of golden equality/update; failed execution cannot be blessed.
- [ ] Deterministic new backend goldens contain no timings/absolute paths, previous source/native/TLC artifacts remain exact, and focused tests/lint/actual tool controls/review/audits/final required gate pass with honest scoped evidence.
## Done summary
Blocked:
Owner deferred fn-211: "defer fn-211: Apalache work and move on for now". Unfinished work retained at work/fn2114-deferred, fee5fc56ae. This is a scheduling deferral, not a technical blocker or completion.
## Evidence
- Commits:
- Tests:
- PRs:
