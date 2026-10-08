# Gomad v3: Milestones to a Deterministic Temporal Functional Test

## Purpose

This document is the delivery ladder for running Temporal's functional tests (`./tests`, built on
the `testcore` one-box cluster with in-memory SQLite and loopback gRPC) under Gomad v3 so that the
same seed produces the same run and a retained artifact replays byte-exactly.
[GOMAD_NEXT.md](.plans/GOMAD_NEXT.md) remains the capability roadmap across all four tracks.

## Work tracking

Flow specs and tasks own scope, acceptance criteria (R-IDs), blockers, and evidence.
Remaining native linux/amd64 qualification and Linux CI work stay deferred under
[fn-128](.flow/specs/fn-128-gomad-deferred-linux-qualification-and.md). The owner-approved
Darwin deferral transfers remaining native darwin/arm64 qualification to
[fn-149](.flow/specs/fn-149-gomad-deferred-darwin-qualification.md), as mapped in the
[native transfer manifest](.flow/artifacts/native-scope-transfer-2026-10-07.md).
Missing transferred native evidence does not block source-spec completion. Implementation,
ordinary host-source coverage, lint, preservation, both-source-set static checks, generated
validation, source review and other independent acceptance remain required. Native full-host
execution stays with the native owner; portable coverage does not substitute for that gate.
Both platforms remain unverified for the current candidate until their owners retain native proof.
No PR, push or CI action is authorized by the deferral or its revival.
List every task of each open spec below, including completed tasks; remove a spec's
entire section only when the spec is complete. Keep descriptions to one line and
spec sections to task tables—no progress prose. Refresh statuses from `flowctl list --json`;
a committed source candidate is not completed acceptance.

Status: ✅ Done · 🚧 In progress · ⛔ Blocked · ⬜ Todo.

Work a spec with `/flow-next:work <spec>`; list ready tasks with `flowctl ready`.
Record a deferred task's revival trigger in its owning Flow task before implementation.
Completed specs and their evidence remain in `.flow/` and Git history.

## Immediate delivery order

1. Reconcile the combined D26/fn-110 source candidate, fn-114 tasks 13/14, fn-112 task 5 and D27 against their retained source checks, preservation and source reviews. Native runtime/clock proof belongs to deferred fn-149.1 and fn-128.1/.4/.7.
2. Finish retained source acceptance for merged fn-112 tasks 16/9, fn-113 tasks 1-4 and fn-109 tasks 2-6 against the integrated candidate. Keep lint, both-source-set static checks, generated validation, first-baseline and preservation requirements open wherever unproved.
3. Continue fn-109 tasks 7-12, fn-110 tasks 3-4, then fn-109 tasks 13-21 and fn-110 task 5 in their source delivery order, including the admitted correction owners. Predecessor source integration/review and retained source acceptance remain required. Missing native qualification now owned by fn-149 or fn-128 cannot block source admission or completion; unproved source requirements still do.
4. Resume fn-105 D8-D10's shared source work when the real checkout is available. Native Darwin consumer analyses/packs/replay/guidance belong to fn-149.3; Linux execution belongs to fn-128.6. Neither native owner removes the actual-checkout/source-review prerequisite.
5. Keep fn-149 deferred until an explicit Darwin qualification request and native darwin/arm64 execution are available. Then establish its pinned candidate/runtime, retain inherited integration/model/pack and downstream evidence, and reconcile an actual scheduled/dispatched soak plus final matrix. Publication and CI actions need separate authority.
6. Keep fn-128 and its Linux CI work deferred until an explicit Linux qualification request and native linux/amd64 execution are available. Its eventual sequence remains baseline, causal D12 strict-replay fix, D21-triggered D11 audit, native model/consumer/soak evidence and final Linux matrix. No PR or CI run is part of the current work.

## Verification instructions for agents

Apply this workflow to Gomad milestone work. Optimize elapsed time by choosing checks that
cover the changed behavior and reusing valid results from the same source revision.

Be wary of overthinking: follow the existing acceptance criteria, choose a grounded
recommendation, and move to implementation. Revisit a decision only when new evidence warrants it.

