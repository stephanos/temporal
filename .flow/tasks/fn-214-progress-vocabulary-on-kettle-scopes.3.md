---
satisfies: [R4, R5]
---
# fn-214-progress-vocabulary-on-kettle-scopes.3 Kettle's progress questions as fixtures; fairness and emitter contract in DESIGN.md

## Description
Implement R4 and R5 with supplementary Kettle question fixtures and the narrow query seam they need.
This task consumes .1/.2; it does not introduce the future suite's Space, Run or starts DSL.

**Size:** M
**Files:** checker registered-progress query, shared fixture adapters/questions and current documentation.
**Touches:** [model/umpire/checker/src/explore.rs, model/umpire/checker/src/progress.rs, model/umpire/checker/src/lib.rs, model/umpire/checker/src/refusal.rs, model/umpire/checker/tests/**, model/umpire/tests/fixtures/kettle_progress*/**, model/umpire/tests/support/**, model/umpire/wasm/src/refusal.rs, model/umpire/wasm/tests/dispatch.rs, model/umpire/proto/README.md, model/README.md, model/DESIGN.md]

### Approach

- Add a narrow `Result::verify_progress(&self, world, &registered_claim) -> Result<ClaimResult, Refusal>` operation over an existing exploration. Share one borrowed evaluator with explore-time checking, returning a local result rather than mutating cached claims. Do not re-explore, clone retained rows or duplicate the progress algorithm.
- Retain the original ordered action catalog and registered progress descriptors. Reject changed ordered catalogs before indexing completed-action columns. Resolve a unique original registration by name and execute its canonical callbacks, applying only the supplied explicit assumption/bound overrides. Refuse absent/ambiguous registration. A public descriptor does not establish provenance; do not treat function addresses or names as proof of same-world identity. Document the caller requirement to use the same unchanged world. A fabricated same-name selector must never execute fabricated predicates.
- Use precise refusal diagnostics for catalog/registration misuse. Existing action-admission and budget categories do not describe these errors. Update the exhaustive WASM refusal bridge and native generated-protobuf roundtrip using the existing malformed-request category where accurate; no schema, operation, generated Go or ABI change is needed.
- Import the canonical attribute Kettle by its existing fixture source path. Use a fixture-side generic Machine adapter to select room/cold starts, delegate actions/internal/step to the original world and forward the original registered progress callbacks. Construct starts through the existing public Model fields. Do not duplicate transition or boils predicate bodies, expose private methods by rewriting the sample, or change either canonical sample directory.
- Select the original `boils` registration from `Machine::progress()`. The future `Model::boils.assuming(...)` suite notation is not yet an existing descriptor API; use the registered claim now and preserve the future obligation for fn-213.
- Produce supplementary checker-owned `report.txt` and `paths/*.txt` with a fixture `run()` using existing Files/consumer discovery and deterministic double rendering. Do not add arbitrary output filenames or a second fixture runner.
- Pin query invariants: unchanged cached reports/counts/states/rows, no extra step calls, one combined evaluator call per reached state per query, no evaluator calls with zero starts, registration/catalog refusals, independent override replacement and existing hole/cut/panic/refutation behavior. Validate available metadata even with zero starts; do not fabricate a state.
- Compare the entire two canonical sample directories with the committed task baseline, including source and snapshot metadata. Preserve them byte-for-byte while pinning all R4 questions in separate fixture directories.
- Update README and DESIGN for explicit fairness, numeric authoring bounds, query operation and legacy compatibility. Record that fn-212 emits `WF_vars(A)` only for explicit fair entries. Keep the generic hint text without claiming that fair(heat) repairs Kettle's press/depress cycle; heat is disabled while the element is off.

### Investigation targets

**Required:**
- `model/umpire/checker/src/explore.rs` - retained graph, registration and cached report seam.
- `model/umpire/checker/src/progress.rs` - .2's shared progress algorithm and local result boundary.
- `model/umpire/tests/fixtures/sample_kettle/model.rs` - canonical world, public fields and private authored predicates.
- `model/umpire/testing/src/lib.rs` - fixture producer and Files API.
- `model/umpire/testing/src/runner.rs` - recognized output ownership and deterministic comparison.
- `model/archived/framework/samples/Kettle.test.scala:255` - reference questions, paths and off-by-one boundaries, read-only.

**Optional:**
- `model/DESIGN.md` - current compatibility and follow-up sections.
- `.flow/tmp/fn214-query-seam-findings.md` - read-only design advice, not an implementation or task record.

### Quick commands

From `model/`, `mise exec -- cargo test -p umpire-checker -p umpire-tests -p umpire-testing`.
From the repository root, run `make model-check` and record one `make model-time` output under the 10 s / 30 s soft policy.

## Acceptance
- [ ] Supplementary deterministic goldens pin all six R4 questions with original registered predicates: longest 81 from 20, longest 141 from -40, unrestricted depress/press cycle, press-only deadlock in the specified switched-on state, bound-80 missed deadline, and budget-10 Unknown.
- [ ] The new query operation reuses an existing exploration and one progress checker, honors replacement assumptions/bounds, refuses catalog/registration misuse and does not duplicate Kettle behavior. Query path/hint fixtures pin actual deterministic witnesses; cached results/rows remain unchanged and no transition is re-executed.
- [ ] Both complete canonical sample directories are byte-identical to the task baseline; no aliases, duplicated Models, new runner, unregistered expected filenames or future suite DSL are added.
- [ ] README/DESIGN describe explicit fairness, bound/query syntax, legacy compatibility and the future `fair(A)` to `WF_vars(A)` contract; no implicit internal fairness remains in the current or future emitter description.
- [ ] Focused tests, `make model-check` and recorded `make model-time` pass; fn-213's ergonomic tests.rs presentation remains a named downstream obligation.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
