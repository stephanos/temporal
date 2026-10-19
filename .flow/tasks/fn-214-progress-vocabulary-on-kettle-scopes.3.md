---
satisfies: [R4, R5]
---
# fn-214-progress-vocabulary-on-kettle-scopes.3 Kettle's progress questions as fixtures; fairness and emitter contract in DESIGN.md

## Description
Implement R4 and R5 with supplementary Kettle question fixtures and the narrow query seam they need.
This task consumes .1/.2; it does not introduce the future suite's Space, Run or starts DSL.

**Size:** M
**Files:** checker registered-progress query, shared fixture adapters/questions and current documentation.
**Touches:** [model/umpire/checker/src/explore.rs, model/umpire/checker/src/progress.rs, model/umpire/checker/src/lib.rs, model/umpire/checker/tests/**, model/umpire/tests/fixtures/kettle_progress*/**, model/umpire/tests/support/**, model/README.md, model/DESIGN.md]

### Approach

- Add a narrow `Result::verify_progress(&self, world, &registered_claim) -> Result<ClaimResult, Refusal>` operation over an existing exploration, reusing .2's algorithm. It checks the selected registered claim with replacement assumption/bound options, without re-exploring or adding Model behavior. The later suite can wrap this operation.
- Import the canonical attribute Kettle by its existing fixture source path. Use a fixture-side generic Machine adapter to select room/cold starts, delegate actions/internal/step to the original world and forward the original registered progress callbacks. Construct starts through the existing public Model fields. Do not duplicate transition or boils predicate bodies, expose private methods by rewriting the sample, or change either canonical sample directory.
- Select the original `boils` registration from `Machine::progress()`. The future `Model::boils.assuming(...)` suite notation is not yet an existing descriptor API; use the registered claim now and preserve the future obligation for fn-213.
- Produce supplementary checker-owned `report.txt` and `paths/*.txt` with a fixture `run()` using existing Files/consumer discovery and deterministic double rendering. Do not add arbitrary output filenames or a second fixture runner.
- Compare the entire two canonical sample directories with the committed task baseline, including source and snapshot metadata. Preserve them byte-for-byte while pinning all R4 questions in separate fixture directories.
- Update README and DESIGN for explicit fairness, numeric authoring bounds, query operation and legacy compatibility. Record that fn-212 emits `WF_vars(A)` only for explicit fair entries. Keep the generic hint text without claiming that fair(heat) repairs Kettle's press/depress cycle; heat is disabled while the element is off.

### Investigation targets

**Required:**
- `model/umpire/checker/src/explore.rs:64` - current exploration registration and cached report seam.
- `model/umpire/tests/fixtures/sample_kettle/model.rs` - canonical world, public fields and private authored predicates.
- `model/umpire/testing/src/lib.rs:22` - fixture producer and Files API.
- `model/umpire/testing/src/runner.rs:65` - recognized output ownership and deterministic comparison.
- `model/archived/framework/samples/Kettle.test.scala:255` - reference questions, paths and off-by-one boundaries, read-only.

**Optional:**
- `model/DESIGN.md` - current compatibility and follow-up sections.

### Quick commands

From `model/`, `mise exec -- cargo test -p umpire-checker -p umpire-tests -p umpire-testing`.
From the repository root, run `make model-check` and record one `make model-time` output under the 10 s / 30 s soft policy.
## Acceptance
- [ ] Supplementary deterministic goldens pin all six R4 questions with original registered predicates: longest 81 from 20, longest 141 from -40, unrestricted depress/press cycle, press-only deadlock in the specified switched-on state, bound-80 missed deadline, and budget-10 Unknown.
- [ ] The new query operation reuses an existing exploration and the progress checker, honors replacement assumptions/bounds, refuses misuse and does not duplicate Kettle behavior. Query path/hint fixtures pin actual deterministic witnesses.
- [ ] Both complete canonical sample directories are byte-identical to the task baseline; no aliases, duplicated Models, new runner, unregistered expected filenames or future suite DSL are added.
- [ ] README/DESIGN describe explicit fairness, bound/query syntax, legacy compatibility and the future `fair(A)` to `WF_vars(A)` contract; no implicit internal fairness remains in the current or future emitter description.
- [ ] Focused tests, `make model-check` and recorded `make model-time` pass; fn-213's ergonomic tests.rs presentation remains a named downstream obligation.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
