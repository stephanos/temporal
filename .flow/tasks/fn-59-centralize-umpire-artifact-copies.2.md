---
satisfies: [R1, R2, R3, R4]
---
# fn-59-centralize-umpire-artifact-copies.2 Adopt shared copies at artifact and runtime boundaries

## Description
Make artifact admission and runtime output consume Task `.1`'s copy authority, remove their duplicated artifact-model traversal, and prove R1-R4 at the existing caller boundaries. Preserve artifact-specific admitted-closure composition and every observable contract.

**Size:** M
**Files:** `tools/umpire/artifact/set.go`, `tools/umpire/artifact/set_test.go`, `tools/umpire/artifact/set_execution_test.go`, `tools/umpire/runtime/engine.go`, `tools/umpire/runtime/request_test.go`
**Touches:** [tools/umpire/artifact/set.go, tools/umpire/artifact/set_test.go, tools/umpire/artifact/set_execution_test.go, tools/umpire/runtime/engine.go, tools/umpire/runtime/request_test.go]

### Approach

- Replace artifact-model clone calls in admitted-set construction and runtime Output construction/accessors with the root copy operations from Task `.1`.
- Keep set-member bytes, manifest bytes, admitted-set composition, and artifact-only closure wrappers in artifact; they are not part of the shared artifact-model representation.
- Remove redundant nested clone functions and now-unused imports only after all callers delegate to the internal authority.
- Extend caller-level tests for copy-on-input, repeated copy-on-output, nil/empty preservation, schema-valid nested mutation isolation, admitted Raw Evidence scalar values, and retained original encoded bytes.
- Preserve validation/admission ordering, error codes and wrapping, canonical bytes/checksums, all existing comments, and the unchanged package README contracts.

### Investigation targets

**Required** (read before coding):
- `tools/umpire/artifact/set.go:118-305,700-887` — admission closure, retained bytes, and current clone callers
- `tools/umpire/runtime/engine.go:10-35,75-133` — Output ownership contract and duplicate clone implementation
- `tools/umpire/internal/artifactv2/evidence.go:287-335` — admitted Raw Evidence field-value domain
- `tools/umpire/artifact/set_test.go:107-149` — admitted-set mutation isolation
- `tools/umpire/artifact/set_execution_test.go:12-88` — execution closure and exact-byte retention
- `tools/umpire/runtime/request_test.go:41-103` — runtime request/output value isolation patterns

**Optional** (reference as needed):
- `.flow/memory/bug/integration/behavior-neutral-refactors-must-not-2026-09-04.md:16-25` — hardening must remain separate from refactoring

### Key context

Schema-valid inputs and outputs must be byte- and value-identical, and invalid admitted inputs must fail before the same observable stages with the same classifications. `NewOutput` does not validate arbitrary programmatic `RawEvidenceField.Value` composites today; those invalid values remain outside R2 and must not acquire new validation or generic-copy behavior in this refactor. Do not repair unrelated validation, naming, or lint concerns while touching these files; in particular, do not absorb the existing runtime error-lint waiver into this cleanup.

### Acceptance

- [ ] R1 is complete: artifact and runtime retain no independent traversal of artifact-model nested fields, while artifact-specific set/closure copying remains local.
- [ ] R2 is proven at both caller boundaries for schema-valid values: source mutation after construction and mutation of any accessor result cannot alter retained state or a later accessor result; nil/empty, zero values, and admitted nil/Boolean/string/canonical-`json.Number` Evidence fields remain exact.
- [ ] Invalid composite or custom `RawEvidenceField.Value` values remain outside the isolation contract and gain no new validation, normalization, or generic-copy behavior.
- [ ] R3 is proven by unchanged admitted values, original encoded bytes, checksums, canonical fixtures, error classifications, and failure precedence for valid and invalid admitted inputs.
- [ ] R4 is satisfied with unchanged public signatures, schemas, generated outputs, dependencies, READMEs, and preserved existing comments.
- [ ] `go test -count=1 -tags test_dep ./tools/umpire/internal/artifactv2 ./tools/umpire/artifact ./tools/umpire/runtime` passes.
- [ ] `make umpire-check-regression` retains the complete `^TestUmpire` live-test gate and passes under its approved inherited-failure policy.
- [ ] `make fmt-imports` and `make lint-code` are run; any inherited lint failure is recorded against the pre-edit baseline and the task introduces zero scoped findings.
## Acceptance
- [ ] Artifact and runtime delegate to the shared copy authority with unchanged immutable, byte, checksum, diagnostic, and comment contracts.
- [ ] Focused tests, aggregate Umpire regression/live tests, formatting, and code lint complete with no task-scoped regressions.

## Done summary
Artifact admission projections and runtime Output now delegate artifact-model ownership to the shared artifactv2 root copy operations, while set-member and manifest byte copying remains local. Caller-boundary tests cover constructor and repeated-access isolation, nil/empty/zero preservation, admitted Raw Evidence scalars, unsupported composites outside the contract, and exact retained encoded bytes.

Baseline: the exact focused Go command was inherited-red on the macOS /var symlink temp path and passed with a physical workspace TMPDIR; make umpire-check-regression and make fmt-imports passed. make lint-code was inherited-red before and after the change only on user-owned tools/umpire1/monitor_test.go (undefined v1), with zero task-scoped findings.

Concurrent HEAD exception: implementation evidence is scoped to 6c570c606; later conductor-owned commit 7b51e5ffe is preserved unchanged and excluded.

stage: impl-review - ran [2026-09-04T18:02:26Z..2026-09-04T18:04:48Z]
## Evidence
- Commits: 6c570c6068da641cafa606a448c7accc0cfc7034
- Tests: baseline: red (go test -count=1 -tags test_dep ./tools/umpire/internal/artifactv2 ./tools/umpire/artifact ./tools/umpire/runtime failed pre-edit on inherited macOS /var symlink temp path); green with physical workspace TMPDIR, TMPDIR=<physical-workspace-temp> go test -count=1 -tags test_dep ./tools/umpire/internal/artifactv2 ./tools/umpire/artifact ./tools/umpire/runtime, make umpire-check-regression, make fmt-imports, make lint-code (inherited red before and after: tools/umpire1/monitor_test.go undefined v1; zero task-scoped findings), impl-review codex: SHIP, 0 findings
- PRs: