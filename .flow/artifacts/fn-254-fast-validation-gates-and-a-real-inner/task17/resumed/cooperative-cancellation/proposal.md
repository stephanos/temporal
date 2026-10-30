# B proposal: cooperative orchestration cancellation

**Design only. Do not implement or claim cleanup acceptance from this document.** A is committed as e8a2ae2e28: default Checker guardians retain verified inherited custody, while Java receives neither FD9 nor the locator. A protects host JVM exclusion during guardian cleanup. It does not establish that all cleanup precedes CLI return, especially when an adopted guardian is stopped or when Darwin cannot reap an escaped descendant.

## Recommendation

Keep mise as the sole DAG scheduler and keep Checker shutdown/limits/status semantics unchanged. Add cooperative cancellation only to the native **Orchestration** lifetime policy. Its first cancellation wave should give the actual pinned controllers an opportunity to forward shutdown and reap their children; native cleanup must not immediately kill those controllers. The deadline controls escalation, never evidence of completion.

For admitted mise orchestration set `MISE_TASK_PGID_MANAGED=1` in the managed execution environment. This is a group-management hint, not authority, a JVM allocation, or a cleanup receipt. Continue validating FD9 independently. Preserve its propagation through source-only `env_clear` commands explicitly. Do not copy the marker into Checker/Java targets as an accidental new control interface.

A direct mise target already starts a session under this native supervisor. The marker still matters to nested mise instances: they must not create additional groups outside their current managed domain or duplicate a SIGINT already delivered to their group. Native wrappers may themselves establish separate owned groups, so the marker does **not** imply that the whole validation tree shares one PGID.

## Pinned primary evidence

The assembly research agent supplied these exact pinned references; its retained earlier report is `task17/resumed/process-supervision-design.json`. Later ordinary-pipeline signal findings were chat-only and are recorded here without rerunning probes.

- [mise2026.10.11 command ownership](https://github.com/jdx/mise/blob/v2026.10.11/crates/mise-util/src/cmd.rs): `should_use_pgroup`, `prepare_execute_child`, `sigint_reached_own_pgrp`, `kill_all`, `terminate_all`. Session-leading mise skips child-group creation. `MISE_TASK_PGID_MANAGED=1` suppresses nested group creation and supports SIGINT deduplication for a shared group. SIGTERM forwarding does not have that deduplication.
- [mise task-mode interrupt handling](https://github.com/jdx/mise/blob/v2026.10.11/src/ui/ctrlc.rs) and [run integration](https://github.com/jdx/mise/blob/v2026.10.11/src/cli/run.rs): first SIGINT starts cooperative task shutdown; another interrupt escalates. The run path disables immediate exit-on-interrupt and tracks cancellation while scheduling. Ordinary fail-fast separately sends SIGTERM to siblings. That existing fail-fast wave can race native cancellation; it must be exercised rather than assumed harmless.
- [nextest0.9.149 Unix runner](https://github.com/nextest-rs/nextest/blob/cargo-nextest-0.9.149/nextest-runner/src/runner/unix.rs): tests use separate process groups; termination signals those groups and escalates after grace expiry or another shutdown signal.
- [nextest executor](https://github.com/nextest-rs/nextest/blob/cargo-nextest-0.9.149/nextest-runner/src/runner/executor.rs): setup scripts and normal non-debugger tests receive groups, and execution waits for the direct children. [Default grace](https://github.com/nextest-rs/nextest/blob/cargo-nextest-0.9.149/nextest-runner/src/config/elements/slow_timeout.rs), `default_grace_period`, is10seconds.
- [nextest double-spawn](https://github.com/nextest-rs/nextest/blob/cargo-nextest-0.9.149/cargo-nextest/src/double_spawn.rs) execs the test; it is not a surviving parent-death guardian. No PDEATHSIG facility was identified in the inspected pinned sources.
- [Apple group signalling](https://developer.apple.com/library/archive/documentation/System/Conceptual/ManPages_iPhoneOS/man2/kill.2.html) remains group-limited. [XNU](https://github.com/apple-oss-distributions/xnu/blob/main/bsd/kern/kern_event.c) rejects NOTE_TRACK-family flags; the existing report explains why NOTE_FORK observation alone does not provide descendant custody.

Both shipping nextest profiles currently inherit a30second slow-test reporting period without a repository grace override. That is **not** a30second cancellation grace and does not change the pinned10second default. A native backend should receive an admitted effective cancellation budget from its caller rather than parse nextest profiles or guess that five seconds of SIGTERM is sufficient. Native/other inherited profiles or later overrides can require a larger budget. A default10second observation is not a universal hard cap.

## Minimal native state and barriers

Suggested states belong inside the existing unsafe syscall owner, with parent-prepared data and allocation-free child operations. They own lifetime only, not stage ordering.

| State | Action | Completion condition |
|---|---|---|
| Running | Observe owner pipe, stop signals and direct target status as now. | Natural target completion or cancellation enters draining; it does not prove the family is gone. |
| Cooperative drain | Send one SIGINT wave to the controlled orchestration group. Keep the supervisor, FD9 and relevant controllers alive while they forward to their groups and wait. | Actual owned-child/acknowledged-controller completion permits progress; direct mise/CLI exit alone does not. |
| Escalation | At the admitted deadline or a separately defined repeated-interrupt request, perform owned hard cleanup. | Deadline expiry is a reason to escalate/refuse, never a successful cleanup acknowledgement. |
| Final reap | Linux retains its existing subreaper kill/reap loop to ECHILD, including other sessions/groups. Preserve A's nested Checker custody until Java is killed/reaped. | Actual completion barrier before successful result/admission release. Darwin cannot substitute direct-child ECHILD for adoption ownership. |
| Complete or refused | Preserve original named failure/signal reason, simultaneous failures and truthful partial/cancelled receipts. | Never convert incomplete/unsupported cleanup into success or executed counts. |

Critical ordering: if direct mise exits during cooperative drain, do **not** fall through to the current unconditional root-group SIGKILL. A nextest controller may still be in that group forwarding to separate test/setup groups. Keep draining even after recording/reaping the direct target. Killing that controller early can orphan ordinary tests without any adversarial setsid.

A group-existence check may help avoid killing a still-draining controller, but it is not descendant reaping. EPERM must not mean an empty group. Nor does an empty original group prove that nextest's separate groups, detached descendants or a paused Checker guardian are gone. On Linux the actual subreaper remains the final owner. On Darwin the stronger family completion contract still needs a concrete supported mechanism or explicit controller/guardian acknowledgement that covers the relevant processes; this proposal does not invent that missing mechanism.

Keep the existing failure cause separate from the soft signal used for cleanup. A leaf's37 must remain37 rather than becoming130/137 because siblings were interrupted. A caller's SIGTERM reason may remain143 even if orchestration's cooperative wave uses SIGINT; that conversion and repeated-signal behavior need explicit tests. Checker first-stop precedence and all existing hard bounds stay unchanged.

## Interfaces and source ownership to settle before implementation

- Prefer an explicit parent-prepared orchestration shutdown policy/budget over changing Checker defaults or baking nextest knowledge into the generic backend. The effective grace/drain allocation is part of the admitted caller context. Unknown overrides cannot be turned into an unmeasured guarantee.
- A cooperative request method on `OwnedProcess` could make cancellation intent explicit and let the graph continue harvesting simultaneous leaf receipts. Alternatively, keep Drop as the liveness-EOF trigger and implement the drain in the existing guardian. Either option must wait for actual native completion; do not implement both as competing shutdown authorities.
- If a monotonic deadline is needed after fork, resolve the already-pinned libc clock call in the parent and keep it inside the established syscall/calls boundary. Do not call allocation-capable Rust helpers or add another unsafe module. Time is for the deadline only.
- Graph/CLI admission owns marker propagation and the one cancellation initiation per owned invocation. It continues to consume mise's DAG and actual invocation-local stage receipts. Process ownership must not recreate dependency scheduling.
- Preserve clean `env_clear` semantics, separate raw stdout/stderr, source provenance, full-gate cache bypass, disabled auto-install, target/Cargo-home normalization and all existing selected test/ignored/doctest identities. Do not add flags that make a cancelled suite count as complete.

## Required proof before adopting B

Use the actual pinned mise and nextest, with controlled native cases in existing test binaries. Stub executors alone cannot prove nextest's group and grace behavior. Root must separately admit the small phase; nothing below has been executed for B.

1. Real mise ordinary failure37 with a sibling nextest controller running ordinary test/setup groups. Observe the initial cancellation signals, nextest forwarding/reaping, and all owned PIDs gone before success/refusal completion. Dependent tasks never start. Preserve simultaneous named failures.
2. Nested mise with and without the managed marker: demonstrate the actual group topology and SIGINT delivery count. The accepted path must avoid an accidental second shutdown wave. Exercise automatic mise SIGTERM fail-fast racing native SIGINT; source reasoning alone does not establish the timing outcome.
3. Direct mise exits early while nextest continues draining. Prove the native guardian remains alive and keeps FD9, does not root-group SIGKILL prematurely, and waits for a genuine completion barrier. Repeat with natural direct-target failure and zero exit.
4. Owner EOF/SIGKILL, user SIGINT, user SIGTERM and deliberate repeated interrupt. Preserve statuses and finite escalation without claiming that elapsed grace proves cleanup. Checker timeout/output cancellation and Darwin prevention controls must remain unchanged.
5. Pause a nested Checker guardian after its Rust test is gone. A second independent gate/quick invocation must stay excluded because of A's custody. The first CLI must not report cleanup complete while that guardian/Java remains. Resume it and require actual termination/reaping/acknowledgement. This distinguishes exclusion from cleanup-before-return.
6. Renew separate-session/group escape controls on Linux. Explicitly retain Darwin escaped/adopted cases as unsupported/unproven until a real supported owner exists; marker/grace cannot satisfy those cases. Do not skip original macOS checks or declare task17 complete from Linux proof.
7. Freeze the joined implementation before rerunning public independent gate/quick admission, the correctly selected real six-consumer Kettle and five-consumer Temporal quick workloads, and the production standalone control. Preserve complete semantics and timing provenance. No new baseline campaign and no claimed speedup from cancellation design.

## Decision boundary

A is a completed narrow source/proof unit. B remains a proposal. The next useful step is a separately scoped implementation and actual ordinary-pipeline controller proof, with a stated Darwin completion boundary. Cooperative SIGINT is a justified improvement over immediate controller SIGKILL; it is not a replacement for kernel/lifecycle ownership or proof of portable task acceptance.
