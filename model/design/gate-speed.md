# Validation gate costs

**Decision, 2026-10-25:** use historical evidence to choose fn-254's optimizations.
The owner waived fresh baseline measurements: “can we ignore baseline measurement?
we already have historic data”. No fresh gate, build, test inventory or TLC measurement
was run for this report. Final measurements after optimization remain required by R6.

The historical runs establish that workspace tests dominate and that repeated backend
startup and compile-time UI checks deserve attention. They do not establish today's
quiet-host latency, a compile/link breakdown, or an exact current migration inventory.
Unknown costs below remain unknown; historical pass receipts do not prove current source.

## Evidence and comparability

Paths in this report are relative to the repository root unless marked otherwise.

| Reference | Command and result | Source and conditions |
| --- | --- | --- |
| H247 | `BASH_ENV=/dev/null UMPIRE_APALACHE=require make model-validate`; exit 0; outer **2092.924 s**, gate **2081.715 s**, checker benchmark **10.813 s** | `0925064e4fe346191addf0c79b534e78fc7d6ab6`; TLC **require**, Apalache **require**; Linux/aarch64, shared target. [Receipt](../../.flow/artifacts/fn-247-one-language-for-the-framework-a-rust/task3/validation.json). |
| H237 | `make model-check` **441.325 s**, `UMPIRE_BACKENDS=require make model-check` **960.602 s**; both exit 0 | Activity implementation `e04f0dc1a3a28d33c18ebd94b6e45ac039b01b98`; older source/coverage. Apalache skipped. Default and required runs were separate executions, not a controlled experiment. [Evidence](../../.flow/artifacts/fn-237-activity-timeouts-heartbeat-and/activity-policy-implementation-evidence.json). |
| H237-time | `make model-time`; exit 0; outer **373.928 s**, gate **367.163 s**, checker benchmark **6.566 s** | Same task's separate default-mode execution. Timings differ from H237's standalone default gate; do not mix their stage totals. [Task record](../../.flow/tasks/fn-237-activity-timeouts-heartbeat-and.3.md). |
| H251 | `fn2511-dependents.log` and `fn2511-r2-dependents.log`; individual harness groups report success | Isolated `fn2511-graph` worktree, target `fn2511-target`, unoptimized profile with debuginfo; exact invocation environment, quiet-host status and source binding are not in these two raw logs. Their individual result lines do not establish an independently observed command exit. Preserved under `.flow/tmp/fn254-baseline/historical/`. |

H247's [tool record](../../.flow/artifacts/fn-247-one-language-for-the-framework-a-rust/task3/tools-platform.log)
records Linux `dev-build 7.0.14`, aarch64, rustc 1.99.1 (`58cf8f9c3`), Cargo 1.99.1,
Go 1.27.0, protoc 29.5, mise 2026.10.11 and Temurin 27+35. Its target was
`/home/agent/.cache/codex-build/cargo-target`; that historical path is not authorization
to reuse or clean it. Core count, memory, complete profile overrides, sustained host quietness,
and comparable cache warmth are unknown for that run. H237/H251's matching host-resource
and cache-state records are unavailable here. None supplies native Darwin certification.

The current canonical Makefile has native tooling stages introduced after H237, and fn-246
adds coverage after H247. H247 also required deferred Apalache, unlike routine fn-254 use.
Consequently **519.277 s** (H237 required minus default) is only an observed difference,
not measured TLC-only cost; **396.35 s** of H247's tests is an Apalache harness group,
not a measured prediction of today's routine gate.

## Per-stage historical times

All values are seconds. H247 is one complete required pipeline; H237-time is one separate
default gate. “—” means the stage was absent under that name in the retained timer output,
not zero work. H237-time values are retained in
`.flow/tmp/fn254-baseline/historical/fn2373-final-model-time.log`;
H247 values come directly from its committed receipt above.

