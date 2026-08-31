# fn-27-hermetic-ci-execution-and-qualification.6 Expose the ordinary pinned CI test command

## Description
Wire the portability proof into the ordinary pinned Go test command and aggregate repository gate. Workflow configuration delegates to that command with read-only permissions and pinned actions; it does not enumerate semantic checks, accept secrets/OIDC, or expose a second Umpire CI runner or Claim Assessment command.

## Acceptance
- [ ] The ordinary test command is the sole public CI execution surface.
- [ ] Workflow actions and toolchains are pinned with minimal read-only permissions.
- [ ] No semantic flags, profile selector, credentials, cache authority, or custom policy command is added.

## Done summary
Added one read-only, pinned Umpire workflow that delegates to the ordinary `TestHermeticCIPortability` Go command also exposed through `umpire-check-regression`; mise is fixed by version and extracted-binary SHA-256, and incomplete workflow path filters were removed. Added a typed workflow regression test that locks triggers, permissions, concurrency, runner, pinned actions/toolchain inputs, and exact delegation through the aggregate Make target.

Strict TDD established RED while the workflow was absent and again for the review fixes before both GREEN transitions. All four canonical Quick commands, workflow action lint, targeted YAML formatting, focused workflow regression, and Go vet pass; non-mutating `make lint-code` reports zero task-diff issues and retains the inherited `tools/umpire/runtime/errors.go:60` `et:unw+` failure.

stage: impl-review - ran [2026-08-31T21:08:57.096399Z..2026-08-31T21:20:48.146887Z]
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: b31d95639b69ecdd8984e3071853d78c8584bd7d, 2817a1a89bfc27258dff942fdb94526c51355a93, 5937ed8a0cc15f43cb3545c08e96efe216741928
- Tests: STRICT_RED: mise exec -- go test -count=1 -tags test_dep ./tools/umpire/regression -run '^TestHermeticCIWorkflowDelegatesToOrdinaryPinnedTest$' (failed before implementation: workflow absent), STRICT_RED: mise exec -- go test -count=1 -tags test_dep ./tools/umpire/regression -run '^TestHermeticCIWorkflowDelegatesToOrdinaryPinnedTest$' (failed before review fix: obsolete path-filter contract), cd model && mise exec -- lake build Temporal.Tool.RunEvaluationTests, mise exec -- go test -count=1 ./tools/umpire/runtime/... ./tools/umpire/runevaluation/... ./tools/umpire/temporal/nexus/..., mise exec -- go test -count=1 ./tools/umpire/temporal/nexus/... -run '^TestHermeticCIPortability$', mise exec -- make umpire-check-regression, mise exec -- go test -count=1 -tags test_dep ./tools/umpire/regression -run '^TestHermeticCIWorkflowDelegatesToOrdinaryPinnedTest$', mise exec -- make lint-actions, .bin/yamlfmt-v0.16.0 -conf .github/.yamlfmt -lint .github/workflows/umpire.yml, mise exec -- go vet -tags test_dep ./tools/umpire/regression, git diff --check, GOLANGCI_LINT_BASE_REV=92db3f787800685d32e74c2b56a1a323673bcc25 GOLANGCI_LINT_FIX=false mise exec -- make lint-code (0 task-diff issues; inherited tools/umpire/runtime/errors.go:60 et:unw+ failure)
- PRs:
