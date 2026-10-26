Task14 source is committed as `5ba482b5b73ca4d85ab758568d9a25ec4c4126b4`. Mise and pinned tools are now assumed installed: bootstrap and proto generation execute directly, guest readiness preflights are removed, and Java admission resolves only through `mise which java`. Actual tool execution, disabled auto-install, archive/jar hashes and guest cache identity remain.

Removed obsolete negative controls and installed-tool fake setup. The bootstrap control proves missing mise fails with the actual command-not-found/bootstrap error, no Cargo execution, no native binaries and no guest publication. Process positive admission uses its real enclosing workspace pin; it no longer copies an untrusted temporary mise file or skips after preflight failure.

Validation passed: 110 native guest-driver/process/xtask tests; seven TLC runner controls and one required real TLC initial-counterexample check (118 Rust tests total); scoped all-target clippy; three shell control suites; edited-file rustfmt/bash syntax and diff checks. The real TLC check ran with `UMPIRE_BACKENDS=require`, passed in 1.79 seconds, and released the exclusive JVM lease directly to .6. No broad model or Go gate ran.

Two fresh independent targets rebuilt the actual native driver and Temporal guest from the exact committed candidate. Both drivers hash `b1210c27a5bfe8cd30d5e238fe9ed4341f3d4fc04f98053a527e0593ad60a09c`; both cache keys are `ae498134b457fa5c38222e7f6eb775bea352e2caeef9c9e50cd7adde27f2a67d`. Old committed and both rebuilt WASM hashes are `c83e883a38647cdc988ace1b32efbccd546a4a1f2acbe10d53485e7d8438c4cc`; exact byte comparisons pass. Neither WASM nor canary policy needed publication. Source, driver, tool and key manifests and raw logs are beside this summary.

Task13 path/remap proof is reused only for unchanged mappings. Task14 changes native driver bytes and explicitly re-proves native identity and guest reproduction; no old-driver freshness result is reused.

The required scan has four remaining harmless hits: mise.toml's pin, the framework design table, TLC's integrity-refusal fixture setup and a frozen disabled Scala fixture. No active removed presence/version/readiness probe remains.

No manifest, lockfile, independent Model, Flow lifecycle or milestone file changed. Two implementer subagents handled disjoint guest and shell surfaces. The coordinator owns the sole commit; the worktree remains for root review/integration.

`proof-index.json` records exact hashes and sizes for all retained proof inputs, logs, binaries and WASM artifacts. The root conductor owns durable Flow proof/closure projection and source review. No validation or implementation work remains in this worker lane.