1. **Check cheap boundaries first.** Run focused regressions during implementation. After
   import or package-boundary changes, run `TestPackageArchitecture` in the nested module's
   root package. Run `make -C tools/gomad3 validate` before broad tests when changing files
   that may affect generated code, protocol identities, or toolchain inputs. Check the
   generator's input list before editing a shared host/runtime file.
2. **Run one full host gate per frozen batch.** Use
   `GOFLAGS='-tags=test_dep -count=1' make -C tools/gomad3 test-host` with the pinned native Go
   on `PATH` and the documented toolchain setup. This includes Runner and CLI packages;
   count it as covering their overlapping task test commands instead of running both broad
   suites. Retain focused regression evidence and identify the packages covered. Runtime,
   overlay, integration, race, and platform-specific requirements still need their own gates.
3. **Scope reruns to the new change.** After a small review fix, run the affected regression,
   package, and relevant boundary checks. Repeat the full gate when a shared dependency,
   runtime or protocol change, broader regression, or unresolved coverage concern warrants
   it; record the reason. Documentation-only edits need document/diff checks. Keep source
   stable during each test command and identify the revision each result covers.
4. **Keep handoffs small.** Retain the meaningful failing regression, final passing commands,
   exit codes, elapsed times, source revision, and review verdict in one task handover.
   Reference existing evidence instead of copying it into successive manifests and reports.
   Keep generated binaries, bulk traces, and scratch snapshots local unless delivery requires
   them. Use one independent review for a completed batch; re-review actionable fixes.
5. **Commit each task separately.** After its source checks and review pass, commit the task's
   implementation, tests, documentation, and Flow records together before starting the next
   task. This supersedes older
   task instructions reserving commits for the user. If a required gate still owned by the task
   is unavailable, commit verified progress and keep its acceptance open. Record transferred
   native gates under fn-128/fn-149 and link the owning task from the source task. Complete the source
   task only after all requirements it still owns pass. Preserve unrelated
   changes; push only when authorized. Broaden testing only for a concrete remaining risk,
   and retry an unchanged environment failure only when its cause or relevant inputs change.

### Orchestration

- **Root ownership.** The root/main agent owns task selection, dependencies, scope, dispatch,
  integration, independent review coordination, Flow lifecycle, and completion. It remains
  accountable for verifying every owned acceptance requirement against the integrated candidate.
  The root may edit orchestration documentation and lifecycle metadata; it routes task code
  and test fixes back to workers.
- **Worker ownership.** Use a fresh task-worker subagent for each task's investigation,
  implementation, tests, evidence, and review fixes. Independent reviewers assess the integrated
  changes and supply review verdicts; workers cannot certify their own acceptance.
- **Durable handovers.** Give each task a uniquely named, small handover using the evidence
  fields above. Bind the actual source and tool inputs, commands and results, and remaining
  requirements. Reference existing evidence and the independent review verdict.
- **Parallel boundaries.** Implement tasks in parallel only when their dependencies are
  independent and their scopes disjoint, using isolated worktrees. Serialize overlapping work
  and shared Go, build, lint, and generator gates. Keep the checked source frozen throughout
  each gate and bind its results to that candidate before integration or completion.
- **Root context.** Retain decisions, dependencies, and handover pointers in the root context.
  Resume workers from authoritative Flow records, handovers, and referenced evidence. Keep
  full task histories with their workers and artifacts to reduce root context growth.

Apply this division of work within the immediate delivery order, Flow rules, verification
requirements, and native deferrals above. It grants no additional native qualification,
CI, PR, or push authority.

## Constraints

- **No policy widening.** Gomad never grants `syscall`, `os/exec`, `os/signal`, or
  `golang.org/x/sys` generically. Every exception is an exact compatibility pack bound to a
  module version, go.sum hash, per-file SHA-256, owner, and workload, reviewed under the
  existing `discover`, `review`, `generate --approve-review`, `check`, `qualify` flow.
- **No source translation and no test rewriting.** Determinism comes from the patched
  toolchain and the reviewed boundary. A Temporal test that needs a Gomad-specific overlay of
  its own source, as gomad1 required, is a blocker to record, never a fix to ship.
