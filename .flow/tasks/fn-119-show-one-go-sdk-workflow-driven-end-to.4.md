---
satisfies: [R1, R3, R8]
---
# fn-119-show-one-go-sdk-workflow-driven-end-to.4 Model the activity workflow example and run its Queries live from the gate

## Description
Author the example Model in the finished DSL and wire it into the model gate and the existing live test job.

**Cross-spec entry gate:** start only after fn-118 and fn-120 are both closed (and therefore fn-112/fn-114): the example uses the final DSL, named choices, Scala-owned roots and declared API behavior, with no proto name as a string and no literal wait.

**Size:** M
**Files:** `model/examples/activityworkflow/{Model,Properties,Queries,Realization}.scala` and its IR-file declaration (module map reserves `model/examples` for fn-119; the spec records this location); Makefile `MODEL_SOURCES` (:670), `MODEL_JAR` find (:702), scalafix roots; `model/check/Gate.scala` `models` (:50-54); `model/ir/<example>.json`; `model/cases/**`; `.plans/UMPIRE_MODULES.md`.
**Touches:** [model/examples/**, model/temporal/realize/**, tests/testpilot_generated_test.go, tests/testcore/testpilot/**, Makefile, model/check/**, model/ir/**, model/cases/**, .plans/UMPIRE_MODULES.md, .flow/specs/fn-119-show-one-go-sdk-workflow-driven-end-to.md]

### Approach
- Product machine for workflow + one activity with completion, retry-then-completion and timeout; at least those three `find` Queries with authored totals and `RunExpectation`s; realization via task 3's DSL and fn-118 hints. Keep it to a few screens.
- Declare in the shared kit, with fn-118's existing hint kinds and a server citation each, the visibility of every write->read pair the example uses that fn-118 did not already declare; no new hint kind.
- Wire `model/examples` into all four source lists so lint, compile, lift and jar build see it; confirm the location in the module map.
- Cases land in `model/cases` and `manifest.json`, so `TestTestpilotGeneratedCases` (`tests/testpilot_generated_test.go:142`, `make umpire-check-live-tests`, CI `umpire.yml:33-36`) runs them live and replayed; no new CI workflow (declined-scope ledger: no new CI coverage).
- The runner skips Cases rejected as `PreparationUnsupported` (:187-191). Make the example's Cases fail instead of skip when unsupported, through generic manifest/expectation data, not a Go test naming the example.
- A DSL gap is a finding (R1 errors) appended to `.flow/tmp/fn119-3/findings.md`, never worked around in Go.

### Investigation targets
**Required:**
- `model/temporal/features/standaloneactivity/` (final fn-112 form) - style
- `tests/testpilot_generated_test.go:100-210`, `tests/testcore/testpilot/model_fixture.go:80-115`
- `model/check/Gate.scala:40-60,290-380`
- `Makefile:665-740`

### Quick commands
```bash
make umpire-gen-model && make umpire-check-model
make umpire-check-live-tests
```

### Execution constraints
- No Go file names the example (task 5 adds the check; respect it from the start).
## Acceptance
- [ ] The example Model has a product machine, at least three Queries (completion, retry then completion, timeout) and a realization, and passes the model gate.
- [ ] Every Query lowers to a Case; the live runner runs each against the in-process cluster with a satisfied Verdict and conforming assessment, live and replayed; an unsupported example Case fails rather than skips.
- [ ] The example contains no proto name as a string and no literal wait; any hint it needed is declared once in the shared kit.
- [ ] The example is in the model gate and the existing live test job; no new CI workflow.
- [ ] Its location under `model/examples` is reflected in the module map.
## Done summary
Blocked:
Blocked: deferred by the owner on 2026-10-04 as not needed for the code deliverable (the DSL and its execution). fn-119 (the Go SDK workflow showcase) waits until it is revived; its generic Driver primitives (tasks 1-2) are done.
## Evidence
- Commits:
- Tests:
- PRs:
