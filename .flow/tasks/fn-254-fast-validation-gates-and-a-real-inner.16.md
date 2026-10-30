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
Earlier owner stop and resource interruptions remain preserved in `.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/session-wrap-20261027/`; the explicit milestone resume and successful proof below supersede that hold.

Provider source checkpoint: `60f1c762a618d32478b58ae4e79c151f4355aeb3`, reviewed against `f0a82c7a969ec6fd1717a73f98536d3b9c6fe8ee` (three-lens formal review finalized SHIP).

Framework-owned isolated mise HTTP declarations and exact-version lockfiles replace custom TLC download and Apalache download/extraction. TLC remains required, Apalache explicitly optional, and execution cannot auto-install. Runtime regular-file, containment/no-symlink and separate JAR-hash admission remain. Mise owns archive extraction/coordination and its best-effort leftovers; native bounded capture cleanup and existing .17 process custody remain. Canonical provider roots that are non-UTF8 or cannot be represented as one path-list entry now fail explicitly before mise/data/capture side effects; guest-driver raw OS-path contracts remain.

Fresh exact-source Linux standalone proof passed all17child stages in978.949s, with909nextest inventory entries:907executed and2ignored. Four doctests passed and one was ignored, yielding914discovered/911executed/3ignored/0skipped. Parent/child receipt hashes, exclusive authority and absence of parent double charging were checked. Before/after source snapshots match; all observed owned processes are absent. Four external-watchdog controls passed, both actual filesystems were initially admitted at>=50GiB and remained above the unchanged30GiB floor; retained harness/growth guard bytes did not change.

Current proof and curated complete logs: `.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task16/validation-60f1c762/`. Historical focused controls remain labeled:267/267 repaired framework controls,16/16 system caller controls and5/5 changed-root/path-list controls (plus32/32 provider renewal). The19checked material identities justify reuse of real TLC and actual Apalache clean installs/execution at their recorded D4 source;76receipts/148stdout-stderr hashes and downloaded/installed assets were independently verified. Current full proof revalidates the final-source required-TLC standalone path. No old pass is relabeled current.

No broad final combined gates or task8timing acceptance is claimed. Darwin runtime remains owner-deferred. Tasks15/12/8 remain open. Migration remains owner-deferred and in-tree framework authority remains.

Tier: implementer - gpt-6.1-sol (explicit invocation) at medium effort (requested); actual host model not reported. Formal CLI reviewer: configured claude:claude-fable-5-1:high, different family from Codex writer.

Formal review `d56cefa991c84615bd07f20be7556d9b`: correctness/contracts/integration all SHIP. Raw findings and coordinator format repair are retained under task16/review-60f1c762. Runtime artifact reads were denied in the reviewer sandbox; the conductor independently verified the accepted proof, historical reuse identities and recovery. Nonblocking extra bootstrap belongs to the existing task15 wrapper retirement and task8 cost measurement. Other nonblocking suggestions concern future pin-drift guards, diagnostics, documentation and simplification; no current release inconsistency or failing task acceptance remains.

Proof binaries are independently recoverable from `.recovery/fn25416-provider-proof`: one 338,000,596-byte CAS object, 3,548 exact members including all 69 test binaries. Targets remain intact until post-integration cleanup.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 98e586323f5058350139a14ab4f50e677ce9f89c, cf3fc2d7905f0ce7d1edf7d994056b5a343e24d6, f2e0f4276139b8f7c93a16f8b74a268ec8e64435, b8ae8a23617911a7e20af3bf2f5dbc6c43d54c3b, d89b10d52da97b10924306cb25860b19aef733ff, d4a79702ce170f35698fe736fb72792f55c2d3af, 3b9f41e8dd4da9bf54743240a2f67ba7ed4c2e84, 3368481d75b40dd49aea5a5dc7b54467e3779052, 60f1c762a618d32478b58ae4e79c151f4355aeb3
- Tests: env -u BASH_ENV PYTHONDONTWRITEBYTECODE=1 python3 /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn25416-resume-20261028-watchdog-01/watchdog.py, PYTHONDONTWRITEBYTECODE=1 python3 /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn25416-resume-20261028-watchdog-01/test_watchdog.py
- PRs: