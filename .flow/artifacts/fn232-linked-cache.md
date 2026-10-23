# Installed linked compiled-cache preservation — fn-232.13

Base `9d54dfc9dc`; branch/worktree `fn232-linked-cache`, `/home/agent/wt/fn232-linked-cache`. Source checkpoint `79b018aafa`; lifecycle guard checkpoint `08cffa8c73` integrated as `68bcc7c69f`. Caller/refinement checkpoints `ca2f6d0a66` and `450471f913` complete the implementation source. R9/R12 remain open; no qualification or review verdict is claimed.

## Behavior and identity

Installed admission `compile_cache_root` previously never reached linked execution. The original installed-operator source at `1d695061b6382009d0659238c652a5a765b1d5f5`, captured verbatim in `/home/agent/evidence/fn232-test-wasm-reach/host-contract-map.json`, requires one private `.gomad-cache-v3` publication, selected-installation source-less replay reuse, poisoned CRC refusal (status 3), symlink refusal (status 3), and successful restored replay for both stock-exploratory and cooperative profiles. The lifecycle owner retains the actual Go guest matrix.

The selected installation supplies a typed host-only `Prepared.compile_cache_root`. Initial preparation, standalone execution, clusters, simulation, replay, resume and reconstructed inspection executions forward it. Existing resume entry points default to `None`; no-selection detached replay/resume stays uncached and never opens old compiler/source/installation paths. Selected replay/resume first passes the existing retained implementation-context guard. No cache setting enters Configuration, source build keys, provenance, canonical records, failure signatures or choice identities. Provenance is a record stream and therefore cannot carry the cache path.

Linked preparation validates absolute cache paths without I/O; actual execution reuses `cache::module`. Its loader now accepts borrowed bytes through `B: AsRef<[u8]>`, avoiding a new up-to-512-MiB module clone on misses. Existing same-UID/private-mode checks, no-follow/single-link checks, bounded reads, engine/platform/executable identity, CRC, serialization, atomic staging/publication and storage-full behavior remain. No new dependency, environment/global cache control, host process, OS primitive or unsafe block was added. Cache infrastructure failures and exact cache-artifact capacity return replay status 3 before guest divergence comparison; unrelated capacity mismatch behavior is retained.

## Evidence

Pinned toolchain Cargo/Rust 1.99.1; locked/offline; jobs 1; unchanged existing debug profile/features. Exclusive target `/home/agent/.cache/codex-build/fn232-mandatory-workloads`. Commands use `cargo +1.99.1 ... --locked --offline --manifest-path tools/gomad/rust/Cargo.toml`, with `CARGO_TARGET_DIR` and `CARGO_BUILD_JOBS=1`. Test commands below end `-- --test-threads=1`. Raw logs and adjacent numeric `.exit` files live in `/home/agent/evidence/`.

| Command suffix | Log | Numeric exit / result |
| --- | --- | --- |
| `test -p gomad-wasmhost --test linked_compile_cache` on no-op setters | `fn232-linked-cache-red3.log` | 101; intended entry-count 0 vs 1 for standalone/cluster, relative path wrongly admitted |
| `test -p gomad --lib compile_cache_tests` before host forwarding | `fn232-linked-cache-host-red2.log` | 101; canonical identity comparisons pass, selected cache directory absent |
| `test -p gomad-wasmhost --test linked_compile_cache` after wiring | `fn232-linked-cache-green.log` | 0; 3 passed |
| `test -p gomad --lib selected_cache` before installed replay caller | `fn232-linked-cache-host-caller-red.log` | 101; identity regression passes, selected installation cache absent |
| `test -p gomad --lib installed_lifecycle_selected_cache` after caller, before error classification | `fn232-linked-cache-replay-classification-red.log` | 101; publication succeeds but poisoned cache incorrectly returns normal replay result |
| `test -p gomad --lib selected_cache` after classification | `fn232-linked-cache-host-green2.log` | 0; 2 passed |
| `test -p gomad-wasmhost --lib cache::` | `fn232-linked-cache-library-green.log` | 0; 7 passed, original manual CRC throughput measurement ignored |
| `test -p gomad --lib installed_lifecycle` | `fn232-linked-cache-lifecycle-green.log` | 0; 7 passed |
| `clippy -p gomad -p gomad-wasmhost --lib --tests -- -D warnings` | `fn232-linked-cache-clippy{,2}.log` | 101; owned setter argument / two function-size issues; source fixes made, final rerun pending |

The three helper tests execute actual small WAT modules: default disabled/relative refusal, exactly one regular mode-0600 entry, cold compile/warm cache hit with zero compilation, unchanged bytes/evidence, poison refusal without fallback, symlink and hardlink refusal, restored reuse, and cluster publication/infrastructure refusal. The host regression compares complete canonical records, failure signatures and payloads for disabled/two different roots/warm execution and observes exactly one store with unchanged fuel. Existing qualification process CPU math is untouched: cache work remains in measured linked-process self CPU; no fictitious helper-process CPU is added. This is not an empirical workload CPU qualification claim.

Early RED1/2 exposed a missing PWD in the new cluster fixture, corrected before meaningful RED3. Initial host RED exposed an omitted new constructor field, corrected before meaningful RED2. One GREEN attempt exposed the lifecycle fixture's 1-second wall bound being shorter than debug executable hashing; the new cache-specific WAT fixture has its own 10-second bound, while existing lifecycle fixtures remain at 1 second. No production/operator/qualification bound changed.

Strict Clippy's owned issues were resolved by borrowed `Option<&str>` setters, a read-only authenticated `installation_cache` helper, a short local cache accessor in inspection, and an existing linked/cache/conditions/network admission extraction into `campaign::execution::admit`. Further metadata passes `fn232-linked-cache-clippy-final{,2,3}.log` (each exit 101) caught the extraction-local configuration reference and three test-style issues; all source fixes are in `450471f913`. The last pass reached helper integration tests and reported only the nonempty-byte assertion; it is now `assert_ne!(original, [])`. Final strict/affected reruns remain required. `fn232-linked-cache-rustfmt-final.log` exited 0 across 24 changed Rust files; diff checks pass.

Causal evidence caveat: the earlier replay poison-status RED used the 1-second fixture and can be confounded by debug executable hashing/watchdog precedence. The clean missing-cache REDs and 10-second fixture GREEN are retained. Before claiming an isolated replay-classification RED, temporarily omit only the status-3 branch on the current 10-second fixture, retain the expected failure, restore source, and rerun GREEN.

## Resources and remaining gates

Initial target debug size 4.1 GiB grew to at least 7.0 GiB; the original 2-GiB estimate was exceeded, explicitly reported, and root revised the existing-profile replacement allocation after root-owned cleanup. Global free space reached 2.4 GiB, then 1.9 GiB during the final all-tests metadata pass. Compilation stopped immediately when the floor breach was observed. No final link ran below the 2-GiB floor. Root was notified of the actual growth and offered ownership-confirmed reclamation of this worker's newly built helper/cache test executables, with hashes/logs retained; no cleanup ran without that coordination. No other owner's files, target, binary, evidence or worktree was removed.

- Final strict Clippy, formatting/diff checks and affected native reruns after borrowed API / resume / inspection refinements.
- Root integration and independent batch review; exact installed two-profile Go/runtime cache matrix, resume/inspection runtime reach, CPU/resource ceilings, and changed-candidate receiving/qualification gates remain root-owned and unrun here.
- Inventory owner was notified that existing cache effect sites are newly reachable from linked standalone/cluster compilation; no primitive site was added. Final source commit is supplied for exact append reconciliation.

No Go compiler, external public CLI process, production workload, full qualification, push, rebase, amend or Flow completion ran here. Worktree retained for ongoing implementation/integration; no additional owned scratch roots are retained.
