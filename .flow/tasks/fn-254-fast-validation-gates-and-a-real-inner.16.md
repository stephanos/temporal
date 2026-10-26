---
satisfies: [R7]
---
# fn-254-fast-validation-gates-and-a-real-inner.16 Provision TLC and Apalache downloads through mise

## Description
Owner request: file the mise download work in fn-254. Replace the bespoke TLC JAR download and Apalache archive download/extraction with framework-owned mise HTTP backend declarations. Reuse the feasibility investigation in `.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/planning/mise-provisioning/report.md`: real TLC checksum admission works; an already-installed corrupted JAR survives `mise install --locked`, so runtime JAR hashing remains necessary. Real Apalache provisioning was not exercised by that investigation.

**Size:** M
**Files:** `model/umpire/mise*.toml`, corresponding lockfiles if adopted, `model/umpire/dev/xtask/**`, `model/umpire/scripts/fetch-tlc.sh`, `model/umpire/backends/process/src/jar.rs` and focused admission tests, backend tool-path call sites, remaining root fetch bridge, framework README and guest-build documentation. Remove Cargo dependencies only when the deleted installer was their last consumer.
**Touches:** [model/umpire/mise*.toml, model/umpire/mise*.lock, model/umpire/dev/xtask/**, model/umpire/scripts/**, model/umpire/backends/process/**, model/umpire/backends/tlc/**, model/umpire/backends/apalache/**, model/umpire/Cargo.toml, model/umpire/Cargo.lock, model/scripts/fetch-tlc.sh, model/README.md, model/GUEST_BUILDS.md, model/umpire/**/README.md]

Runs after .13's workspace/tool extraction and .14's tool-probe removal. Task .10 may already compose the framework gate against .13's stable provisioning commands; this task preserves those public entry-point/status contracts while replacing provider internals. Renew affected gate/provider caller checks for changed paths or admission inputs. Preserve .13's frozen evidence as evidence of its exact earlier inputs; implement from the current integrated closure and reproduce changed identity inputs when required.

- Pin the existing TLC 1.7.4 and Apalache 0.62.4 official assets and download SHA-256 values in framework-owned mise configuration. Apalache belongs to an explicit optional environment; routine installs/gates must not install or require it. No vendored archives/JARs or new installer framework.
- Delegate fetching, archive extraction and installation coordination to mise. Resolve exact installed versions through the admitted framework context. Avoid a second custom cache/copy layer. Any jar-specific mise data/cache/config overrides stay scoped to provisioning/lookup, without moving ordinary Rust or Java installations or reading Temporal parent configuration.
- Retain runtime JAR SHA-256, regular-file and contained-path/no-symlink admission, including the Apalache JAR hash separately from its archive hash. Respect .14: do not restore mise/tool-presence, Java-version or protoc-version preflight probes. Missing required artifacts fail at actual resolution/admission; successful unrelated `mise exec` alone proves nothing.
- Preserve explicit provisioning versus execution with auto-install disabled, backend modes, failure status and public entry-point behavior. Map old installer controls to replacement controls, with explicit reasons for delegated or obsolete cases. Exercise corrupted/partial installs, bad checksums, path escape/symlinks, interruption/concurrent installation and cleanup to the extent required by existing contracts; do not assume mise has identical cleanup semantics.
- Task .15 consumes this implementation and ports only the remaining wrapper logic; it must not recreate native curl/tar downloaders. Guest-driver independence remains intact. Focused provider/admission checks run here; unrelated full model suites and final timings stay in .8.

## Acceptance
- [ ] Framework-owned mise declarations provision the exact pinned TLC and Apalache assets with download checksums. Record one actual clean TLC install and one actual clean Apalache install, then successful runtime JAR admission/execution; the prior tiny archive fixture is insufficient for Apalache.
- [ ] Default installation and routine gate controls exclude Apalache; explicit opt-in works. Missing required artifacts and corrupt installed JARs fail even when mise reports the version installed. No removed .14 prerequisite/version probes return.
- [ ] Custom download/extraction plumbing is removed. Existing status, cleanup, path safety and concurrency obligations have a checked control mapping and focused passing evidence, with any changed contract explicit. No vendored binary/archive or parent Temporal configuration dependency is introduced.
- [ ] Focused xtask/process/backend provisioning tests and affected fmt/clippy pass. Documentation and entry-point callers use the mise provider; broad model gates/timing remain assigned to .8 and directory-only gate proof to .11.


## Actual release feasibility

The isolated follow-up successfully installed the actual official Apalache release through mise HTTP, independently matched the archive/JAR hashes and ran the admitted JAR `version` command under pinned Java. Exact configs, resolver environment, statuses and limits are retained at `.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/planning/mise-provisioning/real-apalache/report.md`. This is preparatory evidence, not a passing implementation/provider test. Future changed resolver/admission inputs need renewed relevant controls; production .14 probes remain removed. The installed data stays outside Git for reuse.

## Throughput-first order

Run after priority .17. Preserve its single stage graph, public entry points and nextest/resource contract while replacing provisioning internals. Download cleanup must not delay .5/.17's validation speedups or recreate orchestration in the tool provider.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
