# Task12 TLC count source research

Source HEAD: `9babff6c815beb7cff92d7001f647ddca19663bf`. Read-only research while task15 proof runs; task12 still waits task15 closure. No implementation, runtime, review verdict or budget admission.

The current unknown totals are correct under the current evidence contract. A `tlc-finish ... 0` says the captured output did not confirm TLC startup; it does not prove zero launches. A trusted exec-outcome channel can prove some non-starts, but cannot by itself classify every successful synthetic protocol process as real-TLC zero.

## Current producer and collector

`work::spawned` recognizes any command argument equal to `tlc2.TLC` after the owned supervisor has spawned. It records a PID/sequence attempt with configuration hash, `probe`, or `-`. `finished` recognizes two TLC startup strings and otherwise records0. See [model/umpire/backends/process/src/work.rs:81](../../model/umpire/backends/process/src/work.rs#L81) and [model/umpire/backends/process/src/work.rs:103](../../model/umpire/backends/process/src/work.rs#L103).

`run_process_cancelled_with_policy` owns the capture and cleanup. Refusal/cancellation/output limits can end before a banner is captured. NonUTF8 output currently returns before finish publication; that remains a missing-completion error rather than a numeric count. See [model/umpire/backends/process/src/support.rs:451](../../model/umpire/backends/process/src/support.rs#L451).

`tlc_counts` requires paired observations and makes launch totals numeric only when all finishes confirm startup. It includes admission `-h` launches, excludes probes from unique source/configuration hashes, and refuses configuration certainty for a started `-` identity. Current successful receipt validation permits nullable TLC modalities; the historical budget candidate demands all6 actual counters. See [model/umpire/dev/xtask/src/gate/work_counts.rs:324](../../model/umpire/dev/xtask/src/gate/work_counts.rs#L324) and [model/umpire/dev/xtask/src/gate/work_counts.rs:137](../../model/umpire/dev/xtask/src/gate/work_counts.rs#L137).

## Existing evidence

The retained [mixed standalone summary](../../.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task17/resumed/standalone-renewal-8350ba8f/summary.json) reports836 attempts and836 finishes:752 startup-confirmed and84 unconfirmed. Its limitation explicitly leaves aggregate launch/configuration totals null. The [narrow real-TLC producer proof](../../.flow/artifacts/fn-254-fast-validation-gates-and-a-real-inner/task17/resumed/work-count-producers/tlc-counts.json) records6 launches including1 admission probe and3 unique model configurations. Neither is current task12 acceptance.

## Exact source contributors

All five `runner::batch_tests` controls below use `fake_tlc`, a Bash executable that writes selected protocol/dump bytes and emits no TLC startup banner. They retain important refusal, recovery, output budget and concurrency coverage:

- `runner::batch_tests::batch_protocol_requires_complete_selected_evidence`
- `runner::batch_tests::batch_resource_refusal_does_not_launch_singleton_recovery`
- `runner::batch_tests::singleton_recovery_measures_worker_peak_and_owns_each_scratch`
- `runner::batch_tests::singleton_output_budget_precedes_later_evaluation_and_stops_new_waves`
- `runner::batch_tests::singleton_evaluation_stops_recovery_after_current_wave`

Five cache-control families use `fake_backend` and the synthetic `cache_child` branch:

- `runner::cache_tests::populated_cache_hits_and_authoritative_disable_runs_fresh`
- `runner::cache_tests::verified_configuration_publishes_despite_refuted_sibling`
- `runner::cache_tests::identity_refreshes_between_module_calls_and_is_shared_between_configurations`
- `runner::cache_tests::cache_root_refusal_reports_its_cause_without_following_symlinks`
- `runner::cache_tests::failures_refutations_and_incomplete_modules_never_publish`

These constructors bypass the admitted real jar/Java path. `Tlc::execute` still adds `tlc2.TLC`, so their scoped processes are recorded as attempts and finish unconfirmed even when the protocol test succeeds. The real-Kettle branch of `cache_child` calls `Tlc::discover` and must keep real checker attribution. See [model/umpire/backends/tlc/src/runner.rs:139](../../model/umpire/backends/tlc/src/runner.rs#L139) and [model/umpire/backends/tlc/src/runner.rs:561](../../model/umpire/backends/tlc/src/runner.rs#L561).

The producer events have no per-test identity. These are source-proven contributing families, not a fresh one-for-one reconstruction of historical84. Cache feature guards can suppress some families under strategy-equivalence; preserve the exact feature/input pin when measuring.

Missing/corrupt jar and escaping-cache controls refuse before TLC; generic sleep/printf timeout/output/dump controls lack `tlc2.TLC`. They do not account for these unconfirmed TLC attempts. Invalid cached-batch admission refuses before its missing-backend sentinel can launch.

## Three required terminal states

| State | Positive evidence | Budget consequence |
| --- | --- | --- |
| confirmed-started | Admitted real checker startup evidence associated with the owned command; preserve recognized model and -h version banners. | +1 launch; hash config contributes unique source+cfg identity; probe excluded from config count; started configuration- remains unknown configuration count. |
| proven-not-started | Complete trusted supervisor evidence for failure before successful target exec, or a reviewed closed applicability/provenance boundary for a synthetic process (not banner absence). | +0 actual checker launches only with positive proof; retain attempt and reason in evidence. Never infer this state from finish0, status1/127, refusal, cancellation, empty output or generic script success. |
| unknown | No conclusive startup or non-start proof, failed/partial exec evidence, truncated/nonUTF8 output, banner not captured before kill, or unclassified synthetic command. | Aggregate remains null/refuses launch budget. Preserve all paired events and cause; no numeric substitution. |

A refusal, status1/127, empty/missing banner, successful generic shell exec or cancellation is insufficient proof of a non-start. An actually started admitted checker remains a real launch even when a resource/evaluation negative control returns refusal.

## Two potential seams; implementation choice remains open

The strongest general execution-state seam is the existing owned `Prepared`/`ChildContext`/`target_main` boundary ([model/umpire/backends/process/src/syscall.rs:797](../../model/umpire/backends/process/src/syscall.rs#L797)). All candidate execve failures reach `child_error`, but it currently emits a human stderr line and exit1, indistinguishable from executed-program failure. A bounded private structured channel tied to the owned attempt can report trusted pre-exec failure. It must preserve async-safe post-fork code, guardian ownership, descriptor closing, lifecycle/cancellation/completion and leases. EOF alone after a killed/partial writer is not proof. This extends sensitive existing code; it must not add another scheduler.

Synthetic provenance or a fixture-owned observation scope is a complementary design possibility, not an accepted shortcut. It must attest actual pinned executable/script identity and complete owned execution, retain every protocol negative and its unknown/corruption assertions in a linked child scope, and leave parent actual Rust inventory/execution counts intact. A fixture env flag, name filter, missing banner, clearing observation scope or fake startup banner is not such proof. A generic script could exec Java in-place, so script success alone cannot establish real-TLC zero. The real cache-child branch must never inherit an unconditional synthetic exemption.

One possible attribution boundary is to retain admitted `PinnedJar`/Java provenance with real TLC invocations while private fixture construction uses an explicitly evidenced protocol-process authority. The exact type/scope API and whether that proves inapplicability need task12 design and controls. Synthetic unknowns must remain visible in their own complete receipts; they cannot be erased to make real totals numeric.

## Required controls before budget admission

- Preserve existing unknown-control: attempt alone ->missing completion refusal; finish0 ->tlc_launches None.
- Structured failure-before-exec yields proven non-start only after complete identity-matched terminal evidence; a real executed program exit1/127 is distinct.
- Real admitted JVM exits before TLC banner; cancel/timeout before banner; output limit/nonUTF8/drop: unknown unless positive non-start proof.
- Fake protocol success/failure/partial/evaluation/output limit remains executed synthetic work with its full original assertions and owned cleanup. No fake banner added.
- Forged synthetic provenance, changed script/executable hash, misdirected real cache-child branch, corrupted/missing linked fixture receipt: fail closed rather than lower real counts.
- A mixture of known real, proven non-start and unknown cannot become known aggregate; all-known exact sum and config dedup checked including admission probes.
- Retry source/tool/feature/fixture provenance remains identical and records both attempts; no count-only retry after unknown/count refusal.
- Parent Rust counts retain every executed control; child synthetic counts never double-charge inclusive parents and never disappear behind scope reset.

The existing unknown unit control must remain: an attempt with no completion refuses, and a finish0 keeps `tlc_launches=None`. The retained budget work at `4835a4734a` deliberately refuses missing counters/uncalibrated tables; `f7ad9c4e58` wires opt-in retry around the existing scheduler. Those are source donors, not current validated integrations. Keep their count-refusal and invocation/child authority rules; adapt selectively to final task15 source.

No system Model, test selection, DAG, build target or Flow state changed. The companion JSON contains exact symbol anchors/source hashes, historical evidence hashes, ten contributing test identities, non-contributing controls, design tradeoffs and unperformed validation requirements.
