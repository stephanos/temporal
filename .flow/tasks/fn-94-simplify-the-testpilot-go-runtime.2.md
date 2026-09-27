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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
