---
satisfies: [R2, R3]
---
# fn-247-one-language-for-the-framework-a-rust.2 Port lock closure and rustc identity normalization into host-only xtask

## Description

Introduce native xtask and port lock/compiler identity normalization (R2, R3), after the Rust supervisor checkpoint. This task owns shared serialization/hash support; shell orchestration and freshness move in its successor.

**Size:** M
**Files:** `model/dev/xtask/{Cargo.toml,src/**,tests/**}`, `model/{Cargo.toml,Cargo.lock,.cargo/config.toml}`, `model/scripts/{guest-lock.py,guest-rustc.py,tests/test_guest_lock.py,tests/test_guest_rustc.py}`.
**Touches:** [model/dev/xtask/**, model/Cargo.toml, model/Cargo.lock, model/.cargo/config.toml, model/scripts/guest-lock.py, model/scripts/guest-rustc.py, model/scripts/tests/test_guest_lock.py, model/scripts/tests/test_guest_rustc.py]

## Approach

- Apply the conductor's narrow host-only `toml` 1.1.8+spec-1.1.0 direct promotion with its existing lock checksum, `default-features=false` and only `std`, `parse`, `serde`. Add `sha2` 0.11 as an R18-admitted host-only dependency. Capture before/after dependency and feature inventories first, with each new transitive package's R18 basis; no host-only package may enter a guest closure and `cargo deny check bans` passes. Recheck actual post-.3 resolution before editing Cargo.
- Capture old CLI/function cases from both Python test modules into a named behavioral corpus before replacing sources. Identity encodings need not match Python bytes (escaping, compact/spaced JSON); they must be deterministic, host-independent and stable across runs.
- Add a host-only xtask dispatcher and cargo alias using existing workspace serde/serde_json/error patterns: plain `serde_json` (key order from its default BTreeMap-backed `Map`; keep `preserve_order` off workspace-wide), in-process `sha2` hashing (no `shasum` subprocess), and `std::env::args` argument handling (no clap). Maintain the approved full parser semantics for workspace and sysroot locks. The original Python uses TOML1.0 while the resolved parser supports TOML1.1; preserve old acceptance/refusal behavior for inputs outside their shared grammar through compatibility validation or return that precise discrepancy before broadening it. Verify new workspace edges/features cannot alter guest closure identities.
- Port `guest-lock` and `guest-rustc`; preserve dependency closure checksums/edge shape, dev exclusions, all-platform/build inclusion, ambiguous coordinates, unification, build-std exemption and native macro collisions. Preserve arbitrary OS argument/environment bytes where the wrapper contract permits them, previous-wrapper execution and signal statuses.
- Publish sidecars before child compilation, preserve jobserver fds and loader environment, remove sidecars after launch/compiler failure and use atomic publication. Keep the Python command files until guest orchestration switches to the new implementation in the next task; ported tests may run both to compare closure structure and outcomes, not bytes.

## Investigation targets

**Required**:
- `model/scripts/guest-lock.py:16` — closure schema and ordering.
- `model/scripts/guest-rustc.py:56` — compiler normalization and sidecars.
- `model/scripts/tests/test_guest_lock.py:57` — old closure controls.
- `model/scripts/tests/test_guest_rustc.py:55` — old wrapper/FD controls.
- `model/Cargo.toml:74` — admitted workspace dependency policy.
## Quick commands

```bash
cd model && mise exec -- cargo test -p xtask --test guest_identity --locked
cd model && mise exec -- cargo clippy -p xtask --all-targets -- -D warnings
```

The task creates the named integration target. Run the retained behavioral corpus directly; no full model gate here.

## Acceptance

- [ ] The explicit TOML amendment retains its exact package/version/checksum and existing minimal features; `sha2` 0.11 is the only other new direct dependency; before/after inventory records each new package's R18 basis and proves no guest-closure change or unapproved grammar widening/restriction.
- [ ] Native xtask/alias build independently of guest target/profile/wrapper flags; lock maps, identities and sidecars are deterministic (repeat runs and differently located checkouts agree) and contain no host paths; JSON key order comes from serde_json's default `Map` with `preserve_order` off; no `shasum` subprocess or clap.
- [ ] Every guest-lock/rustc unittest has a named Rust counterpart, including closure propagation/unification, malformed/ambiguous inputs, build-std, macro collision, pipelining, previous-wrapper and inherited jobserver/loader environment.
- [ ] Wrapper status and atomic publication/failure cleanup controls pass; unrelated host-only additions do not change normalized guest dependency identities.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