| Stage | H247 TLC + Apalache required | H237-time default |
| --- | ---: | ---: |
| temporal-guest-freshness | 61.934 | 32.825 |
| retired-names | 4.582 | 0.772 |
| timing-controls | 2.091 | 0.033 |
| guest-build-controls | — | 6.443 |
| capability-controls | 1.039 | 0.085 |
| source-boundary | 43.054 | 4.089 |
| protobuf-controls | unknown: no separate timer row | — |
| protobuf-freshness | 0.544 | 0.080 |
| kettle-protobuf-freshness | 0.454 | 0.076 |
| temporal-protobuf-freshness | 0.210 | 0.054 |
| fmt | 1.596 | 0.611 |
| clippy | 39.662 | 3.747 |
| docs | 70.494 | 5.166 |
| tests | **1707.092** | **214.114** |
| wasm32-wasip1 | 1.227 | 0.091 |
| core-wasm-clippy | 4.105 | 0.089 |
| wasm-guest | 78.845 | 45.350 |
| go-host | 2.550 | 2.220 |
| kettle-go-host | 42.423 | 40.009 |
| temporal-go-host | 13.024 | 11.099 |
| gate total | **2081.715** | **367.163** |
| edited-checker benchmark, outside gate | 10.813 | 6.566 |

H247's tests consumed **82.0%** of its gate (1707.092 / 2081.715). Both historical
gate totals missed the old 60 s soft budget. H247's benchmark missed 10 s; H237-time's
met it. The benchmark touches checker source without changing bytes; it is not an authored
activity edit, a TLC cache measurement, or evidence for the proposed `model-quick` target.

## Binary runs, build costs and slow tests

The H247 [workspace observation](../../.flow/artifacts/fn-247-one-language-for-the-framework-a-rust/task3/workspace-tests-observed.log)
retains harness summaries under `cargo test --workspace --locked --quiet`. Quiet output
omits Cargo executable/package headings. The first three identities below are attributed
through their distinctive test names/source and group sizes, not binary hashes. Repeated
observer text is not counted as additional execution.

| Historical group or exact binary | Run interval (s) | Compile / link costs | Identity and timing limit |
| --- | ---: | --- | --- |
| H247 Temporal fixtures, 33 passed / 1 ignored | **581.72** | unknown / unknown | Named `activity_tests::*` and `fixtures` group; includes required TLC; no executable hash or individual timestamps. |
| H247 Apalache controls, 14 passed | **396.35** | unknown / unknown | Named `real_kettle_*` / arithmetic controls; opt-in backend. |
| H247 shared fixtures, 47 passed | **213.52** | unknown / unknown | Named world-fairness/action-input controls; required TLC; no current-inventory equivalence claim. |
| H251 `ui-1e2485a61e57b59a`, `tests/ui.rs` | **162.67**, repeat **112.81** | unknown / unknown | Temporal annotation admission; includes nested trybuild compilation. Raw log lines 537 / 422. |
| H251 `probes-d25949aae1f9679e`, `tests/probes.rs` | 20.45, repeat 16.56 | unknown / unknown | 12 tests; raw log lines 265 / 190. |
| H251 `realization_admission-8f2bdc25b8cfa059` | 17.30, repeat 14.22 | unknown / unknown | 1 test; raw log lines 352 / 277. |
| H251 `supplemental_admission-d8fcb1a902729231` | 16.22, repeat 13.41 | unknown / unknown | 1 test; raw log lines 379 / 304. |
| H251 `phase_exhaustiveness-2ba82ab9009f9dd6` | 15.83, repeat 17.59 | unknown / unknown | 1 test; raw log lines 247 / 172. |
| H251 `exhaustiveness-40a30b020ad15a78` | 14.83 | unknown / unknown | Kettle, 1 test; raw log line 131. |

H251's first Cargo test build reports **25.00 s**, and the repeat reports **15.64 s**,
before their executable runs. These are aggregate Cargo build intervals including reused
dependencies; they cannot be assigned to individual binaries or separated into compiler
and linker time. Nested trybuild build summaries (for example 7.17 s inside the 162.67 s
UI harness) belong inside the run interval and must not be added twice. Complete per-binary
compile/link attribution and fresh warm-build costs remain unknown because no fresh build
was authorized.

Exact per-test elapsed times are unavailable. The retained slow-test evidence is:

| Test / source group | Historical observation | What is unknown |
| --- | --- | --- |
| `declarations_reject_unclassified_or_incompatible_shapes` (H251 annotation UI) | Sole test in the 162.67 s group; repeat 112.81 s | Test-only elapsed excluding harness/trybuild setup. |
| `activity_tests::policy_checks::selected_deadline_worlds_agree_compiled_interpreted_and_required_tlc` | H247 libtest “over 60 seconds” notice | Exact elapsed and ranking within the 581.72 s group. |
| `activity_tests::proof_checks::actual_tlc_selected_fairness_and_starvation_lassos` | H247 “over 60 seconds” notice | Same. |
| `world_fairness_tests::world_fairness_required_tlc_mutants_replay_heartbeat_and_starvation` | H247 “over 60 seconds” notice | Exact elapsed within the 213.52 s shared group. |
| `real_kettle_checks_source_safety_and_history` | H247 “over 60 seconds” notice | Exact elapsed within the 396.35 s Apalache group. |

These notices identify slow work; they are not a complete slowest-test ranking. Stable
libtest 1.99.1 does not offer supported JSON/report-time output. The optional scratch
observer timestamps flushed partial bytes in a serial run; that changes concurrency and
would need an explicit instrumented label. No unstable/bootstrap timing was used here.

## TLC launches and coverage gaps

| Fixture / route | Actual launch count | Evidence available |
| --- | --- | --- |
| Activity, each selected World | **unknown** | Runner/source structure predicts 24 existing model JVM launches versus 14 with healthy batching: main check plus six scoped progress checks. This is the planning handoff's code-derived prediction, not a process count. |
| Shared Model fixtures and mutants | **unknown** | Successful historical suite intervals and named agreement controls; no launch log keyed to emitted module/config hashes. |
| Kettle and other direct TLC controls | **unknown** | Historical backend checks; no complete per-fixture process inventory. |
| Java version / TLC help admission probes | **unknown** | Launch code admits pinned Java/TLC; those probes must be counted separately from model exploration. |

Total TLC wall time, per-fixture totals and fixed startup cost are also unknown. The spec's
earlier small-model startup observations motivate batching, but their raw probe is not a
complete baseline here. Average launch duration would include exploration and cannot
establish startup-only cost. A PATH-only Java wrapper would miss the absolute executable
returned by `mise which java`; the optional scratch observer intercepts that resolution.

H247's [workspace result record](../../.flow/artifacts/fn-247-one-language-for-the-framework-a-rust/task3/workspace-results.json)
reports 790 passed, 0 failed, 4 inherited ignored across 185 result summaries, including
doctests. The ignored entries were the isolated catalog-field rebuild control, a release-only
timing test, optional cargo-insta CLI test and an authoring documentation example. H237
reported 722 passed / 4 ignored under older source. These counts do not preserve the exact
current package/target/test/ignored identities or provide a one-to-one migration mapping.
The historical 121 UI-file claim and planning source search's 128 files are also not an
executed case inventory. Task .3/.4/.7 test consolidation and .5 UI conversion must retain
their exact identity/diagnostic mapping controls; the measurement waiver removes none of
those coverage obligations.

The fn-246 [required attempt1](../../.flow/artifacts/fn-246-realization-as-a-checked-refinement-a/task8/joined-program-correction/required-validation-attempt1.json)
took **415.12 s**, exit **2**, at source `f85d9399c48c09cc2dc8db08ee3cbe1fae071b9d`.
It failed at the Reset program assertion in workspace tests. Twelve completed stage timers
are partial evidence only; tests never completed and there is no passing gate total.
Attempt2 at source `8dab804688abb1c2d07a208fefd79799a641d06e`, Model tree
`0ccc7c4c5a0f15b7bbe3c32e4c18734b3a1404a6`, was still pending in the retained snapshot
`.flow/tmp/fn254-baseline/historical/fn246-attempt2-observed.json`. It is tracked separately
as fn-246 validation, not admitted as a fresh fn-254 baseline or represented as a pass here.

## Priorities and expected savings

The five largest separately identifiable H247 cost groups are Temporal fixtures **581.72 s**,
Apalache controls **396.35 s**, shared fixtures **213.52 s**, the annotation UI group
**126.89 s**, and wasm-guest **78.845 s**. The UI identity follows the retained case list;
its quiet-output executable heading is unavailable. These are nested in their owning
stages and must not be added to the 1707.092 s tests stage. H251's separate 162.67 s UI
observation corroborates the hotspot under different source/cache conditions.

The following is a priority ranking of **expected** savings, not measured improvements.
Overlap prevents summing them, and no numerical speedup is promised from these histories.

| Rank | Later task | Expected benefit and limit |
| --- | --- | --- |
| 1 | .6 real `model-quick` / unchanged-TLC cache | Largest inner-loop saving: avoid unrelated work and repeat backend runs after an activity edit; absolute saving unknown until R6. |
| 2 | .2 TLC batching / bounded property parallelism | Large required-suite opportunity in Temporal/shared fixtures; the 24→14 activity launch prediction must be verified without changing verdicts/traces. |
| 3 | .3 parallel execution across binaries | Potentially large full-suite saving while one long binary otherwise leaves cores idle; memory and JVM concurrency cap the gain. |
| 4 | .5 composition / trybuild trimming | Moderate-to-large: overlap independent stages and replace only UI cases whose diagnostic coverage survives. Apalache is already skipped in routine use, so merely making that explicit has no routine-mode saving. |
| 5 | .4 framework binary consolidation / test profile | Build/link saving plausible but unquantified: individual linker costs were never measured; current conductor runs already override debug to 0. |
| 6 | .7 Temporal binary consolidation | Similar unquantified build/link opportunity; waits for fn-246 closure and exact inventory mapping. |
| 7 | .10 framework entry point / root delegation | Enables framework-only checks and reduces unrelated iteration scope; no demonstrated same-coverage full-gate saving by itself. |
| 8 | .13 independent workspaces / owned tools | Enables that scope boundary; direct latency saving unknown and split dependency builds may initially cost more. |
| 9 | .9 stage/coupling inventory and design decision | No direct measured runtime saving; prerequisite for .13/.10. |
| 10 | .12 per-stage budgets | Prevents later cost drift; no immediate saving established, and added enforcement has a cost. |
| 11 | .11 standalone framework control | Coverage for independence; adds validation work rather than a demonstrated latency reduction. |
| 12 | .8 final measurements / documentation | Quantifies achieved savings and misses; adds measurement work. R6's final activity-edit/default/required targets remain to be tested. |

## Reproduction and final measurement boundary

`bash .flow/tmp/fn254-baseline/run.sh historical` reads retained receipts and preserves
the H251 logs, extracting 77 exact Cargo `Running`/harness-result pairs into
`historical/summary.json`. It starts no production command. File hashes, source references
and the pending attempt2 snapshot are retained beside it. Synthetic observer controls use
`python3 .flow/tmp/fn254-baseline/probe/controls.py`; the README records an initial failed
synthetic attempt and its correction. These controls validate only the observer mechanics.

The prepared `run.sh` host/build/inventory/default/serial modes remain optional future
support, not completed measurements. They require an explicit conductor lease for the main
checkout and target `/home/agent/.cache/codex-build/umpire-milestones-20261024/fn24612`;
they must not run from the worker worktree or use the inherited global target. Their
real-tool integration is unverified. Each future attempt needs new output names, retained
failures, monotonic elapsed/exit evidence, actual source/profile/backend identity and host
interference disclosure. Final R6 results must measure the actual edited activity path,
distinguish emission-preserving cache hits from changed-emission fresh TLC, and report
missed targets honestly. Historical data alone does not establish those outcomes.


## Dev-profile observations (fn-254.18, 2026-10-29)

Both workspace roots now set `profile.dev.debug = "line-tables-only"`, matching
`profile.test`. Parsed before/after profile tables establish that this is the sole
profile change: checked overflow, test settings and release/guest optimization,
stripping, LTO and panic settings are preserved. Both xtask aliases explicitly
select `--profile dev`. The native parity reports in
`umpire/dev/xtask/src/build_guests.rs` still select `--profile guest`, which inherits
release; native execution alone does not imply a dev build.

The committed [profile proof](../../.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task18/dev-profile-proof.json)
retains commands, effective rustc invocations, artifact hashes/sizes, parsed profiles,
source/config hashes, toolchain, cache conditions and full-log hashes. Raw logs and the
measurement script are retained under `.flow/tmp/fn25418/` in the conductor checkout.
Source base: `93a13e47a7097bcb7ae77b824e3d6490edda0320` plus the two manifest edits.
The host was Linux/aarch64, rustc 1.99.1 (`58cf8f9c3`), Cargo 1.99.1, 12 logical
processors and 19 GiB RAM. These runs provide no Darwin execution evidence.

From `model/umpire` and `model`, respectively, the focused commands were
`mise exec -- cargo build --locked --profile dev -p xtask --bin xtask -vv` and
`mise exec -- cargo build --locked --profile dev -p temporal-xtask --bin temporal-xtask -vv`.
All before/after builds exited 0. Profile/flag/wrapper overrides were unset (the exact
sanitization is in the proof), with a newly created task-owned target
`/home/agent/.cache/codex-build/fn25418-dev-debug`; tool and registry caches were inherited.
After preparing each profile, each matched observation touched only that binary's
`main.rs` timestamp, preserving source bytes, and rebuilt it with warm same-profile
dependencies and incremental state. Each verbose log contains one actual rustc binary
invocation with `--emit=dep-info,link`, rather than only Cargo Fresh output.

| Matched leaf compile + relink | Before wall (s) | After wall (s) | Before executable bytes | After executable bytes |
| --- | ---: | ---: | ---: | ---: |
| Framework xtask | 0.878 | 0.680 | 60,460,704 | 27,540,720 |
| Temporal xtask | 0.888 | 0.633 | 57,657,728 | 24,492,256 |

These are single observations, including mise/Cargo overhead, without a measured
compile/link split or isolated limiter. Cargo used default job concurrency with
`CARGO_BUILD_JOBS` unset. Independent frontend work and other sessions were active;
there is no quiet-host or repeated comparison. Preparation builds have different cache
histories and are recorded separately. **Speedup attribution is inconclusive**; these
numbers establish neither an end-to-end improvement nor a full-gate pass.

Effective rustc arguments changed from `-C debuginfo=2` to
`-C debuginfo=line-tables-only`. Cargo's JSON artifact profiles additionally report
`overflow_checks=true`, `debug_assertions=true`, `opt_level="0"` and line-table
debuginfo for both binaries. Fresh JSON metadata is settings evidence only.
Actual alias builds/runs passed with `cargo -vv xtask retired-names '^fn25418-proof$'`
(empty stdin, framework) and `cargo -vv xtask gate --list` (Temporal); the latter lists
stages without executing the gate. An earlier framework `gate --list` compiled but
failed at runtime because inherited mise task inventory required undefined
`UMPIRE_TEMPORAL_ROOT`; an omitted regex argument in an initial filtering smoke
returned 2. Both failed attempts remain in the proof; neither is counted as a pass.

The focused driver build also passed. Its executable shrank from 20,172,520 to
10,913,744 bytes; SHA-256 changed from
`4244471a914bcc44ab6f50874302312d8dd0c78276d9c483d21bdc8246cd4572` to
`0d2c5b48585eeac14e41ceb2dce7a0869058c50acdccdf8de439aa99c51bc5f2`.
This changes the guest-cache namespace and invalidates prior driver-bound freshness
evidence. **No guest reproduction was run here.** Task .8 must rebuild the driver
after integration, record its old/new identities, reproduce committed guest bytes in
the new namespace, and renew final combined/standalone checks. Flag inspection does
not renew that proof. No shared cache was cleaned; the task-owned target remains
available with its logs and identities preserved.

## Linux linker decision (fn-254.20, 2026-10-29)

Adopt mise-pinned mold 3.0.0 for the aarch64 and x86_64 Linux GNU target triples
in both workspaces. Only aarch64 execution was available. The Linux-only mise
entries lock both upstream release assets; Darwin and other target triples retain
the existing tools/linkers. The framework-local adapter resolves the qualified
`github:rui314/mold` install from its own tool root, refuses a missing executable,
and passes the existing cc driver `-B<pinned-install>/bin/ -fuse-ld=mold`.
It needs to exist before native Rust bootstrap and adds no global Rust flags.
The root mise pin provisions the framework tool for root-level `mise install`;
it must match the framework pin that the adapter resolves. Target `linker`
configuration remains effective for host
units when the guest driver supplies its own encoded WASI flags.

The [retained proof](../../.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task20/proof.json)
binds the matched measurement and input hashes. The same task20 artifact directory
retains raw logs, scripts, matched linker arguments and `measurement.json`;
`.flow/tmp/fn25420/` retains the local working copies. The host was
Linux/aarch64, rustc 1.99.1, GCC 15.2.0 and GNU ld 2.46, with 12 logical CPUs and
about 19 GiB RAM. Other lane builds paused for the final measurement; unrelated
host activity was not controlled. Sccache and Rust compiler wrappers were absent.

The workload was the real dev-profile guest-driver executable. A temporary linker
capture retained rustc's object inputs, then replayed the same cc arguments with
only the linker selection and output path changed. Every observation forced a
real link. Both arms used identical objects, rlibs, profile/features and toolchain;
one warmup per arm preceded six balanced observations per arm. The mold arm
includes the adapter's mise lookup cost. Median link wall time was **0.230219 s
for GNU ld and 0.081408 s for mold** (about 65% lower for this link). All 14 links
succeeded. ELF `.comment` inspection confirms mold 3.0.0 only in the mold arm.
This establishes a focused link improvement, with no full-build or gate speedup
claim. The preparation build and failed experimental attempts are retained
separately and excluded from that comparison.

Actual framework bootstrap, Temporal tool build and a copied framework-local
configuration outside the parent repository all linked successfully. The complete
guest-driver controls passed (5 unit and 34 integration tests), including target
scope, qualified local mise resolution, spaces/non-UTF-8 arguments, automatic
installation refusal and missing-tool/missing-executable refusal. Scoped clippy
and formatting passed.

A stock WASI build with build-std kept the guest's existing panic/unwind,
optimization, stripping, remaps and rust-lld path. Actual host proc-macro shared
libraries and build-script ELF binaries from that build contain mold's comment.
A baseline host-linker build (`CARGO_TARGET_AARCH64_UNKNOWN_LINUX_GNU_LINKER=cc`)
and a restored mold build used the same driver/lock/compiler namespace; the host
units rebuilt, and all 20 inspected host outputs changed back to mold. Their
portable sidecar identities and host-variant records stayed equal. The stock
WASM bytes stayed equal at 198,992 bytes, SHA-256
`4979d134ba8ae6afc4ba79dadc046251afbbf4ca34397ed61af184205a371222`.
An earlier stock build used a different namespace because the concurrent countdown
lane added a lockfile entry; it is retained but is not the matched namespace proof.

The native current-contract report helper ran with `--profile guest`; a freshly
built control WASM matched its published responses through the CGO-disabled Go
host's current-contract parity test, and stock guest export admission passed.
These are representative affected controls. Final current-input standalone/full
gates, committed Temporal guest freshness and combined timings remain task .8.
No Darwin or x86_64 execution proof is claimed. Linker/pin upgrades require fresh
host-build and guest reproduction evidence; historical linked artifacts do not
certify a changed linker.

The root Make entry explicitly loads `model/.cargo/config.toml` through
`RUST_MODEL_CARGO`. `--manifest-path` alone does not load a workspace's Cargo
configuration. Keeping the caller directory preserves relative Cargo home/target
paths while selecting the same linker as nested workspace commands. The
[Make entry proof](../../.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task20/make-entry-proof.json)
records an actual `make model-quick` bridge/bootstrap/selection cycle around a
nested Temporal tool build. The observer adds verbose Cargo logging and
`quick --select-only`, so this checks compilation/configuration consistency rather
than running the selected validation gates. Before the fix, all three steps
recompiled 43 Rust units for changed config settings; afterward, all three warm
steps compiled zero Rust units and reported no changed config settings. The warm
Make executable contains mold's ELF comment. The permanent Temporal Make control
also guards both supported target entries and caller-relative path preservation.
