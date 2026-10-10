---
satisfies: [R2]
---
# fn-153-gomad-retire-canonical-json-and-private.12 Migrate replay composition, minimizer checkpoints and corpus semantics

## Description
Implements R2 for this owner; see the parent Architecture and Delivery sections.

**Size:** M
**Files:** runner/replay_operation.go, internal/execution/worldrecord.go, internal/minimizer/{minimizer,workspace}.go and internal/corpus/{model,features}.go (6 core callers). Wider mechanical test migration includes surviving shared Runner, campaign and corpus test fixtures after fn152; no storage implementation rewrite.
**Touches:** [tools/gomad3/runner/replay_operation*.go, tools/gomad3/runner/*_test.go, tools/gomad3/runner/internal/execution/worldrecord*.go, tools/gomad3/runner/internal/minimizer/**, tools/gomad3/runner/internal/corpus/model*.go, tools/gomad3/runner/internal/corpus/features*.go, tools/gomad3/runner/internal/corpus/*_test.go, tools/gomad3/runner/internal/campaign/*_test.go]

### Approach

- Route World payloads/transitions/terminal comparisons through the proved World domain codecs; ordinary host checkpoint/state identity encoding uses stdlib and strictjson.
- Keep exact replay/control ordering, minimizer parent/config/candidate binding, checkpoint interruption semantics, corpus feature/selection projections and artifact handle lifetime.
- Behavior pin: frozen replay outcomes and exact World payloads, interrupted/resumed minimization vs uninterrupted state, full identity-input mutations and corpus semantic feature/index comparisons.
- Re-anchor fn152's model/features survivors. Do not remigrate corpus log/storage or alter borrowed/owned artifact cleanup.
- Use shared replacement wrappers already adopted by minimizer; no new private publisher.
- Migrate surviving shared Runner test dependencies after the choice/simulation owners, including diagnostic_identity, runner, coordinator_transport, completion, retention, inspect and campaign-options fixtures. Reinventory campaign/corpus tests after fn152 and migrate only remaining generic-package references. Preserve their frozen behavior assertions; do not restore deleted journals or rewrite log semantics. Retain task10/11's already-migrated tests unchanged unless a concrete remaining reference requires a coordinated correction.

### Investigation targets

**Required** (read before coding):

- `tools/gomad3/runner/replay_operation.go:164`
- `tools/gomad3/runner/internal/execution/worldrecord.go:32`
- `tools/gomad3/runner/internal/minimizer/minimizer.go:291`
- `tools/gomad3/runner/internal/minimizer/workspace.go:339`
- `tools/gomad3/runner/internal/corpus/model.go:132`
- `tools/gomad3/runner/internal/corpus/features.go:73`
- `tools/gomad3/runner/internal/minimizer/minimizer_test.go`

### Verification

Focused command: go -C tools/gomad3 test -tags test_dep -count=1 ./runner/internal/minimizer ./runner/internal/corpus ./runner/internal/execution. Also run the exact affected root Runner test names from the survivor inventory; bind those selectors to the handover instead of rerunning unrelated suites.

Capture the frozen behavior pin named above, run focused negative controls and follow the parent Delivery and verification section. Declare scope growth to the conductor before implementation. Shared gates run serially on the integrated frozen candidate.
## Acceptance
- [ ] World replay/terminal/transition composition preserves exact wire bytes and current semantic authentication; minimizer and corpus identities retain all inputs.
- [ ] Checkpoint interruption/resume, artifact ownership and corpus selection behavior match the frozen pin; storage ownership remains fn152's.
- [ ] Malformed/duplicate/unknown/trailing, invalid string/numeric/config, stale parent/candidate and changed replay evidence retain rejection and public errors.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
