---
satisfies: [R1, R4, R5]
---
# fn-130-model-views.6 Render actual witness sequences and per-IR overviews

## Description
Join the view inventory with actual expecting-find witness sequences and overview links.

**Size:** M
**Files:** `tools/umpire/render/witness.go`, `tools/umpire/render/witness_test.go`, `tools/umpire/render/overview.go`, `tools/umpire/render/overview_test.go`
**Touches:** [tools/umpire/render/witness*, tools/umpire/render/overview*]

### Approach
- Enumerate FORM_FIND with GetExpectedRun()!=nil from the admitted inventory. Use existing checked receipts with fresh replay, or the existing reader Answer/Replay path as proven in .1; verify the original bound/status and initial state before depicting any ordered sequence.
- Join witness Rows to actual selected results and outcome/state/facts. Preserve repeated rows and timers/internal actors; list ambiguous Because/Choice matches rather than inventing clause-level evidence. Keep Exercised query-wide.
- Generate Mermaid text and per-IR README links for all applicable machine/refinement/diff/composition views and lint coverage. Every non-found/limited/unknown/replay-rejected Query retains its standing rather than an invented diagram.
- Independently enumerate expected outputs from admitted input declarations and test dropped/extra machines, compositions, finds and broken links. Do not use the renderer's own output list as its completeness oracle.

### Investigation targets
**Required:**
- `tools/umpire/check/checking.go:134` - receipt/witness/row carriers.
- `tools/umpire/check/checking.go:653` - independent Query replay.
- `tools/umpire/check/claims.go:338` - public Find declaration binding.
- `tools/umpire/check/replay_test.go` - definition-ID replay pins.
- `tools/umpire/lint/lint.go:209` - coverage tallies.

### Quick commands
```bash
mise exec -- go test -count=1 -tags test_dep ./tools/umpire/render -run 'Test(Witness|Overview)'
```

## Acceptance
- [ ] Every qualifying Query has a truthful sequence or explicit attributed standing with preserved replay/limits/identity.
- [ ] Ambiguous/repeated rows, missing initial state, malformed results and replay failures cannot produce misleading success text.
- [ ] Overview links and independently expected full inventories match all applicable views; deterministic Mermaid/Markdown escaping controls pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
