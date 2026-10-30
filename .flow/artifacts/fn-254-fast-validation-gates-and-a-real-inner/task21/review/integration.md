**Scope read:** four byte-identical integration test files moved under each crate's `tests/main/` and registered with `#[path]` modules, plus a README selector note and task evidence. No library code, Cargo manifests, nextest config, or xtask gate code changed.

**Integration verification (axis: changed code meeting unchanged code):**

- **Retired binary names have no live consumers.** A repo-wide search for `--test reach|retained|catalog_admission|fixture_composition`, nextest `binary(...)` filters, and `tests/<name>.rs` paths finds only two historical task files. The nextest override at `model/umpire/.config/nextest.toml:13` selects `binary(=main)` only for the tlc/apalache/tla packages and `binary(=fixtures)` for umpire-fixtures, so the JVM group is unaffected. The xtask runner builds `-E` expressions from package/test predicates, not from these binary names.
- **No process-global state crosses the new module boundary.** `catalog_admission.rs:10` `STEPS` is a private static incremented only by `over::Model::user` in that module; `fixture_composition.rs:37` `CALLS` is `pub(crate)` inside the nested `composed` module and read only by its own test; `retained.rs:11-12` are thread-locals. The `fixture!` macro expansion (`dev/testing/src/lib.rs:239-294`) defines only module-local functions, no registry, so the `umpire-testing` main's "without process-global fixtures" contract still holds. The `#[machine]` macro emits no `inventory`/`linkme`/`ctor` registrations.
- **No relative-path or nested-module breakage.** None of the moved files use `include_str!`, `file!()`, `CARGO_MANIFEST_DIR`, or nested `mod x;` declarations, so the change from mod-rs root to `#[path]` module resolution is inert. Inner `#![forbid(unsafe_code)]` attributes remain legal at module-file top.
- **Affected execution confirms the mapping.** `affected-run.log` shows 95 tests across the 4 binaries with every moved test present under its new `module::name`, and the negative controls (`retired-target.log` exit 101, `missing-case` exit 4) prove the old selector no longer silently matches nothing.
- **Retained-target contracts are concrete.** `identities.json` gives per-target dispositions naming the actual isolation reason (trybuild UI dirs, OUT_DIR registries, `current_exe` child runners, incompatible `crate::CatalogContext` aliases, the ignored Kettle live importer).

**FYI (non-blocking):**
- The README link `../../.flow/artifacts/.../task21/identities.json` from `model/umpire/README.md` resolves to the repo root correctly, but it ties user docs to a task-numbered evidence file. Fine for now; it will go stale if artifacts are ever restructured.
- The forced-build before/after timings are explicitly not causal (before also rebuilt feature-dependent deps). The acceptance criterion permits "clearly explained unavailable attribution", and the summary does explain it.

No findings survive at P0–P2.

## Requirements coverage
| R-ID | Status | Evidence |
|------|--------|----------|
| R4 | met | Four residual integration targets consolidated into existing `main` binaries (39→35 workspace integration targets); 393 scoped identities including the one ignored retain a one-to-one mapping per `identities.json`; affected run passes 95/95; JVM test-group filter unaffected. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>