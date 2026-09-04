---
satisfies: [R7, R8]
---

# fn-22-deterministic-replay-semantic.7 Expose bounded replay orchestration and reporting
## Description
Expose a thin local replay/minimize/propose command over the deep library controller. Accept only canonical subject inputs, fixed local Profile, and explicit proposal destination; emit bounded progress and one closed terminal report with reproduction, reduction, cleanup, proposal, and tooling states separated.

**Size:** M
**Touches:** `tools/umpire/cmd/umpire-replay/**`, `Makefile`

## Acceptance
- [ ] The CLI exposes no arbitrary Host, checker, executable, semantic edit, or compatibility option.
- [ ] Output, exit statuses, cancellation, reporting failure, and publication ambiguity are canonical and bounded.
- [ ] Reporting or proposal failure never installs a regression or reruns target effects.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
