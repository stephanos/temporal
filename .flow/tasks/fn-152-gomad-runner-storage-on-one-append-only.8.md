---
satisfies: [R1, R2, R8, R9]
---
# fn-152-gomad-runner-storage-on-one-append-only.8 Expose log and corpus state through existing Inspect dispatch and migrate consumers

## Description
Implement R9 and migrate the surveyed active storage consumers.

**Size:** M
**Files:** Core files (4-5): runner/inspect.go, inspect_capacity.go, cmd/gomad/internal/cli/cli.go and focused Runner/CLI inspection tests. Wider touches: characterization/projection/exploration-divergence fixtures; qualification/set/prune.go and tests only if campaign path assumptions changed; e2e readiness fixtures move to committed log evidence; qualification diagnostic and artifact fixture constructors retain their source contracts.
**Touches:** [tools/gomad3/runner/inspect*.go, tools/gomad3/cmd/gomad/internal/cli/**, tools/gomad3/cmd/gomad/e2e_test.go, tools/gomad3/qualification/set/**, tools/gomad3/qualification/qualification.go, tools/gomad3/qualification/soak/**, tools/gomad3/cmd/gomadtool/diagnostic.go, tools/gomad3/internal/gomadtool/conformance/testdata/runner_external/consumer.go]

### Approach

Extend existing inspect PATH dispatch by the current log envelope kind, including direct corpus-directory paths. This is a root design inference from the existing campaign/artifact/plan dispatcher, not a quoted original user command. Preserve --json and --choices grammar. Project log kind/lineage, snapshot EOF, validated record count/offset, observed/repaired tail distinction, corruption location/reason, final committed state and applicable count/byte limits. A recognized corrupted store returns a structured report plus typed error so CLI writes its human/JSON diagnosis on stderr and returns status 2; JSON errors have no additional plaintext duplicate; output delivery failure remains status 3. Healthy/torn observation uses status 0 and never mutates storage. Preserve artifact and portable-plan inspection, existing outcomes/replay commands and diagnostics visibility. Update the complete active consumer inventory rather than replacing unrelated qualification manifests also named corpus.json.

### Investigation targets

**Required** (read before coding):

- `tools/gomad3/runner/inspect.go:361`
- `tools/gomad3/cmd/gomad/internal/cli/cli.go:145`
- `tools/gomad3/cmd/gomad/internal/cli/cli_test.go:785`
- `tools/gomad3/qualification/set/prune.go:36`
- `tools/gomad3/qualification/qualification.go:614`
- `tools/gomad3/cmd/gomad/e2e_test.go:379`
- `.flow/memory/bug/integration/engine-summary-fields-need-the-runner-2026-10-03.md`

### Verification

Investigate tools/gomad3/runner/inspect.go:361,437,455; cmd/gomad/internal/cli/cli.go:145,200; cli/cli_test.go:785; qualification/set/prune.go:36; qualification/qualification.go:614; cmd/gomad/e2e_test.go:379; .flow/memory/bug/integration/engine-summary-fields-need-the-runner-2026-10-03.md. Focused command uses go -C tools/gomad3 test -tags test_dep -count=1 with existing portable Inspect/CLI selectors, retained exactly.

Follow the parent spec's Delivery and verification section. Retain exact selectors and current command scope; do not substitute portable coverage for supported-host evidence. If deletion/fixture migration expands the surviving implementation beyond this cohesive owner, stop for conductor scope splitting before implementation.

## Acceptance
- [ ] R9: campaign, merged campaign and direct corpus-directory inspection in human and JSON form derives state from read-only log replay and shows log count/bytes/offset/kind/version and snapshot/tail status.
- [ ] R9: recognized corruption emits a human or JSON diagnosis on stderr and returns status 2 without successful-state stdout or a plaintext JSON duplicate; partial replay is explicitly an invalid store, missing/invalid paths retain prior diagnostics precedence, and report delivery failures return status 3.
- [ ] R1/R9: inspect reports an observed torn final tail without repair or writer-lock acquisition, remains usable while a writer appends, and distinguishes corruption/unsupported lineage from the validated prefix.
- [ ] R2/R8/R9: active qualification/pruning/diagnostic consumers and CLI/manual fixtures use current APIs or payload paths; old journal readers are gone, unrelated qualification corpus manifests are preserved, and artifact/plan/--choices inspection still works.
- [ ] R9: public projections retain classified outcomes, replay evidence and existing byte/count/novelty/lifecycle fields whose semantics survive the storage rewrite.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
