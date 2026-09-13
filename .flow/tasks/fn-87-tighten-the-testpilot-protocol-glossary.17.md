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
TBD

## Evidence
- Commits:
- Tests:
- PRs:

