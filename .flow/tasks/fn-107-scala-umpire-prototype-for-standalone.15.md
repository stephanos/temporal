---
satisfies: [R2, R3, R9]
---
# fn-107-scala-umpire-prototype-for-standalone.15 Admit and interpret the portable finite semantic IR

## Description
**Touches:** [model/scalav2/goir/**]

Extract IR admission and primitive finite interpretation from task 3. Task 14 owns generic checker algorithms; task 3 binds authored claims and receipts after both lanes finish. Do not depend on unintegrated task 14 APIs.

**Size:** M
**Files:** existing load.go/eval.go/machine.go, reusable interpreter helpers, and focused admission/interpretation tests.

### Approach
- Admit the task-2 IR with located version/reference/type/duplicate/bounds diagnostics. Reject malformed or unsupported executable constructs explicitly.
- Interpret finite values, bounded FIFO/unordered channels and their declared delivery/loss/duplicate choices, nondeterministic transitions, and declared/undeclared semantic holes through one evaluator.
- Preserve disabled-empty transitions separately from reachable holes, source identity, and all supported legacy Nexus tables/results/fingerprints.
- Expose reusable evaluation/metadata for task 3 to bind monitors, assumptions, composition, claims and queries; no partial consumer may silently report a query success while ignoring those declarations.
- Bound catalog/table work before allocating a large finite product. Exercise a tenfold finite-input probe: complete under sufficient ceilings or return an explicit resource-limit result with actual counts under insufficient ceilings.
- Use ordinary source-derived fixtures from task 2 for positive semantics and small generic malformed-IR controls for admission; never hand-construct feature IR or edit feature policy.

### Investigation targets
**Required:** model/scalav2/goir/load.go; model/scalav2/goir/eval.go; model/scalav2/goir/machine.go; model/scalav2/goir/diagnostics_test.go; model/scalav2/goir/parity_test.go; proto/internal/temporal/server/api/modelir/v1/ir.proto; model/scalav2/SEMANTICS.md; model/scalav2/lifter/testdata/.
**Optional:** existing generic Go table/claims APIs, read-only.

### Quick commands
`mise exec -- go test -tags test_dep ./model/scalav2/... ./model/go/...`; scoped Go IR lint. Record any inherited repository-wide lint failure separately.
## Acceptance
- [ ] Unknown versions, invalid/crossed references and types, duplicate declarations, malformed bounds and unsupported executable constructs produce located admission errors.
- [ ] Source-derived bounded-channel/presence/choice fixtures evaluate correctly; disabled actions and semantic holes remain distinct; legacy table/result/identity pins pass.
- [ ] Catalog/table construction completes within sufficient ceilings or reports explicit resource exhaustion with actual work counts, including the tenfold input probe.
- [ ] The interpreter exposes enough typed evaluation and declaration metadata for task 3 without silently claiming unchecked monitor/property/progress results; focused Go gates pass.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
