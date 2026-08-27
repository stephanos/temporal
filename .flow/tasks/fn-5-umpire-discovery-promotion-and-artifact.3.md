---
satisfies: [R1, R2, R7]
---
# fn-5-umpire-discovery-promotion-and-artifact.3 Generate the glossary and machine catalog index

## Description
Project the checked production catalog into reviewable Markdown and canonical JSON for R1/R2/R7.

**Size:** M
**Files:** `tools/common/artifactio/set.go`, `tools/common/artifactio/set_test.go`, `tools/common/artifactio/check.go`, `tools/common/artifactio/check_test.go`, `tools/umpire/internal/generate/catalog/catalog.go`, `tools/umpire/internal/generate/catalog/generate.go`, `tools/umpire/internal/generate/catalog/render.go`, `tools/umpire/internal/generate/catalog/generate_test.go`, `tools/umpire/cmd/umpire-gen-catalog/main.go`, `model/GLOSSARY.md`, `model/Temporal/Tool/Generated/Catalog.json`
**Touches:** [tools/common/artifactio/set.go, tools/common/artifactio/set_test.go, tools/common/artifactio/check.go, tools/common/artifactio/check_test.go, tools/umpire/internal/generate/catalog/**, tools/umpire/cmd/umpire-gen-catalog/main.go, model/GLOSSARY.md, model/Temporal/Tool/Generated/Catalog.json]

### Approach

- Invoke exactly `temporal-model-catalog export` from the known model root and strictly validate the `umpire-semantic-catalog-export/v2` stdout/stderr/exit contract. Reject unsupported versions, missing/extra fields or rows/bindings, duplicates, noncanonical order/bytes, success stderr, nonzero status, and empty/extra stdout.
- Render both outputs from one validated Generated View in canonical identity order.
- Publish the exact two-file set transactionally through `artifactio.Set`; validate candidates before replacement.
- Add a reusable exact candidate-set comparison seam in `artifactio` that shares `Set.Publish` path/containment validation and holds the same artifact-set lock across the complete multi-file read. It rejects symlinked roots or managed components, non-regular managed files, escapes, and permission errors rather than following or weakening them. If interrupted publication state requires recovery, comparison returns a distinct recovery-required error without mutation; publication remains the only recovery owner.
- Add an explicit non-mutating `--check` mode that renders expected bytes in memory and compares the current two files through that locked seam, without temporary regeneration or filesystem mutation.
- Treat Lean output as input authority and never read generated prose back into semantics.

### Investigation targets

**Required:**
- `tools/umpire/internal/generate/regression/generate.go:20-178` — injected inspector, candidate validation, and publication pattern.
- `tools/umpire/internal/generate/regression/generate_test.go:17-99` — deterministic and failure tests.
- `tools/common/artifactio/set.go:16-103` — complete-set publication.
- `model/Temporal/Tool/Inspect.lean:23-67` — stdout/stderr result behavior.
- `tools/umpire/cmd/umpire-gen-regressions/main.go` — thin command boundary.

### Quick command

`go test -count=1 -tags test_dep ./tools/umpire/internal/generate/catalog/...`

## Acceptance
- [ ] Repeated generation yields byte-identical `model/GLOSSARY.md` and catalog JSON.
- [ ] Generation consumes exactly one complete `umpire-semantic-catalog-export/v2` response and rejects unsupported version, incomplete/extra/duplicate/noncanonical content, nonzero exit, success stderr, or partial stdout before rendering.
- [ ] `--check` detects stale or missing outputs by in-memory comparison and performs no writes.
- [ ] Checks reject symlinked roots/components, non-regular files, containment escapes, permission failures, and recovery-required interrupted state without mutation, and cannot observe a mixed set during a concurrent lock-cooperating publication.
- [ ] Every checked catalog entry appears exactly once in both projections with matching identity/kind/digest/reference data.
- [ ] Stable-entry Generated View bindings appear exactly once in catalog JSON with safe fixture paths, unique Generated View keys, and binding identities, without changing catalog Behavior Fingerprint.
- [ ] Malformed output, stderr, nonzero exit, duplicate/stale entries, unsafe paths, and partial publication fail closed.
- [ ] Concurrent/interrupted publication preserves the prior complete pair; concurrent-reader/writer tests prove checks hold the shared set lock for the full read.
- [ ] Go tests use `require` and whole-value comparisons.
- [ ] No generated output becomes a semantic input.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
