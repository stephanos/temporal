---
satisfies: [R3, R4, R5, R6, R7, R8, R10]
---

# fn-29-bounded-production-canary-execution-and.8 Compose the canary controller and its run mode

## Description
Add `tools/canary/controller/controller.go`, taking three injection seams -- a policy source, a transport source and a phase hook -- whose untagged values are the embedded policy, `authority`'s `Transport` and no hook, and composing, in order, with the `Transport` `authority` builds: preflight (which prepares the Case once), the lease, the serial iterations with each iteration's record, admission, assessment and rendering held in memory (the `decide` .4 takes), the cleanup attempt and the lease's release (or, when cleanup is uncertain, the lease left held), then publication whatever that outcome; and `tools/canary/cmd/umpire-canary` with the closed mode `run --output <dir> --recovery <file>` (outside the model; the directory must exist) and no Case, target, Driver, checker, retry, executable, endpoint, credential or release flag. It writes one bounded JSON summary on stdout (each iteration's status, receipt and provenance identities, cleanup) and bounded progress on stderr through the Redactor, and exits by precedence 3 > 2 > 1 > 0: 3 for a preflight or tooling failure, a publication conflict (`publication-conflict`) or a publication it could not report (each with a named status), 2 for `lease-unreconciled` (the lease found open or closed without a canary termination) or when cleanup is uncertain and the lease stays held, 1 when an iteration is rejected or incomplete, 0 when every iteration is accepted. An iteration whose Run returns an error or whose record fn-26 admission rejects is `unconstructible`: it has no receipt, ends the invocation, and exits 3. `make canary-build` builds it.

### Quick commands
`go test -count=1 -tags test_dep ./tools/canary/...`

**Files:** `tools/canary/controller/controller.go`, `tools/canary/controller/controller_test.go`, `tools/canary/cmd/umpire-canary/**`, `Makefile`
**Touches:** `tools/canary/controller/controller.go`, `tools/canary/controller/controller_test.go`, `tools/canary/cmd/umpire-canary/**`, `Makefile`

### Re-plan note (2026-09-23)
Rewritten on fn-85 (the canary set), fn-83 (provisioning), fn-22 (the recorded Run) and fn-26 (Claim Assessment); the spec's **Re-plan** section states the contracts.

## Acceptance
- [x] Stage order and the exit statuses distinguish accepted, rejected or incomplete, lost or uncertain, and tooling failure.
- [x] The prepared Case is reused across iterations while each Run has a fresh Driver and fresh fenced identities.
- [x] No arbitrary Case, target, Driver, checker, retry, executable, endpoint, credential or release option exists.

## Done summary
`tools/canary/controller/controller.go`: `Invoke` takes three seams (policy with its Evaluation Profile, authority, phase hook; untagged `ProductionSeams` = embedded policy and Profile, `authority.Load`, no hook) and composes preflight over a lazily dialed client, the recovery record, the lease and serial iterations decided in memory with fn-26, the cleanup attempt, and publication under its own context whatever the cleanup outcome. It returns a bounded JSON `Summary` and the exit by precedence 3 > 2 > 1 > 0 with named statuses (preflight refusals, `authority-unavailable`, `policy-unavailable`, `tooling-failure`, `interrupted`, `unconstructible`, `no-iteration`, `publication-conflict`/`-unreported`/`-failed`; `lease-unreconciled`, `cleanup-uncertain`; rejected/incomplete; accepted). `tools/canary/cmd/umpire-canary` has the closed `run --output --recovery` mode and refuses every other option; the untagged build's seams file is `!canary_harness`. `make canary-build` builds it. Tests cover every status through a fake server and the recorded Run, and the command line's refusals. Implementation review: NEEDS_WORK, then SHIP; P3 notes applied.
## Evidence
- Commits: 5cd2698eebfaf909ce9b1d688ab48909cac6a3d6, c69ccc0ca2c75a273daf8357cf77548a18e24e7c, 721a2fc5c1201971b53a8d4cfe048d349b4c15b6, 9925b1a5b7009c794911c7133824cb5f9e9aba46
- Tests: go test -count=1 -race -tags test_dep ./tools/canary/..., go test -count=1 -tags 'test_dep integration' ./tests/ -run '^TestTestpilotCanaryLifecycle$', make canary-build, GOLANGCI_LINT_BASE_REV=HEAD make lint-code-fast
- PRs: