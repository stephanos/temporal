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
Migrated replay composition, World transitions, minimizer checkpoints, corpus identities, and the surviving shared Runner fixtures away from generic canonical JSON. World wire data now uses domain codecs; ordinary state identities use deterministic standard-library JSON with explicit string validation and strict typed reads. Review fixes restore invalid UTF-8 rejection for terminal details and compare diagnostic fixtures semantically while retaining repeatability and identity assertions.

Focused packages, affected Runner selectors, race, vet, architecture, toolchain, and qualification gates pass. Aggregate validation reaches only the known stale `golang-x-sys-v047-darwin-arm64` compatibility approval owned by fn153.17. The repository lint executable is unavailable on this linux/arm64 host because it has the wrong executable format; the worker's isolated base-scoped lint passed before integration.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: f0a1166772ef7ba01f763030a275bdcd4b3d4af7, c5fbe3bb575311e76d8d4176eb63206ee282c1e5, 950a452ad6db54cb5581a1a478be828df2f7f9e1
- Tests: go -C tools/gomad3 test -tags test_dep -count=1 ./world ./backends ./runner/internal/minimizer ./runner/internal/corpus ./runner/internal/campaign, go -C tools/gomad3 test -tags test_dep -count=1 -run <affected-runner-selectors> ./runner, go -C tools/gomad3 test -race -tags test_dep -count=1 ./world ./backends ./runner/internal/minimizer ./runner/internal/corpus, go -C tools/gomad3 vet -tags test_dep ./world ./backends ./runner ./runner/internal/minimizer ./runner/internal/corpus ./runner/internal/campaign, go -C tools/gomad3 test -tags test_dep -count=1 ./internal/archcheck, make -C tools/gomad3 validate-toolchain validate-qualification
- PRs:
## Revival trigger (2026-10-13)

Revived after fn153.11 completed with a SHIP implementation review, satisfying the last dependency and freezing the paired simulation formulas and projected-record ownership that shared Runner fixtures consume. This task now owns the remaining replay-composition, minimizer-checkpoint, corpus-semantic, and surviving shared Runner/campaign test migrations after fn152; it must preserve World wire bytes, interruption/resume state, artifact lifetimes, and every identity input without reopening fn152 storage. Its write surface is disjoint from the active qualification and builder-transaction audits, and final generated outputs remain assigned to fn153.17. The forced local claim is required only because Flow chain admission still sees the completed fn152 parent on an unpushed remote branch; pushing remains outside the requested scope.
