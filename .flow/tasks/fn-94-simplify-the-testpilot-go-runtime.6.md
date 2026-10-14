---
satisfies: [R4]
---
# fn-94-simplify-the-testpilot-go-runtime.6 Validate once in the core: bindings, one Profile clone, no ProfileSpec mirror

## Description
Lane C for the facade and `execution`: environment bindings are validated only by `BindingFingerprint`, `Prepare` clones the Profile once, and `execution.Profile` stops mirroring `ProfileSpec`.

**Size:** M
**Files:** `common/testing/testpilot/profile.go`, `common/testing/testpilot/prepare.go`, `common/testing/testpilot/internal/execution/prepare.go` (`bindPolicy`, `bindRolePolicy`), `common/testing/testpilot/internal/execution/program.go` (`Profile`), tests
**Touches:** [common/testing/testpilot/profile.go, common/testing/testpilot/prepare.go, common/testing/testpilot/*_test.go, common/testing/testpilot/internal/execution/prepare.go, common/testing/testpilot/internal/execution/program.go, common/testing/testpilot/internal/execution/*_test.go]

### Approach
- Confirm `execution.Prepare`'s only caller is facade `Prepare` (`prepare.go:38`); then drop the binding re-validation loop in `bindPolicy` (`execution/prepare.go:150-169`). Keep `BindingFingerprint` (`profile.go:127-181`) as the validator; its `profile.environment_bindings` paths stay (six cases in `preparation_error_test.go:175-182`).
- Replace `Snapshot().Snapshot()` (`prepare.go:28`) and the re-clones in `bindPolicy`/`bindRolePolicy` (`execution/prepare.go:171-176,213`) with one clone in `Prepare`. Add a test: mutate the caller's `ProfileSpec` after `Prepare`; the prepared Case and its fingerprint are unchanged.
- One sorted-unique-IDs helper for the two loops in `profile.go:140-151,194-204`.
- `execution.Profile` (`program.go:18-29`, built at `prepare.go:37`): hold the cloned spec's values directly instead of a field-by-field mirror, without importing the facade (use the `contract` package if a shared type is needed).
- Remove facade tests that exactly duplicate `execution` tests of the same rejection; keep the facade copy.

### Investigation targets
**Required:**
- `common/testing/testpilot/prepare.go:20-60`
- `common/testing/testpilot/profile.go:100-210`
- `common/testing/testpilot/internal/execution/prepare.go:110-240`
- `common/testing/testpilot/preparation_error_test.go:170-190`

### Quick commands
```sh
go test -race -tags test_dep ./common/testing/testpilot/...
make umpire-check-case-runtime-conformance
make lint-code-fast
```

## Acceptance
- [ ] Bindings are validated only in `BindingFingerprint`; pinned paths unchanged.
- [ ] One Profile clone per `Prepare`; the caller-mutation test passes.
- [ ] `execution.Profile` keeps only what execution reads, filled by exactly one explicit copy from `ProfileSpec` at the facade boundary in `Prepare` (owner decision 2026-09-27: `ProfileSpec`'s public shape stays, `execution` cannot import the facade, and verification tests build `execution.Profile` directly, so one boundary copy is accepted; no second mirror or re-clone exists).
- [ ] Corpus and fingerprint golden unchanged; tests and lint pass.


## Done summary
BindingFingerprint is now the only validator of environment bindings. Prepare takes one Profile snapshot, and admission holds its values without re-cloning them. One `sortedByID` helper admits both the binding and the configuration entries. `execution.Profile` drops `EnvironmentFingerprint`, and `PreparedProgram` holds its limits and instruction defaults directly instead of a mirrored Profile snapshot. `Prepare` makes one commented boundary copy into `execution.Profile`, under the amended AC3 (owner decision 2026-09-27). The Profile interface doc now requires Snapshot to return a spec the caller owns.

`TestPrepareHoldsItsOwnProfileClone` failed first against a shallow Snapshot, then passed with the fix. The execution-side binding rejection test went with its checks. The Profile-mutation lines in `TestPrepareSlotDataflowAndImmutableViews` also went, because the facade test now pins ownership. This change also fixes the gofmt/gci misalignment that fn-94.4 left in `prepare.go`.

stage: impl-review - ran [fan-out rid 3c1d3c2e149441c2bcdfe33db88dc05a NEEDS_WORK on the mirror criterion, refunded after a concurrent HEAD move; after the owner amended AC3, fan-out rid c3ce74f9731e4a158e978475fc28cb26: 3/3 draws SHIP, review base 5bc236723d]
## Evidence
- Commits: d996b7dc53ad9d18b84a28e0b2e468c836134624, cb4b3a34d458d77c0d3547917291f7dd21121df1
- Tests: baseline: green (go test -race -tags test_dep ./common/testing/testpilot/ ./common/testing/testpilot/contract/... ./common/testing/testpilot/internal/...; temporal/** was mid-edit by fn-94.5), go test -race -tags test_dep ./common/testing/testpilot/... (HEAD d996b7dc53: 11 packages ok), make umpire-check-case-runtime-conformance equivalent in a HEAD export + this diff: both generator modes, diff -ru clean on both testdata trees, generator tests and TestCaseRuntimePublicFacadeConformance ok (Lean binaries reused from model/.lake), make lint-code LINT_CODE_TARGETS='./common/testing/testpilot/ ./common/testing/testpilot/internal/execution/' GOLANGCI_LINT_BASE_REV=951c5516e9: 0 issues; gci list -s standard -s default on the facade and execution: clean, go test -race -tags test_dep ./common/testing/testpilot/ after cb4b3a34d4 (comment-only): ok, make lint-code-fast: not observed green in the shared checkout; another task's uncommitted temporal/worker edits fail typecheck; targeted lint-code on the touched packages: 0 issues
- PRs: