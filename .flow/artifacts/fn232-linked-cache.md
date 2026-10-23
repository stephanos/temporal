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

Strict Clippy's owned issues were resolved by borrowed `Option<&str>` setters, a read-only authenticated `installation_cache` helper, a short local cache accessor in inspection, and an existing linked/cache/conditions/network admission extraction into `campaign::execution::admit`. Further metadata passes `fn232-linked-cache-clippy-final{,2,3}.log` (each exit 101) caught the extraction-local configuration reference and three test-style issues; all source fixes are in `450471f913`. The last failing pass reached helper integration tests and reported only the nonempty-byte assertion; the final typed form is `assert_ne!(original, [0_u8; 0])`. Final strict/affected reruns are now complete as recorded below. `fn232-linked-cache-rustfmt-final.log` exited 0 across 24 changed Rust files; diff checks pass.

The earlier replay poison-status RED used the 1-second fixture and can be confounded by debug executable hashing/watchdog precedence. It is superseded by clean causal proof on the final 10-second WAT fixture: `/home/agent/evidence/fn232-linked-cache-poison-red10s.log` exits 101 at `poisoned installed cache was ignored` with only the status-3 branch omitted. The exact temporary patch SHA-256 is `dcfed4214202d5fc3541a1a5641e69c51dfc1f6ad29a97e984efbad66017a449`; patch/source/free-space receipts are `fn232-linked-cache-poison-red.patch` and `fn232-linked-cache-poison-red-source.json`. Final CLI source was saved and restored byte-for-byte before final GREEN.

## Resources and remaining gates

Initial target debug size 4.1 GiB grew to at least 7.0 GiB; the original 2-GiB estimate was exceeded, explicitly reported, and root revised the existing-profile replacement allocation after root-owned cleanup. Global free space reached 2.4 GiB, then 1.9 GiB during the final all-tests metadata pass. Compilation stopped immediately when the floor breach was observed. No final link ran below the 2-GiB floor. Root subsequently reclaimed independently audited regenerable metadata/Go objects and authorized exactly two worker-owned test executable removals. `/home/agent/evidence/fn232-linked-cache-own-test-exe-cleanup.json` records no live `/proc` refs and exact path/size/SHA-256: 247,081,008-byte linked-cache test (`27d7b1f6…`) and 284,591,344-byte helper unit test (`cafb1ad6…`). No prior Sweep binary was removed. Root cleanup receipts are `/home/agent/evidence/fn232-retained-intermediates-cleanup-20261023.json` and `fn232-additional-go-objects-cleanup-20261023.json`. Before the clean RED link, actual free space was 5,825,613,824 bytes and conservative post-link projection 5,216,394,312; restored host link projected 4,889,869,384; final helper link projected 4,863,221,210. All exceed the 2-GiB floor. No other owner's files, target, binary, evidence or worktree was removed.

- Final strict Clippy passed: `fn232-linked-cache-clippy-restored-final2.log`, exit 0 across both crates' lib/tests. A first restored check found ambiguous empty-array comparison, corrected to `[0_u8; 0]` before this pass.
- Final lifecycle native tests passed 7: `fn232-linked-cache-lifecycle-final-green.log`, exit 0, including restored clean poison status-3 proof and non-cache capacity mismatch preservation.
- Final canonical identity/failure/payload/store-fuel regression passed 1: `fn232-linked-cache-identity-final-green.log`, exit 0.
- Final helper native tests passed 3: `fn232-linked-cache-helper-final-green.log`, exit 0. Final strict scope is both `gomad` and `gomad-wasmhost`, `--lib --tests`, default existing debug features, `-- -D warnings`; no diagnostic/reference-helper feature/profile stack was built. Final formatting checks cover all 24 changed Rust files; diff checks pass.
- Root integration and independent batch review; exact installed two-profile Go/runtime cache matrix, resume/inspection runtime reach, CPU/resource ceilings, and changed-candidate receiving/qualification gates remain root-owned and unrun here.
- Inventory owner was notified that existing cache effect sites are newly reachable from linked standalone/cluster compilation; no primitive site was added. Final source commit is supplied for exact append reconciliation.

No Go compiler, external public CLI process, production workload, full qualification, push, rebase, amend or Flow completion ran here. Worktree retained for root integration and independent review; no additional owned scratch roots are retained. All commands are terminal and the exclusive compile window is released. Final free space is about 5.0 GiB.

## Independent review repair (source checkpoint 8c93175ea0; tests pending)

The same-family independent three-axis Codex review at `fa1b541d11` returned NEEDS_WORK with four findings: `/home/agent/evidence/fn232-linked-cache-review.json`, session `01a1670e-5b99-7363-a36e-cb46b4e63948`, fanout `.flow/review-fanout/3cd31db417ee6f69758cd565629debbc`. Previous GREEN evidence above predates these repairs and does not qualify them.

- `d1fcd3e8c0` refuses compile-cache infrastructure and exact cache-artifact capacity before `execution::publication` and defensively at its entry, so launcher paths cannot put absolute cache-entry names in canonical outcomes. Host-only error diagnostics retain the detailed path. A real WAT launcher regression warms and poisons two roots, expects no finalized result/Receipt, and checks the prepared canonical record does not contain the cache location. Other infrastructure behavior is unchanged.
- `919048b492` restores cache selection in receipt-based `run-guest` only after existing module/argv/selected-installation hash guards. A private `admit_receipt` seam takes the already supplied digest without process-global environment mutation. A real retained receipt test proves selected-root restoration, detached receipt default `None`, source/compiler absence and changed-argv refusal.
- `a6516f6a81` integrates the core owner's unchanged inspection search extraction (`d1525f7b97`) while preserving the selected cache argument. `ebf294356a` adds typed Host versus Evaluation errors, classifies actual cache Outcome/cluster compile errors before instrumentation-divergence wrapping, and makes generic search callbacks return `Result`. Host errors propagate before predicate, report or deadline classification. Tests cover real standalone/cluster cache poison through both scan and bisect, exact cache-capacity classification, and host errors returned after cleanup deadline. Existing six pure algorithm tests retain their previous semantics through an Infallible test adapter; the core owner's separately held unverified-prefix semantic repair is deliberately not included.
- `8c93175ea0` integrates lifecycle owner `2048a06928`: jointly absent output descriptor/payload is empty; one-sided presence, unknown streams and declared identity mismatch refuse. Both lifecycle sparse-stream tests and this branch's cache tests are retained.

All repairs were written source-first under root's explicit instruction. Rustfmt/diff checks pass; no Cargo command has run on this repair checkpoint. Root owns the next native target lease, meaningful causal RED probes and final GREEN/strict checks. Core owner coordinates its provisional `Completion::{Finalized,Failed}` interface separately and must retain these non-publishable cache guards while adapting direct success-test call sites. The root must resume the same Codex review session only after committed GREEN fixes. No review finding is marked accepted/resolved by this worker's own narration.

### Review repair execution and frozen GREEN (2026-10-23)

The source-first repair checkpoint above is superseded by clean tested code source
`80aeaa17dca6bf1f1751d8550d3ebc18090e603a`. All code changes are committed;
this final handover append changes no code. Root must obtain the formal same-session
re-review before reporting the four findings resolved.

Root granted the existing DEBUG development target
`/Users/stephan/Workspace/skunkworks/gomad/.build-cache/fn232-native-checks`
and its `tmp` child for **all compiler/linker outputs**, with pinned Rust 1.99.1,
`--offline --locked -j1`, `CARGO_INCREMENTAL=0`, and no new features/profile.
Each variant used:

```sh
CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/gomad/.build-cache/fn232-native-checks \
TMPDIR=/Users/stephan/Workspace/skunkworks/gomad/.build-cache/fn232-native-checks/tmp \
CARGO_INCREMENTAL=0 cargo +1.99.1 test --manifest-path tools/gomad/rust/Cargo.toml \
  --offline --locked -j1 -p gomad --lib --no-run --message-format=json
```

Cargo JSON selected the exact test executable. The existing `driver.command`
process owner then executed only the named native functions with `--test-threads=1`,
preserving their own in-test concurrency. Every process/PID/argv/raw exit,
executable SHA-256, source HEAD plus temporary patch SHA-256, compiler JSON/stderr,
fixture-space preflight and sampled peak is retained under
`/home/agent/evidence/fn232-linked-cache-review-gates/`.

| Variant / gate | Actual result |
| --- | --- |
| `red1-causal`: only the two prepublication cache-refusal calls omitted | Exit 101, exact `cache failure produced a launcher publication`; a real canonical record containing the cache path was independently captured in `red1-canonical-leak.json` |
| `red3-causal`: only selected receipt cache assignment omitted | Exit 101, exact left `None` versus right selected cache root |
| `red4-causal`: only typed Host error folded into the old invariant result | Exit 101, exact `cache failure became an invariant search report`, with actual poisoned-entry checksum diagnostics |
| `core-search-red`: unchanged extracted predicate semantics | Six tests: five pass; only unverified-prefix predicate case fails its expected `replay_divergence` assertion |
| `green-launcher` | 1 passes; both cache roots refuse without a Receipt or canonical cache path |
| `green-receipt` | 1 passes; selected installation restored after guards, detached receipt remains uncached, changed argv refused |
| `green-inspection-cache` | 2 pass; real standalone/cluster cache poison through scan and bisect, exact cache capacity classification |
| `green-core-search` | All 6 pass; supplied single predicate clause applied after cleanup deadline guard, typed Host errors still propagate |
| `green-lifecycle` | All 8 pass; includes lifecycle owner's silent/stdout-only/stderr-only and malformed-stream regressions |
| `green-identity` | 1 passes; canonical record/failure/payload identity and store-fuel accounting |
| `green-callback` | 1 passes; unrelated output-callback infrastructure still preserves its original partial publication |
| `final-clippy` | Exit 0, both `gomad` and `gomad-wasmhost`, `--lib --tests -- -D warnings`, same host target/profile |
| `final-format` and diff check | Exit 0, all 28 changed Rust files |

The first build caught a test using campaign-private `execution::run`; its warm-up
now uses existing crate-visible `run_with_replay`. The first launcher probe caught
a fixture using ordinary campaign policy; it now matches actual launcher
`parallel=1` and `on_failure=first`. These setup failures were retained separately
and never counted as causal REDs. All final source files were restored exactly
before GREEN. The core owner supplied only the verified-prefix predicate clause
from `96aec9dd3fb47a40ef2f4a01b023d5548f5587bd`; its exact before/after source
hashes are in `core-prefix-source.json`. No unrelated Completion interface was
imported; root coordinates that consumer migration separately and preserves the
non-publishable cache-error guards.

Native fixture `TMPDIR` was exclusively
`/home/agent/evidence/fn232-linked-cache-tests-tmp`. No helper/compiler/executable
copy was made there. The conservative per-test fixture reserve was 16 MiB;
actual 10-ms-sampled allocated peak across the causal probes was 659,456 bytes
and across GREEN was 602,112 bytes. An unrelated external workspace consumed
native space, and the original 256-MiB preflight correctly refused before a test.
Root then explicitly revised **only these development units'** minimum to
64 MiB on the measured bounded footprint. Every subsequent preflight required
64 MiB plus the 16-MiB reserve; minimum sampled native free space during GREEN
was 179,748,864 bytes. Own fixtures were cleaned between commands; the final
fixture directory is empty. No ENOSPC observation counts as a causal failure.
Production bounds and the final 12-GiB native qualification gate are unchanged.

`final-summary.json` binds the actual 20 GREEN tests, raw receipts, source and
filesystems. Final free space at capture: native 3,375,575,040 bytes, host
97,096,622,080 bytes. Every process/session is terminal and the exclusive host
compile window is **released**. This worktree remains for root integration and
formal same-session re-review. No Go, production workload, external public CLI,
qualification pass, push, rebase, amend or Flow completion is claimed.
