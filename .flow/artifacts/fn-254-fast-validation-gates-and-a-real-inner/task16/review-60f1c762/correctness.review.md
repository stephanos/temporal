Review complete. No shell was available in this session, so verification is by reading the diff, the current files, and the retained evidence the dispatch named; no tests were run here.

## Findings

**P2 · confidence 75 · introduced · `model/umpire/backends/process/src/tool_provider.rs:21-30`, `model/umpire/dev/xtask/src/fetch_apalache.rs:18` · R-IDs: []**
Problem: the release identities now live in three uncoordinated places. `JarTool::selector()` and `installation()` hardcode `0.62.4` and `1.7.4`, while `apalache_pins::VERSION` and the TLC backend's `JAR_VERSION` remain the declared single source of truth and are still used for version probing. `install_tlc` likewise hardcodes the TLC jar SHA as a literal instead of the TLC backend's `JAR_SHA256`. Failing scenario: bump `apalache_pins.rs` and `mise.apalache.toml` to a new release, run `fetch-apalache` successfully, then `Apalache::discover` still looks under `http-apalache/0.62.4` and refuses with "missing pinned jar" after a correct install. Nothing ties the two together; the apalache `controls.rs:253` check only covers the pins file.
Suggestion: build the selector and install-dir strings from the shared version constants (format at call time, or move the version pins into the process crate and re-export from `apalache_pins`), and source the TLC jar SHA from one constant shared with the TLC backend.

**P3 · confidence 100 · introduced · `model/umpire/backends/process/src/tool_provider.rs:553-557`, `model/umpire/backends/tlc/src/runner.rs:169-171`**
Problem: `contained()` reports every missing component as "missing pinned jar …", and `command()` reuses it for the `mise-tools` directory and its five config files. A checkout missing `mise-tools/mise.lock` with a valid installed jar fails in `resolve` with "missing pinned jar …/mise-tools/mise.lock", and `Tlc::discover` then appends "run bash scripts/fetch-tlc.sh", which cannot fix it and fails with the same text.
Suggestion: pass a label into `contained` or map the error in `command()` to "missing mise jar configuration".

**P3 · confidence 100 · introduced · `model/umpire/backends/process/src/support.rs:182-190`**
Problem: the new `match policy` is a no-op. `Running::spawn` is defined as `spawn_with_policy(…, Checker)` at `syscall.rs:128-141`, so both arms reach the same call with the same policy. Middle-man code that suggests a behavioural distinction that does not exist.
Suggestion: restore the single `spawn_with_policy(command, directory, stdout, stderr, policy)` call.

**P3 · confidence 100 · introduced · `model/umpire/backends/process/src/tool_provider.rs:546` vs `jar.rs:69`**
Problem: Duplicated Code. Two `contained()` walkers with the same symlink/normal-component/canonicalize logic now coexist; after this diff `PinnedJar::admit` has no non-test caller (only `jar.rs:134,209`), so the older walker exists only to serve its own unit tests.
Suggestion: route `admit` through `tool_provider::contained` or drop `admit` with its tests.

**FYI (non-blocking):**
- Spawning mise under `DescendantPolicy::Orchestration` delegates the inherited JVM lease fd and lifecycle fds to mise (`support.rs:186`, `syscall.rs:175-183`). Harmless because `supervised` waits for mise before returning, but mise has no use for the lease.
- Whether `MISE_CEILING_PATHS` excludes the ceiling directory itself (so `model/umpire/mise.toml` is not loaded during jar commands) cannot be confirmed from code; the retained real-install evidence covers the live behaviour.
- Verified clean: TLC SHA matches the retired script's pin; Apalache archive SHA matches `apalache_pins::ARCHIVE_SHA256`; `Refusal::from(io::Error)` keeps the raw message so the "missing pinned jar" hint still fires; digest runs before `mise where`, so corrupt jars refuse without spawning mise; `Scratch` is a unique tempdir so concurrent captures are independent; no-argument `fetch-tlc` resolves the compiled-in framework root, so the bridge script no longer needing `cd` is correct; no stale `build/tlc/tla2tools-1.7.4.jar` or `build/apalache/apalache-0.62.4` references remain outside the diff.

## Requirements coverage
| R-ID | Status | Evidence |
| --- | --- | --- |
| R7 | met | Pins and caches resolve under `model/umpire`: `mise-tools/` config and lockfiles, `build/mise-jars` data/cache/state, `ToolRoot` admission for the root; standalone copy proof at `task16/validation-60f1c762/summary.json` passed with unchanged source. |

Unaddressed R-IDs: []

Suppressed findings: 2 (ceiling-path inclusivity; offline behaviour of `mise where`, both at 25).
Classification counts: 4 introduced, 0 pre_existing.

```json
{"suppressed_count":{"25":2},"classification_counts":{"introduced":4,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>