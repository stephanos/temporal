**Review of fn-254.20 (mold linker adoption), correctness axis**

I read the task, the eleven production/doc paths, the proof's command list, the Makefile entry points, both bootstrap scripts and the guest driver's Cargo invocation. The adapter, target scoping, WASI isolation and the test are sound. One introduced regression in the Makefile entry path blocks.

### Findings

- **Severity**: P1
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `model/.cargo/config.toml:6` (with `Makefile:467`, `Makefile:1267`, `Makefile:1272`, `Makefile:1284`, `Makefile:1288`)
- **R-IDs**: [R9]
- **Problem**: Cargo discovers `.cargo/config.toml` from the current directory upward, not from `--manifest-path`. Every root `Makefile` Rust entry runs `mise exec -- cargo run --manifest-path "$(CURDIR)/model/Cargo.toml" ...` from the repository root, so those builds never see the new `[target.*].linker` and link with GNU ld. Every nested invocation does see it: the gate runs Cargo from the Temporal root (`model/umpire/dev/xtask/src/gate.rs:61`), the driver bootstrap runs from the framework root (`model/umpire/dev/xtask/src/main.rs:37`), and `scripts/xtask.sh:15` changes into it. Cargo hashes the resolved target linker into every unit's fingerprint (dirty reason "config settings changed"), and the two paths share one target directory and the `dev` profile. Failing scenario on Linux: `make model-quick` compiles temporal-xtask and its closure with no linker; the nested `quick`/gate Cargo run marks the shared units (serde, toml, thiserror, umpire-guest-driver lib, xtask, ...) dirty and recompiles them with `-C linker=.../mold-linker.sh`; the next `make` flips them back. Each Makefile entry pays a recompile of the shared closure twice. This is the inner loop R6 targets and the host build cost R9 claims to reduce. The proof did not exercise this path: the retained Temporal tool build ran from `model/` (its rustc argv uses the workspace-relative `dev/xtask/src/main.rs`).
- **Suggestion**: Make the Makefile's Cargo invocations run from `model/` (for example `cd "$(CURDIR)/model" && $(RUST_MODEL_CARGO) run ...`, or fold the `cd` into `RUST_MODEL_CARGO`), so the top-level build sees the same config as the nested ones. Then confirm with `cargo build -v` across one `make model-quick` cycle that no unit reports a config-settings dirty reason.

- **Severity**: P3
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `mise.toml:9`, `model/design/gate-speed.md:270`
- **R-IDs**: []
- **Problem**: The root manifest carries an independent second pin of mold 3.0.0 that the adapter never consults (it always resolves from `model/umpire`). The doc states the change "adds no ... second version source". If the two pins drift, root `mise install` provisions one version while the adapter demands another and refuses. Visible failure, no silent misuse, hence P3.
- **Suggestion**: Either state in the doc that the root pin exists only so `mise install` at the root provisions the framework's tool and must match, or drop the root pin and document `mise -C model/umpire install` for Linux.

- **Severity**: P3
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `model/umpire/dev/guest-driver/tests/linker.rs:14`
- **R-IDs**: []
- **Problem**: The control asserts only the framework config. `model/.cargo/config.toml` has no control, so it can drift to a different linker path or add a target without any test failing.
- **Suggestion**: Optional. If the Temporal workspace has a parallel control location, assert its two target entries point at `umpire/scripts/mold-linker.sh`.

Checked and clean, for the record: the adapter preserves argv bytes (spaces and non-UTF-8 verified by the test), fails hard on missing mise/tool/executable, forces auto-install off, and resolves relative to its own script path so Cargo's absolute resolution works both in-tree and in the standalone copy. No `target.wasm32-wasip1` entry exists, so WASI units keep rust-lld with the driver's encoded flags; host proc-macro and build-script units under the WASI build correctly take the host linker and the proof shows unchanged guest bytes and sidecar identities. trybuild writes its own `.cargo/config.toml` containing only rustflags, so the relative linker path is not copied into nested projects. Non-Linux hosts and musl triples are untouched.

### Requirements coverage

| R-ID | Status | Evidence |
| --- | --- | --- |
| R7 | met | Standalone copy outside the repository linked with mold (`task20/standalone.log`); the adapter resolves from the framework root; bootstrap runs from `model/umpire`. |
| R9 | partial | Matched relink measurement and adoption decision retained in `gate-speed.md`; the Makefile entry path neither uses mold nor shares fingerprints with nested builds (P1 above). |

Unaddressed R-IDs: []

Suppressed findings: 1
Classification counts: 3 introduced, 0 pre_existing.

```json
{"suppressed_count":{"25":1},"classification_counts":{"introduced":3,"pre_existing":0},"unaddressed":[]}
```

<verdict>NEEDS_WORK</verdict>