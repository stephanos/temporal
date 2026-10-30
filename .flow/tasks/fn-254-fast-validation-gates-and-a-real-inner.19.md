---
satisfies: [R9, R7]
---
# fn-254-fast-validation-gates-and-a-real-inner.19 Enable measured local sccache reuse across worktrees

## Description
Pin sccache through mise and integrate local host compilation caching with the existing admitted command/context paths for cold worktree and bootstrap builds. Prove reuse between different checkout paths with separate Cargo target directories while preserving the ordinary warm quick path's incremental compilation.

**Size:** M
**Files:** `mise.toml`, `mise.lock`, `model/umpire/mise.toml`, `model/umpire/mise.lock`, framework quick/bootstrap and guest-driver compiler/context modules, maintained build docs.
**Touches:** [mise.toml, mise.lock, model/umpire/mise.toml, model/umpire/mise.lock, model/umpire/.cargo/config.toml, model/.cargo/config.toml, model/umpire/scripts/xtask.sh, model/scripts/xtask.sh, model/umpire/dev/xtask/src/main.rs, model/umpire/dev/xtask/src/quick/**, model/umpire/dev/guest-driver/src/**, model/umpire/dev/guest-driver/tests/**, model/umpire/dev/xtask/tests/**, model/umpire/dev/xtask/README.md, model/umpire/dev/guest-driver/README.md, model/GUEST_BUILDS.md, model/design/gate-speed.md]

### Approach
- Use a pinned local disk cache, never a remote cache service. Keep framework pins/config independent of the Temporal parent; serialize root mise changes with fn-256.2 and other pin owners.
- Trace quick's wrapper/profile sanitation (`quick/mod.rs:168`), xtask startup (`main.rs:41`) and bootstrap (`scripts/xtask.sh:22`) before wiring the tool. Preserve bootstrap isolation: ambient wrappers/flags must not become authority merely to enable caching. Compose the admitted cache path explicitly through existing context/command ownership rather than adding another launcher/configuration owner.
- Disable rustc incremental compilation only on explicitly selected cache-enabled cold worktree/bootstrap paths. Keep ordinary warm quick re-edits incremental, with no automatic cache wrapper replacing that policy. Normalize equivalent checkout roots with the pinned tool's supported base-directory/remap behavior, retaining source/compiler/lock/profile/target identity. Preserve independent Cargo target directories and leases.
- The guest driver owns its compiler wrapper (`guest.rs:135`, `compiler.rs:366`). Preserve its previous-wrapper chain, exact compiler/driver/lock admission and guest cache invalidation. Prevent recursion and host cache flags from changing WASI/guest or native parity outputs. A documented guest-cache bypass is acceptable; weakening the existing wrapper is not.
- Measure an uncached control, cache population and a second equivalent checkout with an empty independent target directory, using isolated stats/cache state that cannot reset another session's daemon or counters. Retain actual hit/miss/non-cacheable deltas, command/env/tool/profile/target inputs and wall times. Distinguish hits from Cargo Fresh and final link work. Exercise changed-source invalidation and wrapper/bootstrap boundaries.
- Update build guidance and the comparison table. Renew affected standalone-tool controls; .8 owns the final whole-gate/standalone proof. Do not resume fn-258 or owner-paused .12.

### Investigation targets
**Required:**
- `model/umpire/dev/xtask/src/quick/mod.rs:168` — command environment admission.
- `model/umpire/dev/xtask/src/main.rs:41` and `model/umpire/scripts/xtask.sh:22` — wrapper sanitation/bootstrap.
- `model/umpire/dev/guest-driver/src/guest.rs:135` and `model/umpire/dev/guest-driver/src/compiler.rs:366` — owned wrapper chain.
- `model/umpire/dev/guest-driver/src/context.rs:184` — existing remap/context ownership.
- Both mise manifests/locks — local pins and standalone ownership.

### Key context
Rust incremental compilation must be disabled for sccache; linker-invoking crates are not cacheable, and procedural macros reading files need explicit care. Sources: https://github.com/mozilla/sccache/blob/main/docs/Rust.md and https://github.com/mozilla/sccache/blob/main/docs/Configuration.md . Verify support against the selected pinned release rather than assuming main-branch features exist there.

## Acceptance
- [ ] Mise pins and local cache wiring work through actual affected host cold-build/bootstrap entry points in both workspace contexts, without a parent dependency or restored ambient-wrapper trust. Ordinary quick re-edits retain incremental compilation; matched same-checkout Model-edit measurements report any warm-loop regression separately from cross-worktree savings.
- [ ] Equivalent source in different checkout paths and separate initially empty target directories yields actual positive Rust cache-hit deltas; retained statistics distinguish misses, non-cacheable compilation, final links and Cargo Fresh. Record timings and costs without a prescribed speedup percentage.
- [ ] Focused controls prove changed-source invalidation, no wrapper recursion, isolated bootstrap behavior and preserved guest-driver/compiler/lock identity and guest/native-parity behavior; supported cache inputs are documented.
- [ ] Negative controls vary compiler identity, profile and target separately and prove incompatible compilation is not reused. An unavailable/unwritable cache store is exercised: tool failures and fallback/error statistics are surfaced explicitly, and a local-compile fallback cannot satisfy the cache-hit proof or pass unnoticed.
- [ ] Cache measurement and cleanup leave other sessions' stats, daemon state, caches and target leases intact; final gate proof remains .8.

## Done summary
Blocked:
Sccache 0.19.0 cannot safely normalize Rust workspace compilation across physical checkouts. The project-pinned Rust 1.99.1 A/B/A control records two checkout misses followed by a same-checkout hit, with separate output directories and private cache/daemon. Released Rust hashing retains physical cwd, arguments and Cargo environment. Upstream normalization proposals remain unmerged; normalizing keys alone can reuse checkout-specific ToolRoot authority. Preserve full acceptance; no production cache wiring or dependency-only substitution. Evidence: .flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task19/{measurements.json,research.json,reproduce.py}. Other milestone lanes continue.
## Evidence
- Commits:
- Tests:
- PRs:
