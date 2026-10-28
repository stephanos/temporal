Wrapping up: I have what I need. The residual `umpire.local/model` Go imports all sit inside the six compiled caller trees, so the module swap is complete for code; only two READMEs keep stale references. Writing the verdict now.

## Findings

**Finding 1**
- **Severity**: P2
- **Confidence**: 100
- **Classification**: introduced
- **File:Line**: `.flow/artifacts/fn-258-complete-the-resumed-standalone-umpire/consumer/source-adaptation.patch:97`
- **R-IDs**: [R4]
- **Problem**: The missing-sibling guard was added only to the `model-quick` recipe. `make model-check`, `model-update`'s second step and the `gate` path still start with `mise exec -- cargo run --manifest-path model/Cargo.toml`, and `model/Cargo.toml` now carries twenty `../../umpire/...` workspace path dependencies (patch lines 246 to 265). With no sibling checkout, cargo aborts while loading the workspace with its own "failed to read `.../umpire/dev/guest-driver/Cargo.toml`" error before `framework_root()` in `main.rs` (patch line 366) or `framework-root.sh` ever runs. R4 asks that a missing checkout fail "with actionable setup instructions"; the most-used entry point gives cargo's generic manifest error instead.
- **Suggestion**: Put the same `@/bin/bash "$(CURDIR)/model/scripts/framework-root.sh" >/dev/null` line first in the `model-check` recipe (and any other recipe that invokes `$(RUST_MODEL_CARGO)` against `model/Cargo.toml`), or fold it into a shared prerequisite target.

**Finding 2**
- **Severity**: P2
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `.flow/artifacts/fn-258-complete-the-resumed-standalone-umpire/consumer/readiness.json:346`
- **R-IDs**: [R4]
- **Problem**: Task acceptance bullet two requires that `go env GOWORK` and `go list -m -json` show the intended workspace and host module. The readiness record has Cargo metadata counts but no Go workspace output. The only workspace evidence is indirect: the pre-fix bridge log (`readiness-bridge-controls-before-fix.log:95`) proves some `go.work` was in effect, and the earlier pair's local-edit proof names a `go.work` path for consumer 4035bb70, not 3658582. Nothing records which modules that workspace selected or that `github.com/temporalio/umpire/hosts/go` resolved to the sibling path rather than a cache.
- **Suggestion**: Add one readiness log capturing `go env GOWORK`, `go list -m -json github.com/temporalio/umpire/hosts/go umpire.local/temporal` and `cat "$GOWORK"` from the pair checkout, then hash it into the manifest.

**Finding 3**
- **Severity**: P3
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `common/testing/testpilot/README.md:1`
- **R-IDs**: []
- **Problem**: The candidate rewrites every Go import and `go.mod` identity from `umpire.local/model` to the canonical host module, but `common/testing/testpilot/README.md` and `tools/canary/README.md` still document the retired `umpire.local/model` paths, and `tools/umpire/cmd/umpire/main.go` (patch line 1104) now points readers at `../umpire/hosts/go/README.md`, a path outside the repository. The readiness orphans note (`readiness.json:439`) classifies residual strings as archive or fixture material, which these two READMEs are not.
- **Suggestion**: Update the two READMEs in the consumer branch alongside the import rewrite, and name the sibling README by its documented checkout convention rather than a relative path.

## FYI (non-blocking)

- **Consumer protobuf freshness** is not among the readiness checks; `proto-go.sh temporal --check` appears only in the new host README. The spec assigns protobuf freshness to R5 (task 4), so this is deferred by design.
- **mise include above the repo root** (`includes = ["../umpire/dev/xtask/src/gate/tasks.toml"]`, patch line 219) is only exercised through a fake `mise` in the bridge controls. The real full gate is task 4 scope.
- **`UMPIRE_TOOL_ROOT` forwarding** from `main.rs` (patch line 391) is sound: `ToolRoot::local()` in `model/umpire/dev/guest-driver/src/context.rs:18` honors the override and falls back to the compiled framework root, and `admit` canonicalizes and verifies the three configuration files.
- **GOWORK=off for the isolated fixture** (patch line 420) is the right fix: the fixture module has no dependencies and the test asserts the specific "no matching files found" message (`model/dev/xtask/tests/main.rs:193`), so it cannot pass on a workspace error.
- **Snapshot ownership rescoping** keeps coverage: every `FIXTURES` and `READERS` entry carries a `temporal/` or `umpire/` prefix, and `model/dev` and `model/design` contain no `.snap` files, so dropping the `model/` scan root loses nothing.
- **Duplicated Code (heuristic)**: the sibling existence check (`../../umpire` plus `Cargo.toml` and `scripts/xtask.sh`) appears in `framework-root.sh`, `main.rs::framework_root` and `tests/main.rs::framework_root`. Acceptable for a shell/Rust seam, noted only.
- **`umpire.yml` still runs from `model/umpire`**: intentionally inactive pending task 5.

## Requirements coverage

| R-ID | Status | Evidence |
| --- | --- | --- |
| R1 | deferred | Owned by task 1 (closed per MIGRATE.md); not in this task's scope. |
| R2 | deferred | Task 2 (standalone history join); outside task 3 per the focus statement. |
| R3 | deferred | Task 2 standalone portability; consumer here consumes its exported paths. |
| R4 | partial | Sibling Cargo paths and canonical Go identities adapted and compiled against 05473210 (`readiness-054-binding.json`, 16 unit + 7 integration, Go callers compile, no-Rust sentinel contract). Workspace output and one actionable-error path are gaps (Findings 1 and 2); CI activation and cutover are task 5 by the spec's interim-ownership section. |
| R5 | deferred | Task 4 (final-pair fresh gates, protobuf freshness, red-green). |
| R6 | deferred | Task 5 (publication and cutover). |

Unaddressed R-IDs: []

Suppressed findings: 0
Classification counts: 3 introduced, 0 pre_existing.
Protected-path filter: 0

```json
{"classification_counts":{"introduced":3,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>