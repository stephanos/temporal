---
satisfies: [R6, R11]
---
# fn-154-gomad-simpler-virtual-network-with.1 Add explicit per-direction byte and virtual stall limits

## Description
Add explicit per-direction byte and virtual stall limits (R6, R11). This task owns the named surface; root owns admission, integration and lifecycle.

**Size:** M
**Files:** `tools/gomad3sim/spec.go`, `tools/gomad3sim/spec_test.go`, `tools/gomad3sim/types.go`, `tools/gomad3sim/record.go`, `tools/gomad3sim/record_test.go`
**Touches:** [tools/gomad3sim/spec.go, tools/gomad3sim/spec_test.go, tools/gomad3sim/types.go, tools/gomad3sim/record.go, tools/gomad3sim/record_test.go]

### Approach

- Extend Limits and DefaultLimits at spec.go:63/130 with NetworkConnectionBytes and NetworkStallNanos. Name the 4 MiB per-direction and 10-second defaults; preserve all unrelated limits. Remove the old delivery-count surface only with the history migration task, keeping this checkpoint non-released.
- Extend validateLimits at spec.go:314 and strict decoding with positive/range/signed-duration/strict-threshold checks. Validate actual expiry arithmetic at its later mutation owner rather than claiming config alone proves every future clock addition.
- Carry required fields into record/replay validation and establish the versioned migration inventory. Keep identities coherent at final cutover; intermediate checkpoints are not qualified formats.
- Pin other DefaultLimits fields and old validation precedence. Add absent/zero/negative/maximum/overflow and helper-default controls.

### Investigation targets

**Required:**

- `tools/gomad3sim/spec.go:63`
- `tools/gomad3sim/spec_test.go:12`
- `tools/gomad3sim/types.go:9`
- `tools/gomad3sim/record.go:197`
- `tools/gomad3sim/record_test.go:12`

### Quick commands

```bash
go test -tags test_dep ./tools/gomad3sim -run 'Test(ValidateSpec|DecodeSpec|SpecJSONFieldNames|ClusterRecord)'
```

Patched-runtime selectors require the supported candidate and documented toolchain setup. New selectors named in acceptance are deliverables, not existing-test claims. Keep a missing required gate open; bind each result to the tested source and tool inputs.

## Acceptance
- [ ] Explicit defaults and both new scalar fields have positive, omitted/zero and numeric-overflow regression coverage.
- [ ] Existing unrelated limits and strict unknown/trailing JSON rejection retain their behavior pin.
- [ ] Record/replay limit validation consumes the new fields without granting old records compatibility.
- [ ] Focused source tests pass; the handover names staged identity work and remaining runtime acceptance.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