- **Fail-closed stays.** An unmodeled boundary operation terminates the process. Work that needs
  a new modeled operation adds it with a semantic contract, a resource bound, transcript
  coverage, exact replay, and a negative test, per COMPAT-5 in
  [the compatibility roadmap](.plans/GOMAD_NEXT.md#compat-5-targeted-deterministic-adapters-and-io-models).
- **Evidence over narration.** Work is done when its command produces the stated report on a
  clean checkout. A passing local run that depends on untracked state does not count.
- **Platform.** The boundary manifest qualifies `darwin/arm64` and `linux/amd64`. Each platform
  is its own qualification and artifacts replay only where they were produced. The macOS sandbox
  test and the DTrace clock audit remain `darwin/arm64` only. Each platform's compatibility packs
  are its own; `compatibility-pack-qualification` qualifies the requests that name the host.
- **Server source changes are allowed but bounded.** A change under `common`, `service`,
  `temporal`, or `tests/testcore` is acceptable when it isolates an optional provider behind a
  build tag or an injection seam and the default build is unchanged. A change that alters
  runtime behavior for production builds requires dedicated production review and
  regression evidence.
- **Validation scope.** The full `./tests` set is not run as a gate; a change is validated on the
  smoke selection plus the suites it affects, with `make gomad3-tests-qualification` as the
  on-demand local run.
- **Qualification identity.** Reports bind exact source and toolchain identities;
  pre-integration results do not qualify a combined candidate, and Darwin results do
  not substitute for Linux gates. An unavailable host or unexplained regression leaves
  acceptance incomplete in the task that owns the requirement. Transferred Linux and Darwin
  requirements remain incomplete under fn-128 and fn-149 and do not hold source-spec acceptance open.
- **Patch-policy boundary.** The collector-file prohibition remains in force.
  Overwriting collector-owned GC stamps requires an explicit patch-policy owner decision.
- **Preservation.** Keep CLI grammar/defaults, recorded formats, fixed-identity
  canonical bytes, error precedence, existing comments, fresh processes, native timer
  ownership, and separate seed/round/corpus transactions unless the owning spec
  explicitly authorizes a change.
- **Qualification claims.** Same-seed repeatability does not establish choice-tape
  exact replay; retain each workload's recorded disposition and replay requirements.
- **Feature preservation.** Removing or freezing shipped features requires an explicit
  owner decision; cleanup must not silently drop capabilities.

## Out of scope

- New fault injection, partition, crash-restart, and multi-node capabilities through
  `tools/gomad3sim`. Those are the simulation track and cannot host `testcore`.
- Multi-P scheduling, deterministic GC, DPOR, and preemption bounding. These are BUG-7 research
  items.
- Revival of gomad1 or gomad2. [GOMAD_CMP.md](.plans/GOMAD_CMP.md) records why.

## Runtime limits

- GC timing remains uncontrolled; a replay divergence needs a reproduced cause,
  not a speculative collector or topology change.
- Finite qualification and forced replay choices do not establish an unconditional
  determinism guarantee; retained soak cohorts supply measured statistical bounds.
- Spin loops with non-blocking selects stall virtual time; pollers must back off.
- Every Go bump requires porting and requalification. Dependency bumps affecting
  packed or adapted modules invalidate their pins and reopen the capability closure.

## Specs

<a id="open-findings"></a>
<a id="determinism-gaps"></a>
<a id="f10-follow-ups-deferred-scope"></a>

## F10: follow-ups (deferred scope) — [fn-105](.flow/specs/fn-105-gomad-follow-ups-deferred-scope.md)

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-105.1](.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.1.md) | ✅ Done | D1: shared completed-execution assessment owner |
| [fn-105.2](.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.2.md) | ✅ Done | D2: shared retention policy without merging strategy transactions |
| [fn-105.3](.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.3.md) | ✅ Done | D3: move public executor injection behind private dependencies |
| [fn-105.4](.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.4.md) | ⬜ Todo | D4: architecture fitness checks for package coverage, purity, and signature visibility |
| [fn-105.5](.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.5.md) | ⬜ Todo | D5: reconcile architecture, platform, and determinism documentation |
| [fn-105.6](.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.6.md) | ⬜ Todo | D6: seeded and fixed virtual-clock tick policies |
| [fn-105.7](.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.7.md) | ✅ Done | D7: add required macOS functional smoke CI |
| [fn-105.8](.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.8.md) | 🚧 In progress | D8: closure-mode support for downstream targets |
| [fn-105.9](.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.9.md) | ⬜ Todo | D9: shared downstream packs/source checks; native qualification moved to fn-149.3/fn-128.6 |
| [fn-105.10](.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.10.md) | ⬜ Todo | D10: downstream-seam guide |
| [fn-105.11](.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.11.md) | ✅ Done | D11: transfer conditional Linux audit to fn-128.3 (audit remains deferred) |
| [fn-105.12](.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.12.md) | ✅ Done | D12: transfer Linux replay correction to fn-128.2 (fix remains deferred) |
| [fn-105.13](.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.13.md) | ✅ Done | D13: make choice tracing opt-in for routine qualification |
| [fn-105.14](.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.14.md) | ✅ Done | D14: fix Darwin Chasm replay divergence and restore qualification |
| [fn-105.15](.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.15.md) | ⬜ Todo | D15: support larger choice traces when a workload needs them |
| [fn-105.16](.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.16.md) | ✅ Done | D16: investigate the forward-clock gRPC poll deadline mismatch |
| [fn-105.17](.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.17.md) | ✅ Done | D17: investigate Nexus operation determinism with two clusters |
| [fn-105.18](.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.18.md) | ✅ Done | D18: investigate worker cancellation delivery and timeout budgets |
| [fn-105.19](.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.19.md) | ✅ Done | D19: investigate activity fairness backlog readiness |
| [fn-105.20](.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.20.md) | ✅ Done | D20: investigate heartbeat timeout counting under virtual time |
| [fn-105.21](.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.21.md) | ✅ Done | D21: investigate host-clock reporting escapes and policy-compatible remedies |
| [fn-105.22](.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.22.md) | ✅ Done | D22: fix parallel Nexus outcome endpoint collisions |
| [fn-105.23](.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.23.md) | ✅ Done | D23: correct migration idempotency tests without changing the contract |
| [fn-105.24](.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.24.md) | ✅ Done | D24: synchronize the Nexus reset test before signalling |
| [fn-105.25](.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.25.md) | ✅ Done | D25: fix enhanced DescribeTaskQueue caching of report flags |
| [fn-105.26](.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.26.md) | ✅ Done | Rebind stale compatibility packs to the HEAD deterministic-I/O profile digest |
| [fn-105.27](.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.27.md) | ✅ Done | D18: qualify worker cancellation with forward clock and remove skip |
| [fn-105.28](.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.28.md) | ✅ Done | D19: establish fairness backlog readiness and remove skips |
| [fn-105.29](.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.29.md) | ✅ Done | D20: make heartbeat rejection deadlines explicit and remove skip |
| [fn-105.30](.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.30.md) | ✅ Done | D17: deliver explicit target environment and enable the two-cluster Nexus test |
| [fn-105.31](.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.31.md) | ✅ Done | D26: put forward clock ticks on the virtual clock and remove the D16 skip |
| [fn-105.32](.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.32.md) | ✅ Done | D27: state, pin, and remedy host-clock reporting escapes |

<a id="deep-modules-and-tool-interfaces-fn-109"></a>

## Deep modules and tool interfaces — [fn-109](.flow/specs/fn-109-gomad-deepen-modules-and-tool-interfaces.md)

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-109.1](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.1.md) | ✅ Done | Carry simulation bounds through real isolated execution. |
| [fn-109.2](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.2.md) | ✅ Done | Give local and isolated campaigns one normalized options owner |
| [fn-109.3](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.3.md) | ✅ Done | Give the seed controller one atomic completion transition |
| [fn-109.4](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.4.md) | ✅ Done | Resolve CLI installation and private child modes through one application construction path |
| [fn-109.5](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.5.md) | ✅ Done | Share plan and explore parsing directly and move semantic normalization to Runner |
| [fn-109.6](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.6.md) | ✅ Done | Move public executor injection behind private dependencies (D3); current source acceptance. |
| [fn-109.7](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.7.md) | ✅ Done | Preparation-owner source acceptance verified; native qualification remains deferred |
| [fn-109.8](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.8.md) | ⬜ Todo | Move analysis and compatibility review onto the preparation owner's inspection operation |
| [fn-109.9](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.9.md) | ⬜ Todo | Bounded adapter listing integrated; original acceptance gates remain open |
| [fn-109.10](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.10.md) | ⬜ Todo | Supply build, cache and adapter locations from one validated installation description |
| [fn-109.11](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.11.md) | ⬜ Todo | Capability/source-inventory owners integrated; inventory lint fixed, qualification pending |
| [fn-109.12](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.12.md) | ⬜ Todo | Separate detached Artifact references from owned opened handles |
| [fn-109.13](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.13.md) | ⬜ Todo | Generate host and runtime simulation-time codecs from one versioned definition |
| [fn-109.14](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.14.md) | ⬜ Todo | Hide generic model-wire slots behind typed network and volume commands |
| [fn-109.15](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.15.md) | ⬜ Todo | Characterize simulation progress ordering and choose the lifecycle interface from two designs |
| [fn-109.16](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.16.md) | ⬜ Todo | Implement the simulation progress lifecycle owner and remove caller-side accounting |
| [fn-109.17](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.17.md) | ⬜ Todo | Select backend-specific network listener and connection implementations at creation |
| [fn-109.18](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.18.md) | ⬜ Todo | Select backend-specific filesystem handle and mapping implementations at creation |
| [fn-109.19](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.19.md) | ⬜ Todo | Architecture checks; two World lint findings repaired, qualification pending |
| [fn-109.20](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.20.md) | ⬜ Todo | Reconcile architectural guidance with delivered owners and interfaces (D5). |
| [fn-109.21](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.21.md) | ⬜ Todo | Run final qualification and retain the finding completion matrix |
| [fn-109.22](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.22.md) | ✅ Done | Repair the simulation-exploration target path so a real campaign completes |
| [fn-109.23](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.23.md) | ⬜ Todo | Repair module-aware lint routing and supply the nested host gates |
| [fn-109.24](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.24.md) | ⬜ Todo | Restore repository-relative lint exclusion matching |
| [fn-109.25](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.25.md) | ⬜ Todo | Preserve lifecycle fault resolution while repairing exhaustive lint |
| [fn-109.26](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.26.md) | ⬜ Todo | Restore Runner semantic ownership in CLI callers |
| [fn-109.27](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.27.md) | ⬜ Todo | Correct current R18 preservation disclosures |
| [fn-109.28](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.28.md) | ✅ Done | Preserve campaign policies while checking cleanup errors |
| [fn-109.29](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.29.md) | ⬜ Todo | Preserve private artifact payload cleanup and error identity |
| [fn-109.30](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.30.md) | ⬜ Todo | Preserve public artifact copy cleanup and handle lifetime |
| [fn-109.31](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.31.md) | ⬜ Todo | Preserve artifact directory and shared-verifier cleanup |
| [fn-109.32](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.32.md) | ⬜ Todo | Preserve concrete error callback provenance in architecture checks |
| [fn-109.33](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.33.md) | ⬜ Todo | Preserve artifact reflection helper coverage and clone isolation |
| [fn-109.34](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.34.md) | ⬜ Todo | Check CLI private-mode fixture reader cleanup |
| [fn-109.35](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.35.md) | ⬜ Todo | Preserve corpus reader cleanup and publication failures |
| [fn-109.36](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.36.md) | ⬜ Todo | Preserve Choice Exploration stopping predicates and round identities |
| [fn-109.37](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.37.md) | ⬜ Todo | Qualification cleanup and import lint clean; original qualification open |
| [fn-109.38](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.38.md) | ⬜ Todo | Target and pure-policy lint clean; original qualification remains open |
| [fn-109.39](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.39.md) | ⬜ Todo | Cleanup lint repaired; fault, pin and original qualification remain open |
| [fn-109.40](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.40.md) | ⬜ Todo | Bounded commands and cleanup reviewed; lint clean, original qualification open |
| [fn-109.41](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.41.md) | ⬜ Todo | Canonical JSON exhaustive lint repaired; original qualification remains open |
| [fn-109.42](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.42.md) | ⬜ Todo | Exact-pack exhaustive lint repaired; original qualification remains open |
| [fn-109.43](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.43.md) | ⬜ Todo | Five-import admission repaired; original qualification remains open |
| [fn-109.44](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.44.md) | ⬜ Todo | Four mechanical lint findings repaired; original qualification remains open |
| [fn-109.45](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.45.md) | ⬜ Todo | Seven mechanical lint findings repaired; full lint and qualification remain open |
| [fn-109.46](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.46.md) | ⬜ Todo | Stdout reports repaired; full lint and original qualification remain open |
| [fn-109.47](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.47.md) | ⬜ Todo | Archive cleanup repaired; full lint and original qualification remain open |
| [fn-109.48](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.48.md) | ⬜ Todo | Patch cleanup repaired; full lint and original qualification remain open |
| [fn-109.49](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.49.md) | ⬜ Todo | Adapter cache cleanup repaired; full lint and original qualification remain open |

<a id="runtime-patch-minimization-fn-110"></a>

## Runtime patch minimization — [fn-110](.flow/specs/fn-110-gomad-minimize-the-runtime-patch.md)

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-110.1](.flow/tasks/fn-110-gomad-minimize-the-runtime-patch.1.md) | ✅ Done | Record the patch, overlay, and qualification baseline |
| [fn-110.2](.flow/tasks/fn-110-gomad-minimize-the-runtime-patch.2.md) | ✅ Done | Move the three scheduler implementations into the runtime overlay |
| [fn-110.3](.flow/tasks/fn-110-gomad-minimize-the-runtime-patch.3.md) | ⬜ Todo | Relocate crypto initialization and syscall declarations to overlays |
| [fn-110.4](.flow/tasks/fn-110-gomad-minimize-the-runtime-patch.4.md) | ⬜ Todo | Emit the canonical one-context-line patch and pin regeneration to the descriptor |
| [fn-110.5](.flow/tasks/fn-110-gomad-minimize-the-runtime-patch.5.md) | ⬜ Todo | Qualify the final candidate and publish measurements and guidance |

<a id="quality-assessment-2026-10-01"></a>

## Determinism assurance and test strategy — [fn-112](.flow/specs/fn-112-gomad-determinism-assurance-and-test.md)

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-112.1](.flow/tasks/fn-112-gomad-determinism-assurance-and-test.1.md) | ✅ Done | Reproduce and fix the failing gomad3 workflow jobs |
| [fn-112.2](.flow/tasks/fn-112-gomad-determinism-assurance-and-test.2.md) | ✅ Done | Run the orphaned simulation, overlay, and choice-replay tests in a gate |
| [fn-112.3](.flow/tasks/fn-112-gomad-determinism-assurance-and-test.3.md) | ✅ Done | Record a runtime-state digest at each choice point in a diagnostic trace |
| [fn-112.4](.flow/tasks/fn-112-gomad-determinism-assurance-and-test.4.md) | ✅ Done | Plumb diagnostics through the Runner and add the trace differ |
| [fn-112.5](.flow/tasks/fn-112-gomad-determinism-assurance-and-test.5.md) | ✅ Done | Inventory seeded-stream draw sites and check host-timed paths at runtime |
| [fn-112.6](.flow/tasks/fn-112-gomad-determinism-assurance-and-test.6.md) | ✅ Done | Add conformance fixtures for unverified channels and state the closure-mode limit |
| [fn-112.7](.flow/tasks/fn-112-gomad-determinism-assurance-and-test.7.md) | ✅ Done | Compare the filesystem and TCP models with the host OS on generated sequences |
| [fn-112.8](.flow/tasks/fn-112-gomad-determinism-assurance-and-test.8.md) | ✅ Done | Drive explore, replay, and kill-then-resume through the built CLI |
| [fn-112.9](.flow/tasks/fn-112-gomad-determinism-assurance-and-test.9.md) | ✅ Done | Consolidate the change-detector tests with a retained mapping; retained source acceptance |
| [fn-112.10](.flow/tasks/fn-112-gomad-determinism-assurance-and-test.10.md) | ⬜ Todo | Add the scheduled determinism soak gate and update the docs to the delivered state |
| [fn-112.11](.flow/tasks/fn-112-gomad-determinism-assurance-and-test.11.md) | ✅ Done | Preserve watchdog classification when a killed target has no I/O terminal |
| [fn-112.12](.flow/tasks/fn-112-gomad-determinism-assurance-and-test.12.md) | ✅ Done | Resolve the native model compiler from the standard host-test entrypoint |
| [fn-112.13](.flow/tasks/fn-112-gomad-determinism-assurance-and-test.13.md) | ✅ Done | Execute watchdog diagnostic replay without requiring an exact I/O transcript |
| [fn-112.14](.flow/tasks/fn-112-gomad-determinism-assurance-and-test.14.md) | ✅ Done | Preserve parent cancellation classification when an exploration round finishes |
| [fn-112.15](.flow/tasks/fn-112-gomad-determinism-assurance-and-test.15.md) | ✅ Done | Make TestWatchdogDiagnosticReplayUsesCapturedInputs reliable |
| [fn-112.16](.flow/tasks/fn-112-gomad-determinism-assurance-and-test.16.md) | ✅ Done | Keep two retained successes with one outcome signature as distinct artifacts |

