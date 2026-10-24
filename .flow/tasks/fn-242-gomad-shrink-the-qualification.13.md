---
satisfies: [R2]
---
# fn-242-gomad-shrink-the-qualification.13 Real process custody, accounting, busy workers and cleanup

**Touches:** tools/gomad/rust/gomad/src/cycle/process/**, tools/gomad/rust/gomad/Cargo.toml

## Description
real child custody, accounting, gate and cleanup

Depends on: cycle-foundation. Original .2-owned IDs: 27, all pending. Own the real implementation of launch.py even though .10 owns the proposed endpoint routing record; .10 calls this single implementation.

Description: implement actual bounded stage/leaf subprocess creation with pre-exec registration/admission gate, environment/cwd/argv and PID identity, exact wait4 child accounting, exit/signal retention, command/process observation and fail-closed cleanup. Reuse existing low-level primitives without widening measurement internals. No long wait while holding fork_order spawn lock.


Proof ownership: `.flow/artifacts/fn242-cycle-proof-ownership.json`, group `cycle-process`; inventory source binding and exact ID list remain authoritative. Keep original .2 aggregate acceptance; this task produces evidence, not qualification credit. Reuse existing validation and fold same-boundary bookkeeping only with a named surviving guarantee and executed catching/control proof.

## Conductor architecture correction — supervisor failure

The first review found that a killed per-command supervisor can abandon its leaf and escaped descendants. On this host, PID-namespace-only creation returns EPERM; a user-namespace wrapper changes UID/GID/group semantics. Neither an extra ordinary guardian nor reporting cleanup inconclusive discharges custody.

Implement one explicit exclusive cycle-process owner, claimed once before children or threads. Verify actual owner PID, empty children across every procfs thread, no existing subreaper claim, then enable and verify child-subreaper status. A private PID-bound token is mandatory for every Runner; generic construction must not silently alter arbitrary caller state. Keep the role until process exit. Mixed ownership refuses without killing existing children.

Every cycle spawn, registration and normal supervisor reap participates in one domain protocol. Any supervisor EOF/death/malformed report aborts admission and all active commands. One synchronized drain kills and reaps all registered and adopted domain children to kernel ECHILD before an error returns. Do not invent per-command attribution after ancestry loss. Preserve exact normal leaf wait4 once; missing resource reports remain sticky failures after successful cleanup.

Use a standalone harness=false fixture with the actual strict production initializer before threads/children. The additive Cargo.toml test target is included in Touches. No pre-main ELF hook or test-only relaxed claim bypass. Prove concurrent commands, mixed-owner refusal, exact registered-supervisor SIGKILL, escaped descendants, whole-domain failure and absence of all attributable PIDs. Original catching/control ownership remains binding.

Actual production startup integration is B15, not supplied by a fixture. See `.flow/artifacts/fn242-process-custody-decision.md`; this corrects the earlier isolation design without waiving original R2 or spending a qualification attempt.

## Acceptance
- [ ] Successful, nonzero, signaled, spawn-refused, interrupted and timed-out commands execute actual bounded processes and retain numeric exits/signals, actual parent/child IDs and monotonic spawn/reap boundaries; wait4 user/system/RSS bytes refer to exactly the launched child once. No elapsed-time CPU approximation or RUSAGE_CHILDREN subtraction.
- [ ] Gate release only follows durable registration; refused or interrupted admission cannot execute the leaf. Protect exact/private observation context and explicit environment typing; environment sanitization preserves original contract and excludes task-host GIT_*/SANDBOX_START_DIR/GOROOT for build children.
- [ ] Reap and verify entire owned process groups and sessions, including leader-exits-with-descendant, changed descendant group, malformed/unobservable procfs and interrupted cleanup. Retain original failure precedence; cleanup ambiguity cannot become success.
- [ ] Busy-boundaries/busy-grouped launch twelve real busy worker processes, retain custody and reap every worker on success/failure/interruption before return. Fixtures use bounded handshakes, not full qualification.
- [ ] Prove registry freshness, duplicate PID/registration refusal, accepted status semantics and every assigned predicate with valid controls. No attributable live child may survive tests or handoff.

## Done summary

Implemented actual bounded command launch, durable pre-exec registration, exact leaf wait4 accounting, honest parent identities, private observations, twelve real busy workers and complete descendant cleanup. An explicit CycleOwner claims the dedicated cycle process before children or threads; every Runner requires its PID-bound token. Supervisor loss aborts and drains every domain child/adoptee before returning a sticky failure. Timeout validation retains original 5400/7200 bounds; replaced stream/registration FIFOs refuse without blocking.

Verified all 27 assigned original checks against final source. Integrated gates passed 24 standalone owner cases, 19 library cycle tests, strict Gomad library Clippy, formatting and diff checks. Both supervisor-death controls left no survivors before return; independent integrated audit found all 44 logged PIDs absent. Worker evidence audited 771 observed PIDs absent and 19 exact obsolete fixture paths removed. Historical source-mutant samples and initial failing/compile logs remain honestly bound to their original sources. Current proof: .flow/artifacts/fn242-cycle-process-review-fix-proof.json; raw evidence and integrated logs: /home/agent/wt/evidence/fn242-cycle-process/review-fix.

Same-receipt independent gpt-6.1-sol high re-review returned SHIP, all three prior findings fixed and no new blockers; writer/reviewer are from the same family. Receipt retained in root-completion/review-receipt.json. Tier: session (jev-unavailable(no_key)); project pins retained.

This completes the assigned process implementation. Actual CycleOwner CLI startup wiring remains required in B15; full qualification remains fn-246 after all fn-242 and fn-240 work. No full attempt, global enrollment or reservation ran. Policy/activity accounting remains 0/2; the absent fixed journal binding is not a verified durable counter.

## Evidence
- Commits: fb706498e0, 2280efc245, 05e64987b0, 422f9360eb, 6a0f56791e
- Tests: cargo +1.99.1 --config profile.release.package.gomad.opt-level=0 test --release --locked --manifest-path tools/gomad/rust/Cargo.toml -p gomad --test cycle_process_owner (exit0; /home/agent/wt/evidence/fn242-cycle-process/review-fix/root-integration/owner.log), cargo +1.99.1 --config profile.release.package.gomad.opt-level=0 test --release --locked --manifest-path tools/gomad/rust/Cargo.toml -p gomad --lib cycle:: -- --test-threads=1 --nocapture (exit0; /home/agent/wt/evidence/fn242-cycle-process/review-fix/root-integration/cycle.log), cargo +1.99.1 clippy --release --locked --manifest-path tools/gomad/rust/Cargo.toml -p gomad --lib --no-deps -- -D warnings (exit0; /home/agent/wt/evidence/fn242-cycle-process/review-fix/root-integration/clippy.log), cargo +1.99.1 fmt --manifest-path tools/gomad/rust/Cargo.toml --all -- --check (exit0; /home/agent/wt/evidence/fn242-cycle-process/review-fix/root-integration/fmt.log)
- PRs: