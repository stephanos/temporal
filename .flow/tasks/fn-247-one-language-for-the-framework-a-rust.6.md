---
satisfies: [R2, R3]
---
# fn-247-one-language-for-the-framework-a-rust.6 Port Apalache installation with exact archive, publication and concurrency controls

## Description
Port Apalache installation and its security/concurrency controls (R2, R3), after guest integration establishes shared xtask helpers. This isolates archive admission from the other helper ports.

**Size:** M
**Files:** `model/dev/xtask/{Cargo.toml,src/fetch_apalache*,tests/apalache*}`, `model/{Cargo.toml,Cargo.lock}`, `model/scripts/{fetch-apalache.py,tests/test_apalache.py}`, `model/umpire/backends/apalache/tests/controls.rs`, `Makefile`.
**Touches:** [model/dev/xtask/**, model/Cargo.toml, model/Cargo.lock, model/scripts/fetch-apalache.py, model/scripts/tests/test_apalache.py, model/umpire/backends/apalache/tests/controls.rs, Makefile]

### Approach

- Add R18-admitted host-only `tar` 0.4.46 (`default-features = false`, no xattr) and `flate2` 1.1.10 (miniz_oxide backend) to xtask; record before/after inventory with each new transitive package's R18 basis and no guest-closure change.
- Verify the downloaded archive's SHA-256 (in-process `sha2`) before any parsing. Then stream it through `flate2::read::GzDecoder` and `tar::Archive`, select the one exact named member, require it to be a regular file, read its bytes in memory and verify the jar SHA-256. No system tar, member filtering, verbose listing, GNU/BSD parity proof or macOS receipt. No arbitrary caller-supplied archive grammar is introduced; duplicate/type7/gzip-trailer research cases cannot pass the archive hash and need no handling.
- Keep one Rust pin authority for version/archive hash/jar hash; update the backend's source-include pin consistency control to reference it without copying a second authority.
- Preserve download curl arguments, existing path/status interface, containment/component/symlink checks, partial/wrong installation refusal and private staging cleanup. Open the lock file with `O_NOFOLLOW`, require a regular file, and take the exclusive lock with std `File::lock()` (stable since Rust 1.89). Atomically publish only validated jar bytes; never unpack the archive to disk.
- Port every mocked-download/malicious-lock/archive/staging/concurrency test to Rust, building fixture archives with `tar`/`flate2` in the test. Concurrent installers must still perform one publication and return the same admitted path. Switch only the installer Makefile caller, then remove its Python source/control.

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
- [ ] Archive SHA-256 is verified before parsing; the exact named regular member is read in-process via `tar` + `flate2` and its jar SHA-256 verified before atomic publication; no system tar is invoked. Archive/jar hashes and version remain under one Rust pin authority. The real pinned release installs and the final gate's required Apalache run uses it.
- [ ] Every old installer case has a Rust counterpart; wrong/partial/symlink/nonregular-lock/hash/missing-or-nonregular-member/cleanup controls fail and concurrent installers (std `File::lock()`) publish once.
- [ ] Installer statuses and declared diagnostics match the old contract; dynamic curl/path output is validated structurally.
- [ ] `tar`/`flate2` admissions are recorded with R18 basis and no guest-closure change; Makefile and Apalache pin-consistency controls use Rust; installer Python/control removed after its Rust counterparts pass.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
