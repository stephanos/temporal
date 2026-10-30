Coordinator format repair: preserve the correctness P1 unchanged; collapse integration:1 as the same config-discovery defect. Optional P3 documentation/diagnostic FYIs remain in raw draws.

## Issue 1
- **Severity**: P1
- **Title**: Root Make Cargo misses the workspace linker config and flips shared fingerprints
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: model/.cargo/config.toml:6
- **R-IDs**: [R9]
- **Problem**: Cargo discovers `.cargo/config.toml` from the current directory upward, not from `--manifest-path`. Every root `Makefile` Rust entry runs `mise exec -- cargo run --manifest-path "$(CURDIR)/model/Cargo.toml" ...` from the repository root, so those builds never see the new `[target.*].linker` and link with GNU ld. Every nested invocation does see it: the gate runs Cargo from the Temporal root (`model/umpire/dev/xtask/src/gate.rs:61`), the driver bootstrap runs from the framework root (`model/umpire/dev/xtask/src/main.rs:37`), and `scripts/xtask.sh:15` changes into it. Cargo hashes the resolved target linker into every unit's fingerprint (dirty reason "config settings changed"), and the two paths share one target directory and the `dev` profile. Failing scenario on Linux: `make model-quick` compiles temporal-xtask and its closure with no linker; the nested `quick`/gate Cargo run marks the shared units (serde, toml, thiserror, umpire-guest-driver lib, xtask, ...) dirty and recompiles them with `-C linker=.../mold-linker.sh`; the next `make` flips them back. Each Makefile entry pays a recompile of the shared closure twice. This is the inner loop R6 targets and the host build cost R9 claims to reduce. The proof did not exercise this path: the retained Temporal tool build ran from `model/` (its rustc argv uses the workspace-relative `dev/xtask/src/main.rs`).
- **Suggestion**: Make the Makefile's Cargo invocations run from `model/` (for example `cd "$(CURDIR)/model" && $(RUST_MODEL_CARGO) run ...`, or fold the `cd` into `RUST_MODEL_CARGO`), so the top-level build sees the same config as the nested ones. Then confirm with `cargo build -v` across one `make model-quick` cycle that no unit reports a config-settings dirty reason.

Unaddressed R-IDs: []

Classification counts: 1 introduced, 0 pre_existing.

<verdict>NEEDS_WORK</verdict>
