---
satisfies: [R7, R8]
---
# fn-87-tighten-the-testpilot-protocol-glossary.17 Protocol extension checklist and final gate sweep

## Description
Write the extension section R7 asks for on the final protocol shapes, with the worker-stop fault kind traced through every place as the worked example, sweep the documentation and glossary for words this spec retired, and run the full R8 gate set once on the finished protocol with a fixture receipt.

**Size:** S
**Files:** `common/testing/testpilot/README.md` (new "Extending the protocol" section), `tools/umpire/CONTEXT.md` (glossary entries for Expression, Reference, response read, Case-local name; `_Avoid_:` lines for retired words), remaining Testpilot and model docs with stale wording, `common/testing/testpilot/internal/protocolmigration/README.md` (final step list)
**Touches:** [common/testing/testpilot/README.md, common/testing/testpilot/internal/**/README.md, common/testing/testpilot/temporal/**/README.md, tests/testcore/testpilot/README.md, tools/umpire/CONTEXT.md, model/README.md, model/ARCHITECTURE.md, model/Umpire/ARCHITECTURE.md, common/testing/testpilot/internal/protocolmigration/README.md, common/testing/testpilot/internal/execution/*.go, common/testing/testpilot/internal/ir/catalog.go, common/testing/testpilot/internal/ir/type.go, common/testing/testpilot/internal/ir/expression_test.go, common/testing/testpilot/contract/driver.go, model/Umpire/Property/Correlated.lean, model/Umpire/Case/Tests/CorrelatedFixtures.lean, model/Temporal/Testpilot/GetSystemInfo.lean]

### Approach
- README section, after the opening overview and before "Preparation diagnostics" (plain prose with `##`/`###` subsections, matching the file): for each extension kind (new instruction, fault kind, Run Event payload, expression reference) list every place that must change, in order: the protocol file (by concept), `make proto` and the Lean elaboration (`model/lakefile.lean` schema list if a file is added), `Testpilot.Authoring` constructors, the Go interpreter (`internal/execution`) or evaluator (`internal/ir`, `internal/verification`), the kind/payload table (.7) or context table (.5), the Profile Opcode (`contract.Opcode`) and Driver, the conformance class or unit test, the retired-vocabulary gate, the equivalence mapping (only while fn-87's baseline exists) and the fixture regeneration target. Trace `FAULT_KIND_WORKER_STOP` through each with the real file and symbol names as they exist after .16 (verify each by grep; no stale names). Note that fn-85 R10 is the first planned use.
- Glossary: add `tools/umpire/CONTEXT.md` entries in its format (bold term, one-line definition, `_Avoid_:`) for Expression, Reference, response read and Case-local name, and add avoided words (`response projection`, `opaque capability`, `clause` for Correlated Rule) where they point at the new words.
- Docs sweep: grep the Testpilot and model docs listed in Touches for words retired by .2–.16 and fix stragglers; do not edit `.plans/UMPIRE_CASE_RUNTIME_DESIGN.md` or other historical plans (Boundaries).
- Carried from .3: internal Go names and error strings that still say "projection" or "capability" for response reads and opaque handles (`ProjectionPlan`, `bindProjections`, "capability Slot" error text) and the comment at `model/Umpire/Property/Correlated.lean:155` ("response projection"): rename to the glossary words where they name response reads or handles; leave `Umpire.Case.Projection` and the Driver-contract `Capability*` names alone. Add those paths to this task's Touches when you edit them.
- Final gates, in this order, recording numbers: `go clean -cache`; `make umpire-check-testpilot-protocol`; `make umpire-check-testpilot-authoring`; `make umpire-check-case-runtime-conformance`; `make umpire-check-retired-vocabulary`; `CC=/usr/bin/cc TMPDIR=$(cd "${TMPDIR:-/tmp}" && pwd -P) make umpire-check-live-tests`; `make umpire-check-regression`; `make lint-model` (163); `make lint-code GOLANGCI_LINT_FIX=false` (161 inherited). The receipt lists every fixture with its baseline and final byte size and the equivalence mapping's step list by R-ID.

### Investigation targets
**Required** (read before coding):
- `common/testing/testpilot/README.md`
- `tools/umpire/CONTEXT.md` (entry format)
- `common/testing/testpilot/internal/protocolmigration/mapping.go` (final steps)
- `.plans/UMPIRE4_ORDER.md` "Gate baselines"

**Optional:**
- `model/Temporal/Feature/Nexus/DESIGN.md` sections on need 8 (fn-85's use of the checklist)

### Key context
- `.plans/UMPIRE4_ORDER.md` is updated by the spec owner after the completion review, not in this task.

## Acceptance
- [ ] `common/testing/testpilot/README.md` has the extension section covering instruction, fault kind, Run Event payload and expression reference, with the worker-stop fault traced through every place by current file and symbol names
- [ ] `tools/umpire/CONTEXT.md` has the new glossary entries; no doc in Touches spells a retired protocol word
- [ ] all R8 gates pass with numbers recorded (regression exit 0 and nine live identities, lint-model 163, lint-code at most 161 after `go clean -cache`); the receipt lists every fixture's baseline and final size and the mapping steps by R-ID


## Done summary
Future protocol work (fn-85 R10 first) now has a written checklist. `common/testing/testpilot/README.md` gains "Extending the protocol", which lists the ten places a new instruction, fault kind, Run Event payload and expression reference change, and traces `FAULT_KIND_WORKER_STOP` through each by current file and symbol, every name verified by grep and by the reviewer. `tools/umpire/CONTEXT.md` defines Expression, Reference, Case-local name and response read, and avoids "response projection", "opaque capability" and "clause" for a Correlated Rule.

**Docs and names**
- `protocolmigration/README.md` gains "Final mapping", a table of the 59 `Declared` steps by R-ID plus the two R3 `Added` fixtures. R2 and R11 need no step.
- Docs sweep fixes: response read ordinals (execution README), "applies declared response reads" and "the admitted correlated capability" (the correlated contract's version was removed in .8) in `model/ARCHITECTURE.md`, "declared typed field" and "rule/source provenance" in `model/Umpire/ARCHITECTURE.md`, opaque handle wording in the server and worker READMEs, "path traversal" in the verification README, and the glossary Slot entry ("recorded as an Observation").
- Carried from .3: `ProjectionPlan`/`Projections()` became `ResponseReadPlan`/`ResponseReads()`. `projection`, `bindProjections`, `bindProjectionSinks`, `stageProjection` and `projectionFact` became `responseRead`, `bindResponseReads`, `bindReadTargets`, `stageResponseRead` and `readFact`. `projection.go` and its test became `response_read.go`. A bound read's `sinks`/`Sinks` became `targets`/`Targets` (review P3). Error text now says "response read ... target", "handle Slot", "publish handles" and "opaque handle literals", and the runtime error path is `response_read`. Test helpers are `handleSlot`/`handleFixture`. The Lean strings "unsupported response/trigger projection" became "... pattern field", and two Lean comments now say response read.
- Kept: Driver-contract `Capability*`/`OpaqueCapability`, `Umpire.Case.Projection`, the wire `CorrelatedEvidenceProjection` and `projected_value`, and the Run event source-id format `%s.p%d.i%d`. The source ids stay because changing them would move Run bytes.
- These paths were added to Touches: `internal/execution/*.go`, `internal/ir/{catalog,type}.go`, `internal/ir/expression_test.go`, `contract/driver.go`, and three Lean files.

**Gates** (at 3709f3723f)
- `go clean -cache` ran first.
- `make umpire-check-testpilot-protocol`, `umpire-check-testpilot-authoring`, `umpire-check-case-runtime-conformance` and `umpire-check-retired-vocabulary` all exited 0.
- `make umpire-check-live-tests` exited 0 with 9 passing live identities.
- `make umpire-check-regression` exited 0 with 9 passing live identities (583 Lean jobs) on the first run, with no flake. Green receipt 3709f372 written.
- `make lint-model`: 163 (baseline). `go clean -cache && make lint-code GOLANGCI_LINT_FIX=false`: 161 (baseline), none in testpilot.
- Baseline before editing: green via regression receipt 4941c94b plus the oracle run.

**Fixture sizes in bytes, spec baseline (d9c77573) -> final**
- typed-nexus 315,914 -> 43,215
- async-nexus 35,898 -> 22,107
- worker-outage 23,812 -> 13,801
- typed-unary 19,316 -> 10,471
- get-system-info 4,914 -> 3,287
- synthetic 3,751 -> 2,311
- correlated.json 365,207 -> 356,224
- conformance `case.json`:
  - cleanup-failure 4,969 -> 2,983
  - cross-run 4,062 -> 2,601
  - inconclusive 4,022 -> 2,567
  - satisfied 4,004 -> 2,551
  - static-rejection 4,045 -> 2,586
  - violated 3,996 -> 2,545
  - expression-context: added (R3), 2,814
- `expected.json` files are byte-identical to the baseline (1,995; 1,752; 1,695; 1,722; 92; 1,597). expression-context `expected.json` was added (R3), 247.
- This task regenerated no fixture.

**Mapping steps by R-ID** (in `Declared` order)
- 1-13 R1: Contract, Run and correlated renames
- 14-34 R1: Program, instruction and Value renames
- 35-38 R3: comparison operators, one Expression over Reference, capture `observation_id`, `CorrelatedCaptureReference`
- 39 R1: `ResponseRead.cardinality`
- 40-43 R3: correlated conditions and the lift guard as Expressions (40 is also R5, 43 also R6)
- 44 R4: fault payload paths
- 45 R5: `Deadline` bound oneof
- 46-49 R6: capture `SingularType`, correlated version removed, `NamedValue`/`NamedExpression`
- 50-51 R6: unsigned integer value, one opaque-handle encoding
- 52 R12: ceilings in the Profile
- 53 R15: enum names (Resolve)
- 54 R9: default order and `after`
- 55 R10: derived declarations
- 56 R13: typed provenance rows
- 57 R14: Case-local names (Relate)
- 58-59 R15: path strings, then dropped compared presence
- `Added`: the expression-context Case and its `expected.json` (R3).

**Incident**
- My first follow-up commit 7e25b28206 did not compile (a shadowed `target` in `bindReadTargets`), and something outside this session pushed it to `stephanos/umpire` within seconds.
- I amended the fix locally. A `git pull` rebase that this session did not start then stopped on a conflict.
- I resolved the conflict to the fixed file and continued the rebase instead of aborting it, so local history stays a fast-forward of the remote with no force push.
- Result: 7e25b28206 (broken, on the remote) is followed by 3709f3723f (the fix). The fix commit reuses the same subject line. Its tree equals the intended amended commit 2f3fa9118f, and every gate above ran on it.

**Deferred follow-ups**
- `ir`'s "reference or projection requires an explicit presence guard" and its `project*` compiler methods name path reads, which the glossary now avoids calling projection. Four tests pin the string, and it is outside the response read and handle carve-out.
- Hand-written Go still calls Opcodes "capability" (`opcodeContext(capability)`, "unknown capability", `InstructionCapability`, the `MaxOpcode` comment) even though the glossary Opcode entry avoids that word.
- The `.gitignore` negation for the frozen baseline tree (.1) and protogen `rewriteEnumReferences` P3 (.4) were not applied, since both are outside this task's surface.

stage: impl-review - ran (claude backend, SHIP on the first round; P3s applied in 3709f3723f: targets naming and the `Added` file name, plus the pre-existing verification README wording)
## Evidence
- Commits: 0b932755766c4fc2e061bea5813d0b2b24c0f212, 7e25b2820652cceb5e931aa120eaf7f448c5d4fa, 3709f3723fefd1f423ea4497203fc7cc4e32e9bd
- Tests: go test -count=1 -tags test_dep ./common/testing/testpilot/... ./tests/testcore/testpilot/..., go clean -cache, make umpire-check-testpilot-protocol, make umpire-check-testpilot-authoring, make umpire-check-case-runtime-conformance, make umpire-check-retired-vocabulary, CC=/usr/bin/cc TMPDIR=$(cd "${TMPDIR:-/tmp}" && pwd -P) make umpire-check-live-tests, CC=/usr/bin/cc TMPDIR=$(cd "${TMPDIR:-/tmp}" && pwd -P) make umpire-check-regression, make lint-model, go clean -cache && make lint-code GOLANGCI_LINT_FIX=false
- PRs: