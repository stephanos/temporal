---
satisfies: [R6, R7]
---
# fn-27-hermetic-ci-execution-and-qualification.6 Expose the CI qualification command and isolated workflow

## Description
Implement R6/R7's single user-facing command, repository-root Make target, and manual GitHub Actions closure.

**Size:** M
**Files:** `tools/umpire/cmd/umpire-qualify-ci/**`, `Makefile`, `.github/workflows/umpire-ci-qualification.yml`
**Touches:** [tools/umpire/cmd/umpire-qualify-ci/**, Makefile, .github/workflows/umpire-ci-qualification.yml]

### Approach

- Implement the exact four-flag grammar, fixed sibling resolution, ordered summary/error schemas, statuses 0/1/2, and post-publication reporting behavior from the parent contract.
- Add only the root `umpire-qualify-ci` target with required-variable checks; build the fixed command/checker/profile sibling closure and pass no ambient options.
- Add one `workflow_dispatch`-only workflow with explicit read-only permissions, no environment/secrets/OIDC/cache, full-SHA action pins, one fixed runner, ref concurrency cancellation, and a 30-minute timeout.
- Use the fixed checked-in CI set/pilot evidence, derive the exact run identity, require runner-temp output, upload an existing bounded result for seven days under `always()`, then preserve status-2 job failure.
- Add static workflow tests for triggers, permissions, pins, paths, cache absence, command arguments, timeout, retention, and no default/release dependency edges.

### Investigation targets

**Required** (read before coding):
- `.flow/tasks/fn-19-bounded-local-temporal-execution-and.8.md` — root command/status conventions
- `.flow/tasks/fn-20-local-execution-semantic-conformance.6.md` — fixed sibling and root target conventions
- `.flow/tasks/fn-26-local-qualification-receipts-and-staged.5.md` — qualification CLI/reporting contract
- `.github/workflows/docker-build-manual.yml` — current manual workflow shape
- `.github/workflows/run-tests.yml` — current explicit permissions/concurrency style
- `Makefile` — repository-root Umpire target section

### Acceptance

- [ ] Direct/root bytes, exit statuses, required arguments, sibling checks, and publication/reporting booleans match the parent contract.
- [ ] Workflow policy tests prove manual-only isolation, least privilege, immutable pins, no secret/OIDC/cache/default/release coupling, runner-temp output, and bounded upload.
- [ ] Status 2 uploads inspectable evidence then fails; status 1 never masquerades as a qualified result.
- [ ] No Makefile below repository root is added or modified.

## Acceptance
- [ ] R6/R7 CLI, root Make UX, and isolated manual workflow are complete.
- [ ] Direct/root/static-workflow tests cover success, valid non-success, tooling failure, cancellation, and reporting failure.
- [ ] Existing Make/workflow comments are preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