<a id="maintenance-cost"></a>

## Version-pin maintenance — [fn-113](.flow/specs/fn-113-gomad-reduce-version-pin-maintenance.md)

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-113.1](.flow/tasks/fn-113-gomad-reduce-version-pin-maintenance.1.md) | ✅ Done | Pin-impact source acceptance; [current inventory](.flow/artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-1/source-acceptance-20261008/current-inventory.json) and [bump steps](.flow/artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-1/source-acceptance-20261008/manual-steps.md) |
| [fn-113.2](.flow/tasks/fn-113-gomad-reduce-version-pin-maintenance.2.md) | ✅ Done | Retained R3 source acceptance and three-draw SHIP; native qualification deferred |
| [fn-113.3](.flow/tasks/fn-113-gomad-reduce-version-pin-maintenance.3.md) | ✅ Done | Retained R4 source acceptance verified; native qualification deferred |
| [fn-113.4](.flow/tasks/fn-113-gomad-reduce-version-pin-maintenance.4.md) | ✅ Done | Retained R5/source R6 acceptance and [matched source measurement](.flow/artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-4/source-acceptance-20261008/measurement.md); native gates remain deferred |

<a id="search-path-findings-fn-114"></a>

## Search-path findings — [fn-114](.flow/specs/fn-114-gomad-correct-search-path-defects-and.md)

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-114.1](.flow/tasks/fn-114-gomad-correct-search-path-defects-and.1.md) | ✅ Done | Re-anchor the ten findings and reproduce C3 on the unmodified tree; historical evidence retains its original candidate identity |
| [fn-114.2](.flow/tasks/fn-114-gomad-correct-search-path-defects-and.2.md) | ✅ Done | Historical C2/E3 controls and counterexamples; the predicted C2 same-seed prefix failure is narrowed, not reproduced by cross-seed forcing |
| [fn-114.3](.flow/tasks/fn-114-gomad-correct-search-path-defects-and.3.md) | ✅ Done | C1/C4 delivered in source; corpus identity binds environment/tick policy and provenance rejects coverage instrumentation |
| [fn-114.4](.flow/tasks/fn-114-gomad-correct-search-path-defects-and.4.md) | ✅ Done | C3 delivered in source; typed divergent candidates and completed siblings survive commit and resume |
| [fn-114.5](.flow/tasks/fn-114-gomad-correct-search-path-defects-and.5.md) | ✅ Done | C2 stable creation-bound timer-callback identities delivered; parentless exceptions remain inventoried and same-seed counterexamples retain their meaning |
| [fn-114.6](.flow/tasks/fn-114-gomad-correct-search-path-defects-and.6.md) | ✅ Done | E5 delivered in source; the replay-plan start ordinal controls choice-frontier expansion |
| [fn-114.7](.flow/tasks/fn-114-gomad-correct-search-path-defects-and.7.md) | ✅ Done | E1 delivered in source; ordinary guidance skips answered seeds, regression mode is explicit and selection/counts remain frozen across resume/shards |
| [fn-114.8](.flow/tasks/fn-114-gomad-correct-search-path-defects-and.8.md) | ✅ Done | E6 resume delivered in source with accepted artifacts and consumed budgets persisted; typed scenario shrinking remains open |
| [fn-114.9](.flow/tasks/fn-114-gomad-correct-search-path-defects-and.9.md) | ✅ Done | E2 shared prepared targets delivered with self-contained private-copy fallback |
| [fn-114.10](.flow/tasks/fn-114-gomad-correct-search-path-defects-and.10.md) | ✅ Done | E2 corpus accounting/pruning/merge delivered; retained-byte measurements remain historical and current native measurement belongs to fn-149/fn-128 |
| [fn-114.11](.flow/tasks/fn-114-gomad-correct-search-path-defects-and.11.md) | ✅ Done | E3 readiness recording delivered in source; select-poll records remain in the Choice Trace for replay |
| [fn-114.12](.flow/tasks/fn-114-gomad-correct-search-path-defects-and.12.md) | ✅ Done | E3 frontier suppression delivered for seven proven shapes with two polled non-nil cases; unlisted shapes and selects with three or more polled cases remain expanded, and native counts retain historical identities |
| [fn-114.13](.flow/tasks/fn-114-gomad-correct-search-path-defects-and.13.md) | ✅ Done | E4 source acceptance delivered and historical two-user premise refuted; head-class scheduling offers user-only alternatives, with current native controls deferred |
| [fn-114.14](.flow/tasks/fn-114-gomad-correct-search-path-defects-and.14.md) | ✅ Done | R12 delivered-behavior docs and retained source acceptance complete; current native qualification remains with fn-149/fn-128 |
| [fn-114.15](.flow/tasks/fn-114-gomad-correct-search-path-defects-and.15.md) | ✅ Done | E6 per-parent minimizer workspace isolation delivered; explicit resume fails closed on changed/corrupt state |
| [fn-114.16](.flow/tasks/fn-114-gomad-correct-search-path-defects-and.16.md) | ✅ Done | E2 accounting limit documented; campaign budgets charge each standalone artifact in full, while corpus sharing and merged-record target deduplication use their separate rules |

