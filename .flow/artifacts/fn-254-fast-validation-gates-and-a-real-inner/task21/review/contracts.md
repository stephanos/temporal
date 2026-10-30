**Review scope:** four byte-identical test-file relocations into existing `tests/main.rs` targets, four `#[path]` module registrations, a README section, and the task's evidence bundle. I verified the moved files against their new crate context, the recorded per-target contracts against the code, selector usage across gate/nextest/xtask, and the README's claims about retained targets.

**What checks out**

- The four moved files declare only module-private state. `reach.rs` uses immutable `&'static [Edge]`; `catalog_admission.rs` has a private `STEPS` atomic touched only by its own `over::user` handler; `fixture_composition.rs` scopes `CALLS` to `composed` and reads it from one test; `retained.rs` uses `thread_local!` cells reset per test. The contracts in `identities.json` lines 6740, 6747, 6807 and 6837 describe exactly this.
- None of the moved files use `insta`, `include_str!`, `file!()`, or path-relative submodules, so relocation changes no snapshot or fixture path. The `fixture!` macro expands to module-local functions only.
- No gate, nextest override, or xtask filter names the retired binaries. The only `binary(=...)` filters are for `main`/`fixtures` targets that still exist.
- README claims about retained targets hold: catalog `tests/derive_catalog.rs`, `message_map.rs` and `response_checks.rs` each declare a different crate-root `CatalogContext` alias; `kettle/tests/live_snapshots.rs` carries the `#[ignore]`; `testing/tests/native_backend.rs` spawns `current_exe`.
- Evidence is internally consistent: 20 relocated tests match the per-module counts in `affected-run.log`, 375 + 18 = 393 identities, 30 scoped target dispositions, negative controls exit 101 and 4 as the logs show.
- The .4 pin had kept `reach` and `catalog_admission` separate with "static-case" and "process-global STEPS" reasons. The spec for .21 explicitly asks for a fresh classification naming the actual global state, and the new contracts justify the override, so this is not re-litigation.

**FYI (non-blocking)**

- **Severity:** P3 · **Confidence:** 75 · **Classification:** introduced · **File:Line:** `model/umpire/dev/testing/tests/main.rs:1` · **R-IDs:** [R4]
  **Problem:** The target doc still says "without process-global fixtures", while the merged `fixture_composition` module now contributes a process-wide `static CALLS: AtomicUsize`. It is module-scoped and consumed by one test, so there is no failing scenario, but the one-line contract no longer literally describes the binary.
  **Suggestion:** Either leave as is, or narrow the doc to "without child-process or environment fixtures", which is what the retained `consumers`/`native_backend` targets actually isolate.

- **Severity:** P3 · **Confidence:** 100 · **Classification:** introduced · **File:Line:** `model/umpire/README.md:46` · **R-IDs:** []
  **Problem:** A product README now links into `.flow/artifacts/.../task21/identities.json`. The link resolves today, but it binds user-facing docs to pipeline evidence that may be pruned under the closed-spec cleanup rules.
  **Suggestion:** Keep the contract summary in the README prose and drop the artifact link, or point at a durable design doc.

- **Severity:** P3 · **Confidence:** 100 · **Classification:** introduced · **File:Line:** `.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task21/inventory-after.json:1` · **R-IDs:** []
  **Problem:** The file is empty (sha of the empty string in `proof-index.json`). The summary and `concurrent_delta` field explain why, so this is a content note only.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R4 | met | 393 → 393 libtest identities with ignored 1 → 1 and a checked bijection in `identities.json`; four fewer integration link targets; JVM group unchanged in `.config/nextest.toml`. Link-time savings are recorded as target counts, with the timing explicitly labelled non-causal. |
| R2, R3, R5, R8 | deferred | Owned by other tasks in fn-254; not in this task's scope. |

Unaddressed R-IDs: []

Suppressed findings: 0
Classification counts: 3 introduced, 0 pre_existing.
Protected-path filter: 0

```json
{"classification_counts":{"introduced":3,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>