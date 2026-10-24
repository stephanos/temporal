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
- Inventory useful closure/wrapper controls from both Python modules before replacing sources; map each responsibility to a native Rust test or explicitly retire a Python-only quirk with owner provenance. Native identity schemas need not match Python bytes, spacing or escaping; they must be deterministic, host-independent and stable across runs.
- Add a host-only xtask dispatcher and cargo alias using workspace serde/serde_json/error patterns: plain `serde_json` with default BTreeMap-backed `Map` (`preserve_order` off workspace-wide), in-process `sha2` (no shasum), and `std::env::args_os`/`OsString` (no clap); decode command tokens without corrupting raw compiler/path arguments. Use approved native `toml` 1.1.8 grammar/limits/errors and native JSON schemas. Current Cargo/mise/rust-toolchain/rust-src documents must work. No unbounded feature, Python grammar/BigInt/date/recursion/digit/Unicode/error-order emulation or compatibility parser. Document deliberate CLI/schema/error breaks and update in-repo callers. New host-only edges/features must not enter guest closures.
- Port `guest-lock` and `guest-rustc`; preserve dependency closure checksums/edge shape, dev exclusions, all-platform/build inclusion, ambiguous coordinates, unification, build-std exemption and native macro collisions. Preserve arbitrary OS argument/environment bytes where the wrapper contract permits them, previous-wrapper execution and signal statuses.
- Publish sidecars before child compilation, preserve jobserver fds and loader environment, remove sidecars after launch/compiler failure and use atomic publication. Keep the Python command files until guest orchestration switches to the new implementation in the next task; test semantic closure shape, negative controls and the native contract directly; no old/new parser or byte differential is required.

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

- [ ] The explicit TOML amendment retains its exact package/version/checksum and existing minimal features; `sha2` 0.11 is the only other new direct dependency; before/after inventory records each new package's R18 basis and proves no host-only package entering a guest closure; native TOML grammar/limits/errors and JSON schemas are documented with current workspace inputs and malformed-input controls.
- [ ] Native xtask/alias build independently of guest target/profile/wrapper flags; lock maps, identities and sidecars are deterministic (repeat runs and differently located checkouts agree) and contain no host paths; JSON key order comes from serde_json's default `Map` with `preserve_order` off; no `shasum` subprocess or clap.
- [ ] Useful guest-lock/rustc responsibilities have named Rust controls, with Python-only quirks explicitly retired under owner permission, including closure propagation/unification, malformed/ambiguous inputs, build-std, macro collision, pipelining, previous-wrapper and inherited jobserver/loader environment.
- [ ] Wrapper status and atomic publication/failure cleanup controls pass; unrelated host-only additions do not change normalized guest dependency identities.


## Preparation evidence

Read-only research: `.flow/artifacts/fn-247-one-language-for-the-framework-a-rust/planning/native-identity-research.md` and `.flow/artifacts/fn-247-one-language-for-the-framework-a-rust/planning/sha2-primary-sources.md`. These ground the existing scope, not extra acceptance gates. The before inventory and its source hashes may be reused only while those hashes still match; all-platform normal/build metadata is not a compiled guest-runtime closure proof.

## Done summary
Developers can build the host-only xtask and use its native lock-closure and compiler-wrapper commands. Schema 2 preserves portable dependency/compiler identities, native-macro collision detection, raw Unix arguments and environment, inherited jobserver descriptors, pipelined atomic sidecars, and failure/status cleanup; the Python callers remain active until task .5 switches orchestration.

The 21 retained Python controls passed before edits. The native package did not exist at that baseline. Native stub and graph controls failed first, then passed; the final integrated guest_identity target passed all 25 controls, including actual rustc proc-macro builds in two locations. Scoped Clippy with warnings denied and cargo deny check bans passed. The initial integrated test fixture-path failure, test-only lint failures, and inherited wildcard bans failure are retained in task2 artifacts. The owner authorized the separate consumer dependency-version correction c787887bea; the bans policy stayed unchanged.

Cargo adds sha2 0.11.1 and its six researched R18-admitted transitives. Existing package versions, checksums, resolved features and edges remain unchanged; the exact admitted TOML version/checksum/minimal requested features are retained and serde_json preserve_order is disabled. Actual native CLI runs against before/after Cargo metadata produce unchanged identities for all six guest closures. This proves unchanged all-platform non-development normal/build topology and feature union, including existing host build-tool reachability. Integrated wasm reproduction/freshness belongs to successors. The control ledger explicitly retires Python formatting/parser/interpreter quirks under the owner's breaking-driver permission.

Three fresh codex:gpt-6.1-sol:high draws returned SHIP, and the finalizer recorded SHIP with exit 0. The initial merge-plan finalizer refused the correctness draw's missing structural container before recording; the documented merged-file repair preserved the same three verdicts without redispatch. Reviewers share the writer's Codex/GPT family. Their Cargo reruns were blocked by read-only build locks; they checked the passing logs and matching hashes, and correctness additionally ran three existing read-only controls. All 17 verified source hashes still match after review, so Verify reuses observed passing focused exits. The classifier returned FULL for externally owned MILESTONES.md; this task defines focused checks, and no full Model gate or fabricated skip is claimed.

Durable commands, failures, dependency/features inventories, control mappings, source hashes, actual metadata identities and review receipts live in .flow/artifacts/fn-247-one-language-for-the-framework-a-rust/task2/. Both task-owned port worktrees/branches were integrated and removed; the independent .9 worktree and shared caches remain. Native macOS execution is unclaimed. No guest/frozen artifact, Model semantics, protocol, remote operation or history rewrite occurred. Root's fb25091a0e metadata checkpoint is external work inside the fixed 5fb5b2206cffd8187692b82bdd10037ede26b152 task range; the original spec base remains preserved.

stage: impl-review - ran (three first-pass SHIP draws; finalizer SHIP; task2/review-receipt.json)
Tier: session (jev-unavailable(no_key)); root judged once at .flow/tmp/fn2472-dispatch-judge.json; no key retry/reroute.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 4b7e225cb0d1d419e5ccfe737fc252c1eba4fa53, fb25091a0e73d3c25a2569cd811f78ac31d2eb5f, 74855f06cc5b4b2704a8b6a2e367e253c295a409, c787887bea9c798b2bd61f4925e5a09577e4bbab, 20939f2e94a0c250700d3e159f611b6f5e2d59fe, 5a451ad9edf5bf0b2cd649af86d9b5f67fadd8b0, 13d60d7c36b05bab06f6ec6c28d271cb85ac0f14
- Tests: python3 -m unittest discover -s model/scripts/tests -p test_guest_lock.py, python3 -m unittest discover -s model/scripts/tests -p test_guest_rustc.py, cd model && mise exec -- cargo test -p xtask --test guest_identity --locked, cd model && mise exec -- cargo clippy -p xtask --all-targets -- -D warnings, cd model && mise exec -- cargo deny check bans
- PRs: