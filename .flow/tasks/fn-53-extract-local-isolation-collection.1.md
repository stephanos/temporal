---
satisfies: [R1, R2, R3, R4, R5]
---
# fn-53-extract-local-isolation-collection.1 Extract and test local isolation collection state machine

## Description
Extract the existing collection and pre-probe decision logic into one private concrete module, then delegate the environment's current collection entry points through it. Keep synchronization, context handling, Temporal probing, and receipt construction in the environment.

**Size:** M
**Files:** `tools/umpire/temporal/local/environment.go`, `tools/umpire/temporal/local/isolation.go`, `tools/umpire/temporal/local/isolation_test.go`
**Touches:** [tools/umpire/temporal/local/environment.go, tools/umpire/temporal/local/isolation.go, tools/umpire/temporal/local/isolation_test.go]

### Approach
- Move `isolationCollection` and its transition/decision implementation beside the environment while preserving the existing comments on the environment entry points.
- Initialize the concrete module from `newEnvironment` with the exact prepare, realize, and observe commands plus the operation correlation; do not add a Go interface or an exported seam.
- Keep `environment.mu` as the sole serialization mechanism and call the module only while it is held.
- Return a closed internal failed/canceled/ready decision to `Isolate`; keep context and isolation-command precedence, `executionProbe.Verify`, and all receipt construction in `environment.go`.
- Preserve the exact three collection error strings, permanent poisoning, mutation ordering, and failure-over-cancellation precedence.
- Add table-driven module tests using `require` for valid transitions, each invalid transition, invalid-then-valid sequences, zero/one/multiple counts, open/closed inputs, repeated decisions, and failure conditions combined with incomplete inputs.
- Add environment-level table tests for nil and canceled contexts, unsupported isolation commands, missing and failing probes, cancellation during probing, and crossed inputs. Assert exact receipts, probe calls, and whether the one-shot decision was consumed.

### Investigation targets
**Required** (read before coding):
- `tools/umpire/temporal/local/environment.go:168-184` — current collection state and adjacent probe seam
- `tools/umpire/temporal/local/environment.go:375-470` — record, close, decision, probe, and receipt precedence to preserve
- `tools/umpire/temporal/local/lifecycle_test.go:140-198` — existing closed-collection lifecycle coverage
- `tools/umpire/temporal/local/attached_test.go:156-190` — attached-authority isolation regression path

**Optional** (reference as needed):
- `tools/umpire/temporal/local/environment.go:186-222` — SDK execution probe behavior that remains outside the module
- `tools/umpire/runtime/README.md:52-74` — documented local-authority ownership that should remain accurate

### Key context
- The state module is an internal in-process seam, not an adapter or extension point.
- Failure from invalidation or a count above one must dominate open or incomplete collection cancellation.
- Do not replace existing lifecycle/attached tests; add tests at the extracted module interface and environment-level orchestration seam, then retain the integration-level regression checks.
- Preserve existing comments and all observable diagnostics. No documentation edit is expected if ownership remains unchanged.
- Use no new third-party dependencies.
## Acceptance
- [ ] R1-R4 are implemented without changing the existing environment caller surface or adding an exported abstraction.
- [ ] Table-driven module tests cover wrong command/correlation, duplicate and post-close operations, permanent invalidation, repeated decision, zero/one/multiple counts, missing records, open/closed states, and failure-plus-incomplete combinations with failure precedence.
- [ ] Environment-level tests cover nil and canceled contexts, unsupported isolation commands, missing and failing probes, cancellation during probing, and crossed inputs; they assert probe-call counts, one-shot decision consumption, and exact receipt status, code, facts, and correlations.
- [ ] Existing lifecycle and attached-authority behavior remains unchanged.
- [ ] Existing comments are preserved, and `tools/umpire/runtime/README.md` is confirmed accurate or updated only if the ownership statement became false.
- [ ] `go test -tags test_dep ./tools/umpire/temporal/local` passes.
- [ ] `make fmt-imports` passes; `make lint-code` runs globally and either passes or matches the exact pre-change baseline with zero task-scoped lint findings.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