## Deferred Linux qualification and repairs — [fn-128](.flow/specs/fn-128-gomad-deferred-linux-qualification-and.md)

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-128.1](.flow/tasks/fn-128-gomad-deferred-linux-qualification-and.1.md) | ⛔ Blocked | Native Linux candidate and runtime/host evidence |
| [fn-128.2](.flow/tasks/fn-128-gomad-deferred-linux-qualification-and.2.md) | ⛔ Blocked | Causal Linux replay fix and strict CI restoration |
| [fn-128.3](.flow/tasks/fn-128-gomad-deferred-linux-qualification-and.3.md) | ⛔ Blocked | Conditional Linux host-clock audit or owner disposition |
| [fn-128.4](.flow/tasks/fn-128-gomad-deferred-linux-qualification-and.4.md) | ⛔ Blocked | Linux model, architecture, pack and affected-workload qualification |
| [fn-128.5](.flow/tasks/fn-128-gomad-deferred-linux-qualification-and.5.md) | ⛔ Blocked | Actual Linux determinism soak and measured bound |
| [fn-128.6](.flow/tasks/fn-128-gomad-deferred-linux-qualification-and.6.md) | ⛔ Blocked | Downstream Linux analyses, packs, exact replay and guidance |
| [fn-128.7](.flow/tasks/fn-128-gomad-deferred-linux-qualification-and.7.md) | ⛔ Blocked | Final source-bound Linux matrix, review and documentation |

## Deferred Darwin qualification - [fn-149](.flow/specs/fn-149-gomad-deferred-darwin-qualification.md)

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-149.1](.flow/tasks/fn-149-gomad-deferred-darwin-qualification.1.md) | ⛔ Blocked | Native Darwin baseline, runtime and clock evidence |
| [fn-149.2](.flow/tasks/fn-149-gomad-deferred-darwin-qualification.2.md) | ⛔ Blocked | Darwin model, pack and integration qualification |
| [fn-149.3](.flow/tasks/fn-149-gomad-deferred-darwin-qualification.3.md) | ⛔ Blocked | Real downstream Darwin qualification and guidance |
| [fn-149.4](.flow/tasks/fn-149-gomad-deferred-darwin-qualification.4.md) | ⛔ Blocked | Scheduled/dispatched Darwin soak and final qualification matrix |
