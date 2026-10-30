**Review scope.** I read the task spec, the full non-artifact diff, the measurement and proof scripts, and traced the new linker config into the unchanged consumers: the guest driver's cargo invocation and wrapper identity logic, the standalone gate's copy and clean-environment code, the framework boundary scanner, the xtask bootstrap script, the quick gate, CI workflows, and the Makefile model targets.

**Integration verdict.** The change meets the unchanged code cleanly. Cargo resolves the config-relative `linker` path to an absolute script path, so the adapter's self-location via `BASH_SOURCE` is cwd-independent. The guest driver runs cargo from the tool root, so host build scripts and proc macros pick up the linker while WASI units keep rust-lld, and the wrapper drops `-C linker` from host identities, which matches the "host variants unchanged" evidence. The standalone snapshot copies every framework file with permissions, so the script travels with the copy, and the clean environment keeps `PATH` and `HOME` so `mise` and `cc` resolve. The boundary scanner only inspects `.rs`, `Cargo.toml` and `go.mod` for path escapes, so the script's single `..` is not flagged. CI installs tools through mise-action on Linux, so the Linux-only pin is present before the gate links. No blocking findings.

## Findings

- **Severity**: P3
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `model/.cargo/config.toml:5`
- **R-IDs**: [R9]
- **Problem**: Cargo discovers config from the current directory, not from `--manifest-path`. The Makefile's model targets run `mise exec -- cargo run --manifest-path model/Cargo.toml` from the repository root, so `make model-check`, `model-time` and `umpire-guest` never read this file. The temporal-xtask binary itself still links with GNU ld on Linux under Make, while its guest-driver subprocess (cwd `model/`) does use mold. GUEST_BUILDS.md's "both workspaces select" slightly overstates coverage for the Make path.
- **Suggestion**: Either run the model workspace cargo commands with `-C model` or document that the model-workspace adoption applies only when cwd is under `model/`. Not required for this task's acceptance.

- **Severity**: P3
- **Confidence**: 100
- **Classification**: introduced
- **File:Line**: `model/umpire/scripts/mold-linker.sh:5`
- **R-IDs**: [R9]
- **Problem**: A present-but-unexecutable or missing `bin/ld.mold` exits 1 with no message. Cargo then reports only "linking with .../mold-linker.sh failed: exit status: 1", leaving the operator to guess the cause. The missing-tool case is fine because `mise where` prints its own error.
- **Suggestion**: Print one line to stderr naming the expected path before exiting, keeping the fail-hard behaviour.

- **Severity**: P3
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `model/design/gate-speed.md:277`
- **R-IDs**: []
- **Problem**: The text says the adapter "adds no ... second version source", but the root `mise.toml` and `mise.lock` now also pin mold 3.0.0. The adapter only consults the framework context, so the root pin exists solely for `mise install` from the repo root and must be kept in sync by hand, like the Rust pin.
- **Suggestion**: Reword to say the two manifests carry the same pin and the adapter resolves only the framework's.

FYI (not findings): the `xtask.sh` bootstrap failure message still says only "installed pinned Rust/Cargo are required", though a missing mold now triggers the same path. The baseline arm of the relink measurement calls `/usr/bin/cc` directly while the mold arm pays bash startup plus a `mise where` lookup, so the measured 65% link-time reduction is conservative for mold.

## Requirements coverage

| R-ID | Status | Evidence |
| --- | --- | --- |
| R7 | partial | Framework-local pin and script resolve from `model/umpire`; standalone copy includes `scripts/`. Full standalone gate renewal is deferred to task .8 by the spec's verification cadence. |
| R9 | partial | Matched forced-relink measurement, adoption decision, fail-hard controls, WASI/host-unit isolation and unchanged sidecar identities are retained in proof.json and gate-speed.md. Combined timings and full gates remain .8. |

Unaddressed R-IDs: []

Suppressed findings: 2
Classification counts: 3 introduced, 0 pre_existing.
Protected-path filter: 0

```json
{"suppressed_count":{"25":2},"classification_counts":{"introduced":3,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>