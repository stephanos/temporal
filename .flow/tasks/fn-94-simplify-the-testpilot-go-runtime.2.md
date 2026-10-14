---
satisfies: [R11, R10]
---
# fn-94-simplify-the-testpilot-go-runtime.2 Identity goldens and the R10 baseline

## Description
Adds the byte pins every later lane is checked against, before any lane that could move them lands: `BindingFingerprint` bytes, the Driver catalog identity, and the delivery route wire bytes. Also records the R10 measurement baseline and the passing live identity count R7 compares against.

**Size:** S
**Files:** a new golden test in the facade package (for example `common/testing/testpilot/identity_golden_test.go`), a golden test beside `common/testing/testpilot/temporal/catalog.go` for the Driver catalog identity, a route-bytes golden in `common/testing/testpilot/temporal/internal/delivery/codec_test.go`
**Touches:** [common/testing/testpilot/identity_golden_test.go, common/testing/testpilot/temporal/catalog_test.go, common/testing/testpilot/temporal/internal/delivery/codec_test.go]

### Approach
- `BindingFingerprint`: fingerprint a fixed `ProfileSpec` with several bindings and compare to a literal hex string. Cross-check the literal against the `bindings` hash in the pinned control Run record (`tools/umpire/replay/testdata`) where the same Profile applies.
- Driver catalog identity: compute `NewWorkflowServiceCatalog` (`temporal/catalog.go:17`) and compare `Identity()` to a literal; cross-check against the pinned record's `catalog` hash. Comment that only a wire task may change it, together with `make umpire-rerecord-pinned-runs`.
- Route wire bytes: marshal a fixed binding through `delivery/codec.go`'s encoder and compare to literal JSON bytes (not `protojson`, whose output is unstable).
- Record in the receipt: the four Measurement numbers (spec §Measurement) at the current HEAD, and the passing live identity count from `make umpire-check-live-tests` or `make umpire-check-regression` (or "not run: no cluster" with the reason, in which case fn-94.17 records it).

### Investigation targets
**Required:**
- `common/testing/testpilot/profile.go:127-181` — `BindingFingerprint`
- `common/testing/testpilot/temporal/catalog.go:17-40` — Driver catalog
- `common/testing/testpilot/internal/ir/catalog.go:216-253` — identity hashing
- `common/testing/testpilot/temporal/internal/delivery/codec.go:25-60` — route binding JSON
**Optional:**
- `common/testing/testpilot/prepare_test.go:113-191` — existing relational fingerprint tests
- `tools/umpire/replay/subject.go:126` — how pinned records carry bindings/catalog

### Quick commands
```sh
go test -tags test_dep ./common/testing/testpilot/ ./common/testing/testpilot/temporal/ ./common/testing/testpilot/temporal/internal/delivery/
make lint-code-fast
```

## Acceptance
- [ ] Three goldens exist (fingerprint, catalog identity, route bytes), each a literal compared byte-for-byte, and pass at HEAD.
- [ ] The catalog golden's comment names `make umpire-rerecord-pinned-runs` as the companion of any change.
- [ ] The receipt records the four baseline measurements and the live identity count (or why it was not run).


## Done summary
Added three byte-for-byte identity goldens: `BindingFingerprint` (common/testing/testpilot/identity_golden_test.go, with and without configuration), the Driver catalog identity (common/testing/testpilot/temporal/catalog_test.go, whose comment names `make umpire-rerecord-pinned-runs` as the companion of any change), and the delivery route wire bytes for the workflow and nexus kinds (delivery/codec_test.go). The unconfigured fingerprint literal (5433bebb…c629) and the catalog literal (95533e4d…7eb4) equal the `bindings` and `catalog` hashes in the pinned control Run tools/umpire/replay/testdata/nexusCallerControl-forgedCompletion-run.json. Each golden was run red against a placeholder first.

R10 baseline, measured with the spec's Measurement commands on a `git archive` of HEAD `ea93a2bc8b` (the pre-task commit; the task adds only test files): production 19,349; tests 20,572; live tests 2,399; `.proto` 1,414. The task's own commit adds 103 test lines.

Live identity count for R7: 45 passing identities, 0 failing (unique `--- PASS:` names, subtests included, counted as `umpire-check-live-tests` counts them). Source: the Go step of `make umpire-check-live-tests` run directly at `dd3b7bf278`. The target's `lake build umpire-explore umpire-replay-bridge` prelude was skipped, and the existing model/.lake binaries were used, so a concurrent fn-88.6 edit under model/Umpire/Search could not reach this run.

baseline: green (focused Quick commands, pre-edit)
stage: impl-review - ran [codex fan-out rid af42510a0d9247a594e6b0901bd0538f: correctness/contracts/integration all SHIP..SHIP]
## Evidence
- Commits: dd3b7bf2789bff700d039921e3cd880e3c911aec
- Tests: go test -tags test_dep ./common/testing/testpilot/ ./common/testing/testpilot/temporal/ ./common/testing/testpilot/temporal/internal/delivery/, make lint-code-fast, go test -v -count=1 -timeout 30m -tags 'test_dep integration' ./tests -run '^TestTestpilot' (the Go step of make umpire-check-live-tests; 45 passing identities, 0 failing)
- PRs: