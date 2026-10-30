**Scope and verification note.** I reviewed the full diff and traced every changed call site into unchanged code (process supervisor, `Scratch`, `Refusal` conversion, gate `fetch-apalache` dispatch, the standalone `standalone-tlc-tool` stage, `xtask.sh`, `ToolRoot` admission, `work::spawned`). This session has no shell, so no tests were run. The retained evidence under `.flow/artifacts/.../task16/validation-60f1c762/` lives only in the main checkout and reading it was denied, so the green standalone/real-install claims are taken from the review brief, not re-verified.

## Findings

**1. Standalone TLC stage now pays a second Cargo bootstrap build**
- **Severity**: P2
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `model/umpire/scripts/fetch-tlc.sh:5`, `model/umpire/dev/xtask/src/gate/standalone.rs:418`
- **R-IDs**: [R7]
- **Problem**: The TLC fetch entry point went from a dependency-free bash script to `xtask.sh`, which runs `mise exec -- cargo build --locked -p xtask ... --target-dir $CARGO_TARGET_DIR/xtask-bootstrap`. The standalone gate invokes it through `clean_command` with the standalone's own `CARGO_TARGET_DIR`, so the standalone run now compiles the xtask + guest-driver + `umpire-backend-process` closure a second time into `<target>/xtask-bootstrap` before TLC can be provisioned. The task text says "Download cleanup must not delay .5/.17's validation speedups"; timings are owner-deferred to .8, so this is non-blocking, but it is a new fixed cost in the exact path .8 will measure.
- **Suggestion**: In `standalone.rs` run the already-built gate executable (`env::current_exe()`) with `fetch-tlc <root>` under the clean environment, or let `fetch-tlc.sh` honor a prebuilt tool path, and record the bootstrap delta in .8.

**2. No-op policy branch added to `OwnedProcess::spawn`**
- **Severity**: P2
- **Confidence**: 100
- **Classification**: introduced
- **File:Line**: `model/umpire/backends/process/src/support.rs:182-190`
- **R-IDs**: []
- **Problem**: The new `match policy` dispatches `Checker` to `Running::spawn` and `Orchestration` to `Running::spawn_with_policy`, but `Running::spawn` is literally `spawn_with_policy(.., Checker)` (`syscall.rs:128-141`). Both arms resolve to the same call; the branch adds a code path the change does not need.
- **Suggestion**: Revert to the single `spawn_with_policy(command, directory, stdout, stderr, policy)` call.

**3. Pin literals now live in four places with no drift control for TLC or the mise declarations**
- **Severity**: P3
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `model/umpire/dev/xtask/src/fetch_apalache.rs:16`, `model/umpire/mise-tools/mise.toml:8`, `model/umpire/mise-tools/mise.apalache.toml:7`
- **R-IDs**: []
- **Problem**: The TLC SHA is hard-coded in `install_tlc` separately from `umpire_tlc::JAR_SHA256` and the mise toml/lock. `apalache_pins::ARCHIVE_SHA256` no longer has a code consumer; the archive hash mise actually enforces is the copy in `mise.apalache.toml`. The existing drift test `pinned_installer_and_runner_agree` only checks `apalache_pins.rs`, so a bumped mise declaration with a stale Rust pin (or vice versa) passes tests and fails only at a real install.
- **Suggestion**: Extend that control (or add one in the process crate) to assert the mise toml/lock files contain the URL and checksum constants, and source the TLC literal from one constant.

**4. Missing provider configuration reports "missing pinned jar"**
- **Severity**: P3
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `model/umpire/backends/process/src/tool_provider.rs:108-115`
- **R-IDs**: []
- **Problem**: `command()` reuses `contained()` for the `mise-tools` directory and its five config files, so a root without `mise-tools/mise.apalache.lock` fails with `missing pinned jar <root>/mise-tools/mise.apalache.lock`. Any caller of `Tlc::discover` on a root that has the jar but not the provider config gets a jar diagnostic for a configuration problem.
- **Suggestion**: Use a provider-specific message for the config admission (e.g. "mise jar configuration missing: …").

## FYI (non-blocking)

- **Changed backend contract is real and documented.** `Tlc::discover` / `Apalache::discover` now require `mise` on PATH, the five `mise-tools` files at the root, a `mise where` round-trip per discover, and a UTF-8, colon-free canonical root. All current callers pass the framework root (`model/temporal` tests use `../../umpire`), and the narrowing is recorded in `mise-tools/README.md` and the xtask README. Settled by the task; not re-litigated.
- **`MISE_LOCKED` semantics could not be verified offline.** The README's concurrency argument (locked installs avoid the 2026.10.11 metadata race) rests on that env var being honored by the pinned mise. The `.17` custody path itself is sound: the provider goes through `run_process_cancelled_with_policy` → `OwnedProcess::spawn` → `spawn_with_custody`, so inherited lease/lifecycle descriptors are validated and delegated exactly as for other Orchestration targets, and `work::spawned` ignores non-`tlc2.TLC` commands so mise never inflates checker counts.
- **`admit_mise` calls `installed_jar` twice** (directly and again inside `resolve_cancelled`). Harmless.
- **`model/.gitignore` still lists `/build/tlc/` and `/build/apalache/`** at the model root; stale from the repository-shaped cache era, untouched by this diff.

## Requirements coverage

| R-ID | Status | Evidence |
| --- | --- | --- |
| R7 | met | Jars resolve under `<root>/build/mise-jars/data/installs/...` and config under `<root>/mise-tools`, both from the admitted framework root (`tool_provider.rs:94-118`, `standalone.rs:525`); no Temporal-root path or parent mise configuration is read (`MISE_CEILING_PATHS`, empty global/system config, `MISE_*` sweep). |

Unaddressed R-IDs: []

Suppressed findings: 2
Classification counts: 4 introduced, 0 pre_existing.

```json
{"suppressed_count":{"25":2},"classification_counts":{"introduced":4,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>