---
satisfies: [R5]
---
# fn-94-simplify-the-testpilot-go-runtime.11 One opcode table for instruction binding and effect acceptance

## Description
Lane D1's largest item: one `opcodes` table replaces the per-opcode switches in `dataflow.go` and `scheduler.acceptEffect`. First pins rejection order beyond the corpus.

**Size:** M
**Files:** `common/testing/testpilot/internal/execution/dataflow.go`, `internal/execution/scheduler.go` (`acceptEffect` only), a new order-pin test in `internal/execution`, the root `common/testing/testpilot/README.md` extension-checklist steps that name the replaced switches
**Touches:** [common/testing/testpilot/internal/execution/dataflow.go, common/testing/testpilot/internal/execution/scheduler.go, common/testing/testpilot/internal/execution/*_test.go, common/testing/testpilot/README.md]
**Depends on (cross-spec):** fn-89-one-contract-rule-per-entity.6 (root README)

### Approach
- Pin first: a focused test that prepares Programs carrying two defects at once (for each pair of checks the table reorders, at least one) and asserts the first rejection's category and path. Commit it green before the refactor.
- Build the table: opcode → oneof arm name, entrypoint context, protocol-code flag, bind and dataflow-bind functions. Replace `InstructionOpcode` (`dataflow.go:17`), `opcodeContext` (`:44`), `bindInstruction` (`:79`), `bindOutcomes` (`:192`), `bindNodeDataflow` (`:551`) and `scheduler.acceptEffect` (`scheduler.go:703`). Checks inside each row keep today's order.
- Update the root README's extension checklist (`:141-143,162-165,178-181,257-259`) to say "one row in `opcodes`".

### Investigation targets
**Required:**
- `common/testing/testpilot/internal/execution/dataflow.go:1-200,540-600`
- `common/testing/testpilot/internal/execution/scheduler.go:690-760`
- `common/testing/testpilot/README.md:109-280`

### Quick commands
```sh
go test -race -tags test_dep ./common/testing/testpilot/...
make umpire-check-case-runtime-conformance
make lint-code-fast
```

## Acceptance
- [ ] The two-defect order test lands before the table and passes unchanged after it.
- [ ] No per-opcode switch remains in `dataflow.go` or `acceptEffect`.
- [ ] Corpus unchanged; checklist updated; tests and lint pass.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
