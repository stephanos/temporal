---
satisfies: [R7]
---
# fn-254-fast-validation-gates-and-a-real-inner.10 Move the framework gate into model/umpire; root Makefile keeps Temporal steps only

Touches: [model/umpire/**, model/dev/xtask/**, model/scripts/**, model/Cargo.toml, model/Cargo.lock, Makefile, model/README.md, model/umpire/README.md]

## Description
Implement .9's layout: framework stages run from `model/umpire`, and framework paths, pins and caches resolve there (spec R7).

**Cross-spec gate:** coordinate with in-flight fn-246 lanes; workspace and lockfile moves touch every crate's build. Land at an integration boundary when no lane has uncommitted Cargo changes.

**Size:** M
**Files:** framework entry point (xtask or Makefile per .9), framework xtask modules moved under `model/umpire`, `model/Cargo.toml`/`Cargo.lock` (or a new `model/umpire` workspace), cache roots in `umpire/backends/process` and `apalache`, `scripts/proto-go.sh` split, root `Makefile`, READMEs.
**Touches:** [model/umpire/**, model/dev/xtask/**, model/scripts/**, model/Cargo.toml, model/Cargo.lock, Makefile, model/README.md, model/umpire/README.md]

### Approach
- Replace repo-root-relative framework paths with paths relative to the framework root, or a single framework-root resolver. TLC and Apalache caches move under the framework root (or a `CARGO_TARGET_DIR`-relative location); keep pin and SHA admission unchanged.
- Framework protobuf generation and checks use framework-owned pinned generators, not the root `.bin/`.
- The root `Makefile` `model-check`/`model-validate` call the framework entry point first, then run only Temporal steps: Temporal guest freshness, source boundary, Temporal protobuf, Temporal Go host, Temporal crate tests.
- Stage names, fail-fast behaviour and `UMPIRE_TIME_STEPS` timing are preserved across both halves.
- **Equivalence pin:** the combined stage list and the full test list (with ignored tests) are identical before and after; the TLC-required run passes.

### Investigation targets
**Required:**
- `model/design/framework-gate.md` — layout and decisions
- `model/umpire/backends/process/src/support.rs`, `model/umpire/backends/apalache/src/lib.rs:120-190` — cache roots
- `model/scripts/proto-go.sh` — core/kettle/temporal modes
- `Makefile:1270-1315`

### Acceptance
- [ ] The framework entry point runs every framework stage from `model/umpire`; the root `Makefile` contains no framework stage.
- [ ] No framework path reaches above the framework root; caches and pins resolve under it.
- [ ] Stage list and test list unchanged in sum; `make model-check` and TLC-required `make model-validate` pass.

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
