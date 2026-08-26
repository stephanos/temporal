---
satisfies: [R4, R5, R7]
---
# fn-14-milestone-a-pilot-baseline-and-lean.3 Export integrity-verified Agentworkflow evidence

## Description
Add the generic read-only Agentworkflow evidence export required to prove trial events and exact candidate patches for R4/R5/R7.

**Size:** M
**Files:** `tools/agentworkflow/internal/agentworkflow/evidence.go`, `tools/agentworkflow/internal/agentworkflow/engine.go`, `tools/agentworkflow/internal/agentworkflow/engine_test.go`, `tools/agentworkflow/internal/store/export.go`, `tools/agentworkflow/internal/store/store_test.go`, `tools/agentworkflow/internal/workspace/export.go`, `tools/agentworkflow/internal/workspace/workspace_test.go`, `tools/agentworkflow/internal/cli/cli.go`, `tools/agentworkflow/internal/cli/cli_test.go`
**Touches:** [tools/agentworkflow/internal/agentworkflow/evidence.go, tools/agentworkflow/internal/agentworkflow/engine.go, tools/agentworkflow/internal/agentworkflow/engine_test.go, tools/agentworkflow/internal/store/export.go, tools/agentworkflow/internal/store/store_test.go, tools/agentworkflow/internal/workspace/export.go, tools/agentworkflow/internal/workspace/workspace_test.go, tools/agentworkflow/internal/cli/cli.go, tools/agentworkflow/internal/cli/cli_test.go]

### Approach

- Add a generic `agentworkflow.evidence-export/v1` value and `agentworkflow export <run-id> --json` command; keep the schema free of Umpire/pilot terms.
- Reopen and integrity-validate the admitted request/checkpoint/result, attempt manifests and complete bounded event streams, source/base/candidate workspace digests, and change inventory before exporting.
- Derive deterministic bounded patch bytes from the verified base/candidate workspaces for added, modified, and deleted UTF-8 regular files. Reject binary, symlink, unsafe, oversized, or unreproducible changes.
- Bind patch/event members to the export identity/digest and emit canonical JSON with exact stdout/stderr/exit behavior.
- Keep export read-only: never recover, resume, rerun, apply, rewrite, or update access/state timestamps in the retained run.

### Investigation targets

**Required:**
- `tools/agentworkflow/internal/agentworkflow/engine.go:215-293` — current integrity-checked diff/reopen seams.
- `tools/agentworkflow/internal/store/store.go:33-88` — run and attempt manifests.
- `tools/agentworkflow/internal/store/recovery.go:54-87` — bounded completed-attempt reads.
- `tools/agentworkflow/internal/workspace/workspace.go:179-192` — verified base/candidate diff.
- `tools/agentworkflow/internal/cli/cli.go:370-430` — report/diff command patterns.

### Quick command

`cd tools/agentworkflow && GOWORK=off go test -count=1 -tags test_dep ./internal/...`

## Acceptance

- [ ] A valid retained run exports exact request/config/backend/result identities, every attempt manifest/event, and canonical patch bytes with stable digests.
- [ ] Added/modified/deleted UTF-8 files reproduce exactly; binary, symlinked, unsafe, oversized, or base/candidate-drifted changes fail closed.
- [ ] Corrupt request/checkpoint/result/attempt/event/workspace data and incomplete lifecycle state never produce an export.
- [ ] Repeated export is byte-identical and leaves the run store and source/candidate trees unchanged.
- [ ] The CLI is read-only, generic, canonical, and covered by exact stream/exit tests; existing comments are preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
