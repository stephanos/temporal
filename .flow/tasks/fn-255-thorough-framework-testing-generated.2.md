---
satisfies: [R1, R7]
---
# fn-255-thorough-framework-testing-generated.2 Generated differential harness with shrinking and fixture pinning

Touches: [model/umpire/dev/generate/**, model/umpire/dev/testing/**, model/umpire/dev/fixtures/fixtures/generated_*/**, model/umpire/backends/native/tests/**, model/umpire/backends/tlc/**, model/umpire/backends/tla/**, model/umpire/dev/gate/budgets.toml, model/umpire/Cargo.toml, model/umpire/Cargo.lock]

## Description
Build the source harness chosen by .1: complete graph comparison, independent per-claim agreement including legitimate refutations, bounded shrinking, and fixture pinning (R1, R7). The optional tree-only layer was declined; source generation must cover the macro and front end.

**Size:** M
**Files:** `model/umpire/dev/generate`, shared comparison helpers, generated regression fixtures, narrowly required TLC/emitter seams, and the authoritative budget table.

### Prerequisites and ownership
- Consume committed fn-254.12 and its single authoritative table at `model/umpire/dev/gate/budgets.toml` before execution or task closure. Add the harness's reviewed bounds there; do not create another provisional budget table, environment-default allocation or independently maintained hard-cap source.
- The coordinator serializes .2/.3 TLC seams, framework manifest/lock and shared development-harness changes. Acquire exact exclusive file/hunk ownership before editing overlapping files; preserve current native stage DAG, cancellation, resource admission and count authorities.

### Approach
- Implement the admitted design in `model/design/generated-differential.md`, with bounded optional values, World-selected assumptions, action inputs, checked arithmetic, `reject!`/`stay!` and facts. Source families remain within a proven finite space.
- Compare complete start sets, reachable states and canonical labeled transitions, including action identity, successor, outcome and ordered facts. This complete-transition obligation is separate from per-claim verdict and counterexample checks; counterexample replay alone cannot satisfy it. Preserve independent compiled execution, interpreter evaluation and source-lifted TLC emission.
- Use Verified-only same-schema batching for claims known valid by construction. Include at least one required-mode generated source family with a legitimately Refuted claim. Check such claims individually with TLC and compare each backend's verdict under the same authored claim identity. A legitimate Refuted result is a successful checker result; an unsupported refusal, error, timeout or incomplete exploration is a failure, including matching refusals.
- Preserve each refuted claim's independent TLC counterexample, validate it and replay its labeled steps through compiled and interpreted execution for that claim. Compare verdict and trace validity, without requiring independent checkers to choose the same trace or order. A batch-level refutation cannot stand in for per-claim results.
- Freeze seed, printed source and selected-case/claim manifests. Default mode records compiled/interpreted checks and TLC skipped explicitly; required mode completes every selected backend and reports its actual launches/configurations.
- On disagreement print the seed and shrunk source. Shrinking must preserve the same mismatch and claim identity, with every candidate successfully admitted and checked; refusal or a new error is not detection. Pin every real finding under `dev/fixtures/fixtures/generated_<n>/` with reproducible goldens.
- Seeded controls cover swapped emission comparison, dropped interpreter guard, outcome/fact-only changes preserving states/verdicts, and a change making only one backend refute a claim. Every mismatch is detected and shrunk. Matching backend refusals fail.

### Bounds and measured basis
- Register .1's measured proposals in fn-254.12's table for review rather than treating them as admitted: at most 36 states, 144 emitted edges, shortest depth 35, two starts, 290 augmented assignments per Model and 580 per two-Model batch; one default source Model with no TLC; two required source Models in one same-schema Verified batch; three shrink candidates.
- The measured Verified batch used two model JVMs plus one discovery JVM. Legitimately Refuted families require additional individual claim checks; admit their actual per-case and per-batch launch budgets, including discovery and every claim check, before execution. Do not invent a larger allocation or hide these launches behind the original three-JVM proposal. Cut selected work if necessary to fit the admitted share.
- .1 proposes default differential share 11 s (10 s compilation plus 1 s native runtime), required share 45 s (30 s compilation plus 15 s runtime), 10 s per model JVM, 8 MiB output and 16 MiB dumps. These remain measured proposals until authoritative admission. Keep source-size, case, claim, batch, shrink and launch limits, per-case deadlines and failure reduction within the owning stage's admitted wall time. Override proptest defaults explicitly.
- Over-cap Model, assignment, case/batch and TLC-launch controls fail by stage name, including an extra discovery or individual claim launch. Timeouts refuse agreement. Full campaigns stay scheduled-only; .7 owns final tier activation and combined 30 s/120 s share measurement.

### Investigation targets
- `model/design/generated-differential.md` and its retained .1 measurements.
- `model/umpire/dev/testing/src/backend.rs`, `interpreted.rs` and fixture layout.
- Current TLC/emitter APIs, native lifecycle/count receipts and fn-254.12 admission contract.

## Acceptance
- [ ] Required-mode source cases agree on complete starts, states, labeled edges, outcomes and facts; per-claim checks independently agree on Verified and legitimately Refuted results. Default mode records backend selection and TLC skip.
- [ ] At least one required-mode generated source family is legitimately Refuted; its independently checked counterexample replays in compiled and interpreted execution under the exact claim identity, without imposing identical trace ordering.
- [ ] Swapped-comparison, dropped-guard, outcome/fact-only and single-backend-refutes controls are caught and shrunk; matching refusals, errors and timeouts fail.
- [ ] Every real disagreement is fixed or filed and pinned as a fixture.
- [ ] One authoritative fn-254.12 table supplies admitted sizes, cases, claims, batches, shrink limits, timeouts and TLC launches per case and batch, including discovery and individual checks. Over-cap controls fail with stage identity.

## Done summary
Blocked:
Execution and task closure wait for the committed fn-254-fast-validation-gates-and-a-real-inner.12 closure in umpire and admission of this task's work/time limits to its single authoritative budget table. Disjoint source preparation does not satisfy this gate. Flow task dependencies only accept same-spec IDs; after verifying that closure and admitted entries, the conductor may reset this block and admit the task normally.
## Evidence
- Commits:
- Tests:
- PRs:
