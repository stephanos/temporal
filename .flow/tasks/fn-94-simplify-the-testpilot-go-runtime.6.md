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
- [ ] `execution.Profile` no longer mirrors `ProfileSpec` field by field.
- [ ] Corpus and fingerprint golden unchanged; tests and lint pass.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
