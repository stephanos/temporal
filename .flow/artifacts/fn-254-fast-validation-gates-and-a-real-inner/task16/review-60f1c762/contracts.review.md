**Review of fn-254.16 provider diff (contracts-and-consistency axis)**

Scope read: task .16 spec and R7, the full diff, current `tool_provider.rs`, `jar.rs`, `support.rs`, `syscall.rs`, `fetch_apalache.rs`, `xtask.sh`, `standalone.rs`, gate wiring, mise-tools assets, and all `Tlc::discover`/`Apalache::discover` callers. The task16 evidence directory under the main checkout's `.flow/artifacts` was not readable from this sandbox (permission denied outside the worktree), so real-install, concurrency and watchdog evidence is taken as stated in the dispatch, not re-verified here.

Contracts verified as consistent: exact TLC 1.7.4 / Apalache 0.62.4 URLs and SHA-256 agree across `mise.toml`, `mise.lock`, `mise.apalache.toml`, `mise.apalache.lock`, `apalache_pins.rs`, `umpire_tlc::JAR_SHA256` and the xtask literal; Apalache only enters via `--env apalache` and the gate only fetches it under `UMPIRE_APALACHE=require`; all `MISE_*` inherited selectors are stripped and the ceiling/trusted/data/cache/state/config overrides are scoped to `build/mise-jars`; auto-install is disabled on every provider command; `admit_mise` enforces regular-file, no-symlink, contained path and in-process SHA before spawning mise; unsupported (non-UTF-8 or `:`-containing) roots refuse with `InvalidInput` before any directory creation, as the README states; every production `discover` caller passes the framework root, matching the new install path; `Running::spawn` is literally `spawn_with_policy(Checker)`, so the `OwnedProcess::spawn` change is behavior-neutral.

---

**Finding 1**
- **Severity**: P2
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `model/umpire/dev/xtask/src/gate/standalone.rs:418`, `model/umpire/scripts/fetch-tlc.sh:5`
- **R-IDs**: [R7]
- **Problem**: The standalone gate's `standalone-tlc-tool` step now runs `scripts/fetch-tlc.sh`, which bootstraps xtask through `xtask.sh` into `$CARGO_TARGET_DIR/xtask-bootstrap`. The preceding `standalone-local-tools` step already built xtask and the guest driver into the same `CARGO_TARGET_DIR` (`target/debug`). On a fresh standalone copy this is a second full compile of xtask, umpire-guest-driver and their dependency closure, purely to run `fetch-tlc`. The retired script needed only bash and curl. The task text says download cleanup must not delay the .5/.17 validation speedups; this adds minutes of compile to the standalone setup phase.
- **Suggestion**: In the standalone setup, invoke the already-built binary directly (`target/debug/xtask fetch-tlc <root>`) instead of the shell script, or teach `xtask.sh` to reuse an existing `debug/xtask` in `CARGO_TARGET_DIR` before bootstrapping.

**Finding 2**
- **Severity**: P3
- **Confidence**: 100
- **Classification**: introduced
- **File:Line**: `model/umpire/backends/process/src/support.rs:182`
- **R-IDs**: []
- **Problem**: The new `match policy` in `OwnedProcess::spawn` has two arms that do the same thing: `Running::spawn` (syscall.rs:128) is defined as `spawn_with_policy(..., Checker)`. The branch suggests Checker takes a different path when it does not.
- **Suggestion**: Collapse to `crate::syscall::Running::spawn_with_policy(command, directory, stdout, stderr, policy).map(Self)`.

**Finding 3**
- **Severity**: P3
- **Confidence**: 75
- **Classification**: introduced
- **File:Line**: `model/umpire/dev/xtask/src/fetch_apalache.rs:18`, `model/umpire/dev/xtask/src/apalache_pins.rs:1`
- **R-IDs**: []
- **Problem**: The TLC post-install hash is a bare literal duplicating `umpire_tlc::JAR_SHA256` and the checksum in `mise-tools/mise.toml`; nothing checks the three agree. The Apalache pin-drift control (`backends/apalache/tests/main/controls.rs:253`) compares constants against `apalache_pins.rs` only, not against `mise.apalache.toml`/`.lock`, so `ARCHIVE_SHA256` and `VERSION` are now mirrored by the mise declarations with no drift guard. The `apalache_pins.rs` header still says "shared by installer and backend", but the installer no longer reads `ARCHIVE_SHA256` or `VERSION`. A future bump of one file and not the others yields a successful `mise install` followed by "pinned jar integrity mismatch".
- **Suggestion**: Add one focused test that the committed `mise-tools/*.toml` contain the pinned version strings and `sha256:` checksums from the Rust constants, and source the TLC literal from a single pin constant.

**Finding 4**
- **Severity**: P3
- **Confidence**: 100
- **Classification**: introduced
- **File:Line**: `model/umpire/backends/process/src/jar.rs:21`, `model/README.md:155`
- **R-IDs**: []
- **Problem**: The `admit_mise` doc says "Resolve an exact mise installation, then retain the same hash and Java admission"; the code hashes the installed file first and runs `mise where` second. `model/README.md` lists path, digest, Java, probe and omits the mise resolution step entirely. Minor, but this is exactly the admission order the task asks to make explicit.
- **Suggestion**: State the actual order: contained regular file, SHA-256, exact `mise where` resolution, pinned Java, version probe.

**FYI (non-blocking)**
- `PinnedJar::admit` (`jar.rs:47`) has no production callers after this change and survives only for its own unit tests; the spec asks for the custom cache layer to be removed.
- `tool_provider::resolve_cancelled` and `install_cancelled` are public but only `resolve`/`install` are called outside tests; they could be private.
- `model/.gitignore:2-3` still ignores `/build/tlc/` and `/build/apalache/` at the model root (pre-existing, harmless).
- `mise-tools/README.md` claims about the interrupted-HTTP retry control, concurrent locked installs, and real clean installs point to task16 artifacts that could not be opened from this sandbox; the in-tree tests cover shell fixtures, not real HTTP.

## Requirements coverage
| R-ID | Status | Evidence |
| --- | --- | --- |
| R7 | met | Tool pins, mise declarations and the `build/mise-jars` cache all resolve under `model/umpire`; the standalone copy control is reported passing (17 stages) on this exact commit; the UTF-8 / path-list narrowing of the provider root is explicitly documented as a changed contract in `dev/xtask/README.md` and `mise-tools/README.md`. |

Unaddressed R-IDs: []

Suppressed findings: 2 (confidence 25: mise ceiling-inclusive semantics; global `.tool-versions` reachability under `MISE_GLOBAL_CONFIG_FILE`).
Classification counts: 4 introduced, 0 pre_existing.

```json
{"suppressed_count":{"25":2},"classification_counts":{"introduced":4,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>