**Review scope.** The code change is four byte-identical file relocations into each crate's `tests/main/` directory, four `#[path]` module registrations, and a README paragraph. Everything else is evidence under the task's artifact directory. I read the diff, the four moved files, the four `main.rs` roots, the identities inventory, the selection-refusal controls, and the verification logs, and grepped the xtask gate, nextest config and docs for any selector naming a retired target.

**Correctness of the moved code.** Each moved file carries only `//!` docs and `#![forbid(unsafe_code)]` as inner attributes, which are legal inside a non-root module. None declares a file-backed `mod`, uses `#[path]`, `crate::` paths, `#[cfg]`, `#[ignore]`, `current_exe`, or environment mutation, so nesting under a module changes nothing but the test-name prefix. The runtime root's own `use umpire_runtime::protocol;` does not collide with the import inside the moved Reach module. The affected run shows all 95 tests in the four `main` binaries passing, including the 20 relocated cases (4 retained, 9 reach, 5 catalog_admission, 2 fixture_composition), which matches the claimed changed-identity count.

**Process-global state.** The three statics that moved (thread-local `WORLDS`/`STEPS` in the suite module, module-private `STEPS` atomic in native, module-scoped `CALLS` atomic in testing) are not referenced by any other module in their new binaries, so no new cross-test coupling is introduced. The native `STEPS` atomic was already shared by the same five tests in one binary before this change.

**Selectors.** No gate code, nextest filter, Makefile or doc outside `.flow/tasks` names `--test retained|reach|catalog_admission|fixture_composition`, so the "no selector changes needed" claim holds. The README's new module selectors are correct, and its relative link to the inventory resolves from `model/umpire/` to the repo-root `.flow/artifacts/...` path. The two negative controls (retired target exits 101, missing case exits 4) are recorded verbatim.

**Evidence caveats, correctly disclosed.** The workspace-wide after inventory is an empty file because an unrelated concurrent fixture failed to compile; the scoped 11-package capture is the accepted comparison and the failure log is retained. The forced-build timing explicitly disclaims a causal speedup. Temporal .7 and .12 are untouched per the changed-file list.

## Findings

No P0/P1/P2 findings.

- **Severity**: P3
- **Confidence**: 100
- **Classification**: introduced
- **File:Line**: `.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task21/summary.md:3`
- **R-IDs**: []
- **Problem**: Several numbers are fused to the preceding word ("fails101", "fails4", "were9.854s", "and3.306s", "was3.683s", "all11"), which makes the evidence prose harder to read and grep.
- **Suggestion**: Insert the missing spaces. FYI only; it does not affect the verdict.

## Requirements coverage

The task file has no R-ID bullets; its frontmatter claims R4 from the parent spec.

| R-ID | Status | Evidence |
|------|--------|----------|
| R4 | partial | This task's slice (integration-binary consolidation with checked identity bijection, 393→393 identities, 1→1 ignored) is met; the debuginfo profile and JVM concurrency group parts of R4 belong to other tasks in the spec and were not in scope here. |

Unaddressed R-IDs: []

Suppressed findings: 1 (pre-existing possible intra-binary `STEPS` race in the native catalog-admission module under thread-parallel `cargo test`, confidence 50, untouched by this diff).
Classification counts: 1 introduced, 0 pre_existing.

```json
{"suppressed_count":{"50":1},"classification_counts":{"introduced":1,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>