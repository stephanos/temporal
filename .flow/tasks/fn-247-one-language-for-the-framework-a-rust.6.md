---
satisfies: [R2, R3]
---
# fn-247-one-language-for-the-framework-a-rust.6 Port Apalache installation with exact archive, publication and concurrency controls

## Description
Port Apalache installation and its security/concurrency controls (R2, R3), after completed .2 establishes native xtask support; no .5 guest-bootstrap implementation is required. This isolates archive admission from the other helper ports.

**Size:** M
**Files:** `model/dev/xtask/{Cargo.toml,src/fetch_apalache*,tests/apalache*}`, `model/{Cargo.toml,Cargo.lock}`, `model/scripts/{fetch-apalache.py,tests/test_apalache.py}`, `model/umpire/backends/apalache/tests/controls.rs`, `Makefile`.
**Touches:** [model/dev/xtask/**, model/Cargo.toml, model/Cargo.lock, model/scripts/fetch-apalache.py, model/scripts/tests/test_apalache.py, model/umpire/backends/apalache/tests/controls.rs, Makefile, model/umpire/backends/apalache/src/lib.rs]

### Approach

- Implement as an independent isolated source lane beside .5/.7. Reuse .2 `Error`/`Result` and hashing/encoding/publication interfaces; installer commands call the compiled host tool and do not require the guest freshness launcher. Own tar/flate2 Cargo/lock edits in this lane's copy; conductor merges shared dispatcher/lib/Makefile/Cargo edits and verifies their combined package/features/guest closure before completion. Lane-local fixture controls are useful evidence, not a claim the unmerged combined tool passes.
- Add R18-admitted host-only `tar` 0.4.46 (`default-features = false`, no xattr) and `flate2` 1.1.10 (miniz_oxide backend) to xtask; record before/after inventory with each new transitive package's R18 basis and no guest-closure change.
- Verify the downloaded archive's SHA-256 (in-process `sha2`) before any parsing. Then stream it through `flate2::read::GzDecoder` and `tar::Archive`, select the one exact named member, require it to be a regular file, read its bytes in memory and verify the jar SHA-256. No system tar, member filtering, verbose listing, GNU/BSD parity proof or macOS receipt. No arbitrary caller-supplied archive grammar is introduced; duplicate/type7/gzip-trailer research cases cannot pass the archive hash and need no handling.
- Keep one Rust pin authority for version/archive hash/jar hash; update the backend's source-include pin consistency control to reference it without copying a second authority.
- Keep useful ordinary download/path invocation behavior and document native error/output changes; preserve curl safety, containment/component/symlink checks, partial/wrong installation refusal and private staging cleanup. Open the lock file with `O_NOFOLLOW`, require a regular file, and take the exclusive lock with std `File::lock()` (stable since Rust 1.89). Atomically publish only validated jar bytes; never unpack the archive to disk.
- Port useful mocked-download/malicious-lock/archive/staging/concurrency responsibilities to Rust, explicitly retiring only pure Python quirks under owner permission, building fixture archives with `tar`/`flate2` in the test. Concurrent installers must still perform one publication and return the same admitted path. Switch only the installer Makefile caller, then remove its Python source/control.

### Investigation targets

**Required**:
- `model/scripts/fetch-apalache.py:18` — containment, lock and publication.
- `model/scripts/fetch-apalache.py:52` — download/archive/member/hash contract.
- `model/scripts/tests/test_apalache.py:41` — complete old failure controls.
- `model/umpire/backends/apalache/tests/controls.rs:248` — pin-consistency caller.
- `Makefile:1277` — required backend preflight.

### Quick commands

```bash
cd model && mise exec -- cargo test -p xtask --test apalache_install --locked
cd model && mise exec -- cargo test -p umpire-apalache --test controls --locked
```

Use controlled download/archive fixtures for installer tests; real required Apalache execution belongs to the final gate.

## Acceptance
- [ ] Archive SHA-256 is verified before parsing; the exact named regular member is read in-process via `tar` + `flate2` and its jar SHA-256 verified before atomic publication; no system tar is invoked. Archive/jar hashes and version remain under one Rust pin authority. The real pinned release installs before this task completes; final .3 owns required Apalache consumption of that exact pinned content before parent closure.
- [ ] Every useful installer safety/control responsibility has a Rust counterpart; wrong/partial/symlink/nonregular-lock/hash/missing-or-nonregular-member/cleanup controls fail and concurrent installers (std `File::lock()`) publish once.
- [ ] Installer callers and tests use the documented native status/error/output contract; useful ordinary invocations remain, with no exact Python diagnostics/API/byte parity requirement. Dynamic curl/path output is checked structurally.
- [ ] `tar`/`flate2` admissions are recorded with R18 basis and no guest-closure change; Makefile and Apalache pin-consistency controls use Rust; installer Python/control removed after its Rust counterparts pass.


- [ ] The conductor has joined .5/.6/.7 shared-file edits and verified installer dispatch/Make caller, admitted dependency features and real pinned-release installation against the joined tool before task completion. No extra full gate is required here.

## Done summary
Ported Apalache installation to native xtask with one Rust pin authority shared by installer/backend. Archive SHA is checked before tar/flate2 parsing, one exact regular jar member is read in memory, jar SHA is checked before publication, no-follow regular-file locking uses std File::lock, and containment/private staging/cleanup/concurrency controls are retained. Python installer/tests are removed; native dispatch/Make caller and documented output/error contracts are joined.

baseline: green. Eight installer fixture controls,14 backend test functions (optional real execution unclaimed), scoped Clippy/format/bans and joined81 tests+2 lifecycle controls+Clippy0+cold scan0 are recorded. The frozen joined tool actually installed the pinned183.1MiB HTTPS release, admitted both hashes/exact member and atomically published; no-curl cached reuse returned the same path. Seven new host packages have R18 basis; exact Cargo files match lane source, native complete lock maps match lane/joined, and all6 guest identities agree before/lane/joined. See task6/controls-and-handover.md and task6/joined/{README.md,source-binding.json,execution-receipt.json,native-identity-proof.json}.

Three fresh read-only codex:gpt-6.1-sol:high draws returned SHIP with no findings; finalizer exited0. They verified source/artifact hashes and contracts but their Cargo executions were blocked before tests by read-only locks; independent test reruns are not claimed. Same-family provenance is honest. No task6 source fix or proof/gate repeat occurred. Verify returned FULL/unmatched root-owned MILESTONES.md; task6 has focused checks, not a full Model gate, so source-bound passing evidence is reused without fabricated receipts/skips. Final3 owns required consumption of the exact validated content and the one semantic Go/TLC/Apalache pipeline before parent closure; component completion does not wait for its dependent3.

Exact review base1bff716b76455c357c3db15a981aa75fe71e9939 includes coupled5/7 implementation/root bookkeeping and task7's later reviewed shell fix, not synthetic task-only history. Original task6 source17bce75e1c remains unchanged. Canonical guest and build/compiler/identity inputs are unchanged by this review. No MILESTONES, remote or parent-close operation occurred. Durable review: task6/review/receipt.json and provenance.md.

stage: impl-review - ran (three first-round SHIP draws; backend finalizer SHIP; task6/review/receipt.json)
Tier: session (jev-unavailable(no_key))

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 17bce75e1cbd28bb7e6939a869ee961c0df784a7, 523d131d2cf992b04a43b4d595ea0f1a434f44b6, fb9b9ef7f142f8d1ae5f404be018c8888eef0f95, 6c4e7f39ffd285a679fc1172d5f72dc4cd006709, a7071670854876c1949b92582fc77c295d2af2a2, 8cbe8ee453025d0a494586ecd122af3ba6f11525, c0a76a47490d6ec2d6b3ac447caf54c08e7b4fb4, 127579d79a2cb9b0d5a7a3c01b3a1bb3f42fdb30
- Tests: baseline: cd model && mise exec -- cargo test -p umpire-apalache --test controls --locked (exit 0; 14 passed; optional real execution unclaimed), cd model && mise exec -- cargo test -p xtask --test apalache_install --locked (exit 0; 8 passed), cd model && mise exec -- cargo test -p xtask -p umpire-apalache --test controls --test apalache_install --locked (exit 0; 14 backend and 8 installer controls), cd model && mise exec -- cargo clippy -p xtask -p umpire-apalache --all-targets --locked -- -D warnings (exit 0), cd model && mise exec -- cargo deny check bans (exit 0; bans ok), cd model && mise exec -- cargo fmt --all --check (exit 0), cd model && mise exec -- rustfmt --edition 2024 --check dev/xtask/tests/apalache/cases.rs (exit 0), locked before/after cargo metadata: exact base manifest/lock hashes verified; 7 new host packages, 6 unchanged all-platform non-dev guest package/feature/edge closures, git diff --check (exit 0), cd model && mise exec -- cargo test -p xtask -p umpire-apalache --locked, cd model && mise exec -- cargo clippy -p xtask -p umpire-apalache --all-targets --locked -- -D warnings, bash model/scripts/retired-names.sh, cd model && mise exec -- cargo test -p umpire-temporal-tests compatibility --locked, CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2476-target /bin/bash /Users/stephan/Workspace/skunkworks/umpire/temporal/model/scripts/xtask.sh, /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2476-target/xtask-bootstrap/debug/xtask fetch-apalache /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2476-real-install-523d131, PATH=/nonexistent-fn2476 /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2476-target/xtask-bootstrap/debug/xtask fetch-apalache /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2476-real-install-523d131, cd model && CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2476-target mise exec -- cargo metadata --format-version 1 --locked, /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2476-target/xtask-bootstrap/debug/xtask guest-lock /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2476-before.lock /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2476-native-map-before < decompressed task6/metadata-before.json.gz, /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2476-target/xtask-bootstrap/debug/xtask guest-lock /Users/stephan/Workspace/skunkworks/umpire/temporal/model/Cargo.lock /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2476-native-map-lane < decompressed task6/metadata-after.json.gz, /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2476-target/xtask-bootstrap/debug/xtask guest-lock /Users/stephan/Workspace/skunkworks/umpire/temporal/model/Cargo.lock /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2476-native-map-joined < .flow/tmp/fn2476-joined-metadata.json
- PRs: