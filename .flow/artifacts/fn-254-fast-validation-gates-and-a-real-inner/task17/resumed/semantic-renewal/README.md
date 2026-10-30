F3.1 compilation and identity checkpoint

Frozen ROOT: ede5fadc613e66ed33d083b4141a4ccf9dd2b76a. All code reads and builds used ROOT; this worktree holds artifacts only. No Java, model, rustdoc, or other test bodies ran.

Native tools were refreshed through the production framework bootstrap and targeted Temporal build. `successful-build-final.json` binds the final native binaries to the exact 66 binary and 78 broader source inputs after inventories. Before/after normal CLI hashes remain in the initial/final build receipts; Cargo may rebuild normal binaries while compiling test harnesses.

Inventories use the actual shipping profile: CARGO_PROFILE_* is absent, test profile is workspace line-tables-only, compiler jobs 1, incremental 0, GOMAXPROCS 1 and final GOFLAGS -p=1. The native refresh separately uses dev/debug0. Every emitted harness was listed with all and ignored libtest options; proc-macro loader paths come from the pinned rustc target library directory and debug/deps. No test body success is inferred from these lists.

Actual Cargo/nextest equality:

- Five-package activity selection: 148 identities, 0 ignored, 28 binary footprints. Temporal 118, API 21, realization macros 8, exact Temporal fixtures 1; guest 0 is explicitly compiler-only. Three eligible doctest targets are recorded separately.
- Framework affected packages: 165 identities, 0 ignored, 5 binaries: process 52, xtask 113. This adds 5 process and 8 xtask identities to F2; none removed.
- Temporal xtask: 21 identities, 0 ignored, 2 binaries. Added missing-tool parent/child and invalid-cwd controls replace the former spawn-failure test identity; the exact+3/-1 delta is retained.

F2 full inventories 860/235 plus these deltas project 873/237 identities. These are historical projections, not freshly listed whole workspaces or execution claims. The activity 148 scope is freshly listed against matching Cargo and nextest scopes. Root has separately identified that the locator-only process control must close inherited FD10/11 before executing its body; listing identities is unaffected.

All 1,566 tracked compilation inputs, 202 snapshots, both Cargo locks and 119 registry tuples are unchanged across the phase. Root was clean when released. Combined target+privateGoCache growth was 1,580,206,876 bytes, below the 3 GiB cap; final free space 13,859,491,840 bytes. Both cache and source freeze were released before Root's separate control fix.

No real six/five quick workload, production standalone, cold/warm baseline, or Darwin cleanup completion is claimed.
