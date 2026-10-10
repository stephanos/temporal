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
<a id="maintenance-cost"></a>
<a id="search-path-findings-fn-114"></a>

Completed specs and their evidence remain in `.flow/` and Git history, including
[version-pin maintenance](.flow/specs/fn-113-gomad-reduce-version-pin-maintenance.md)
and [search-path findings](.flow/specs/fn-114-gomad-correct-search-path-defects-and.md).

## Immediate delivery order

Owner priority (2026-10-10). Finish the already-running fn-109 task 72 verification,
then take fn-155 next, ahead of the remaining source queue below. Begin with
fn-155 `.1 → .2 → .8 → .3`, then follow its dependencies
through `.7`'s boundary decision. fn-109.17/.18 and fn-110.3 still wait for fn-155.7.
The execution proof needs darwin/arm64 or linux/amd64. Preserve retained acceptance
and native deferrals; this priority grants no PR, push or CI authority.

1. Finish fn-112 task 10's retained source acceptance for the delivered soak gate and shared documentation. Actual native soak runs and measured bounds remain with fn-149.4 and fn-128.5/.7; no workflow dispatch is authorized.
2. Reconcile the combined D26/fn-110 source candidate, fn-112 tasks 5/16/9, D27 and fn-109 tasks 2-6 against their retained source checks, preservation and source reviews. Keep lint, both-source-set static checks, generated validation, first-baseline and preservation requirements open wherever unproved. Native runtime/clock proof belongs to deferred fn-149.1 and fn-128.1/.4/.7.
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
| [fn-109.8](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.8.md) | ✅ Done | Move analysis and compatibility review onto the preparation owner's inspection operation |
| [fn-109.9](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.9.md) | 🚧 In progress | Bounded adapter listing integrated; original acceptance gates remain open |
| [fn-109.10](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.10.md) | ⬜ Todo | Supply build, cache and adapter locations from one validated installation description |
| [fn-109.11](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.11.md) | ⬜ Todo | Capability/source-inventory owners integrated; inventory lint fixed, qualification pending |
| [fn-109.12](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.12.md) | ⬜ Todo | Separate detached Artifact references from owned opened handles |
| [fn-109.13](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.13.md) | ⬜ Todo | Generate host and runtime simulation-time codecs from one versioned definition |
| [fn-109.14](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.14.md) | ⬜ Todo | Hide generic model-wire slots behind typed network and volume commands |
| [fn-109.15](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.15.md) | ⬜ Todo | Characterize simulation progress ordering and choose the lifecycle interface from two designs |
| [fn-109.16](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.16.md) | ⬜ Todo | Implement the simulation progress lifecycle owner and remove caller-side accounting |
| [fn-109.17](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.17.md) | ⬜ Todo | Select backend-specific network listener and connection implementations at creation |
| [fn-109.18](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.18.md) | ⬜ Todo | Select backend-specific filesystem handle and mapping implementations at creation |
| [fn-109.19](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.19.md) | 🚧 In progress | Architecture digest lint repaired; original acceptance remains open |
| [fn-109.20](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.20.md) | ⬜ Todo | Reconcile architectural guidance with delivered owners and interfaces (D5). |
| [fn-109.21](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.21.md) | ⬜ Todo | Run final qualification and retain the finding completion matrix |
| [fn-109.22](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.22.md) | ✅ Done | Repair the simulation-exploration target path so a real campaign completes |
| [fn-109.23](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.23.md) | 🚧 In progress | Repair module-aware lint routing and supply the nested host gates |
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
| [fn-109.40](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.40.md) | 🚧 In progress | Command-test readiness repaired; retained mechanism acceptance remains open |
| [fn-109.41](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.41.md) | ⬜ Todo | Canonical JSON exhaustive lint repaired; original qualification remains open |
| [fn-109.42](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.42.md) | ⬜ Todo | Exact-pack exhaustive lint repaired; original qualification remains open |
| [fn-109.43](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.43.md) | ⬜ Todo | Five-import admission repaired; original qualification remains open |
| [fn-109.44](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.44.md) | ⬜ Todo | Four mechanical lint findings repaired; original qualification remains open |
| [fn-109.45](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.45.md) | ⬜ Todo | Seven mechanical lint findings repaired; full lint and qualification remain open |
| [fn-109.46](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.46.md) | ⬜ Todo | Stdout reports repaired; full lint and original qualification remain open |
| [fn-109.47](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.47.md) | ⬜ Todo | Archive cleanup repaired; full lint and original qualification remain open |
| [fn-109.48](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.48.md) | ⬜ Todo | Patch cleanup repaired; full lint and original qualification remain open |
| [fn-109.49](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.49.md) | ⬜ Todo | Adapter cache cleanup repaired; full lint and original qualification remain open |
| [fn-109.50](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.50.md) | 🚧 In progress | Check five terminal generator diagnostics without changing statuses |
| [fn-109.51](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.51.md) | 🚧 In progress | Reconcile one usage-status fixture with the preserved original contract |
| [fn-109.52](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.52.md) | ✅ Done | Check remaining maintainer diagnostics while preserving primary outcomes |
| [fn-109.53](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.53.md) | 🚧 In progress | Check CLI diagnostics while preserving primary outcomes |
| [fn-109.54](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.54.md) | 🚧 In progress | Check doctor reports and classify output failures |
| [fn-109.55](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.55.md) | 🚧 In progress | Check verify-only replay report delivery |
| [fn-109.56](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.56.md) | 🚧 In progress | Check builder cleanup while preserving publication outcomes |
| [fn-109.57](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.57.md) | 🚧 In progress | Check test cleanup results without changing resource lifetimes |
| [fn-109.58](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.58.md) | 🚧 In progress | Check inspection and invalid compiler-fixture cleanup |
| [fn-109.59](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.59.md) | 🚧 In progress | Make preserved test-switch no-op cases explicit |
| [fn-109.60](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.60.md) | 🚧 In progress | Check six child-fixture outputs without changing process outcomes |
| [fn-109.61](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.61.md) | 🚧 In progress | Observe competing-build lock contention instead of sleeping |
| [fn-109.62](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.62.md) | 🚧 In progress | Bound progress-test startup and observe early completion |
| [fn-109.63](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.63.md) | 🚧 In progress | Decompose local campaign orchestration before further Runner changes |
| [fn-109.64](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.64.md) | 🚧 In progress | Check watchdog fixture readiness writes without masking setup failure |
| [fn-109.65](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.65.md) | 🚧 In progress | Restore explicit scripted Runner preparation and bootstrap coverage |
| [fn-109.66](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.66.md) | 🚧 In progress | Restore explicit scripted progress and retention assertions |
| [fn-109.67](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.67.md) | 🚧 In progress | Preserve busy host workloads while correcting spin lint |
| [fn-109.68](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.68.md) | 🚧 In progress | Restore explicit scripted Choice Exploration coverage |
| [fn-109.69](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.69.md) | 🚧 In progress | Restore explicit scripted retained-success coverage |
| [fn-109.70](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.70.md) | 🚧 In progress | Restore explicit scripted Choice Exploration divergence coverage |
| [fn-109.71](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.71.md) | 🚧 In progress | Restore explicit scripted completion coverage |
| [fn-109.72](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.72.md) | 🚧 In progress | Restore explicit scripted seed completion statistics coverage |
| [fn-109.73](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.73.md) | 🚧 In progress | Restore explicit scripted Unix campaign-mode coverage |
| [fn-109.74](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.74.md) | 🚧 In progress | Restore explicit scripted retention calibration and policy coverage |
| [fn-109.75](.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.75.md) | 🚧 In progress | Restore scripted diagnostics capacity bounds and simulation inspection coverage |

<a id="runtime-patch-minimization-fn-110"></a>

## Runtime patch minimization — [fn-110](.flow/specs/fn-110-gomad-minimize-the-runtime-patch.md)

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-110.1](.flow/tasks/fn-110-gomad-minimize-the-runtime-patch.1.md) | ✅ Done | Record the patch, overlay, and qualification baseline |
| [fn-110.2](.flow/tasks/fn-110-gomad-minimize-the-runtime-patch.2.md) | ✅ Done | Move the three scheduler implementations into the runtime overlay |
| [fn-110.3](.flow/tasks/fn-110-gomad-minimize-the-runtime-patch.3.md) | ⬜ Todo | Relocate crypto initialization and syscall declarations to overlays |
| [fn-110.4](.flow/tasks/fn-110-gomad-minimize-the-runtime-patch.4.md) | ⬜ Todo | Emit the canonical one-context-line patch and pin regeneration to the descriptor |
| [fn-110.5](.flow/tasks/fn-110-gomad-minimize-the-runtime-patch.5.md) | ⬜ Todo | Qualify the final candidate and publish measurements and guidance |

<a id="syscall-boundary-fn-155"></a>

## Syscall-level I/O boundary — [fn-155](.flow/specs/fn-155-gomad-syscall-level-io-boundary-from.md)

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-155.1](.flow/tasks/fn-155-gomad-syscall-level-io-boundary-from.1.md) | 🚧 In progress | Port the virtual descriptor layer and syscall edge onto the Gomad toolchain |
| [fn-155.2](.flow/tasks/fn-155-gomad-syscall-level-io-boundary-from.2.md) | ⬜ Todo | Select the boundary and adapter exclusions per run; bind into artifact identity |
| [fn-155.3](.flow/tasks/fn-155-gomad-syscall-level-io-boundary-from.3.md) | ⬜ Todo | Record descriptor I/O; prove same-seed determinism for a single-process gRPC workload |
| [fn-155.4](.flow/tasks/fn-155-gomad-syscall-level-io-boundary-from.4.md) | ⬜ Todo | Serve multi-node simulation traffic through virtual descriptors in both backends |
| [fn-155.5](.flow/tasks/fn-155-gomad-syscall-level-io-boundary-from.5.md) | ⬜ Todo | Soak single-process and multi-node workloads with the boundary on |
| [fn-155.6](.flow/tasks/fn-155-gomad-syscall-level-io-boundary-from.6.md) | ⬜ Todo | Classify the 15 adapters and run network adapters excluded |
| [fn-155.7](.flow/tasks/fn-155-gomad-syscall-level-io-boundary-from.7.md) | ⬜ Todo | Measure size, record the files/DNS decision and annotate dependent work |
| [fn-155.8](.flow/tasks/fn-155-gomad-syscall-level-io-boundary-from.8.md) | ⬜ Todo | Admit the edge's socket entry points in capability guard and closure policy |

<a id="append-log-storage-fn-152"></a>

## Runner append-log storage — [fn-152](.flow/specs/fn-152-gomad-runner-storage-on-one-append-only.md)

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-152.1](.flow/tasks/fn-152-gomad-runner-storage-on-one-append-only.1.md) | ⬜ Todo | Add protected framing, guarded opens and read-only versus writer replay |
| [fn-152.2](.flow/tasks/fn-152-gomad-runner-storage-on-one-append-only.2.md) | ⬜ Todo | Gate artifact lineage and owned live-reference cleanup |
| [fn-152.3](.flow/tasks/fn-152-gomad-runner-storage-on-one-append-only.3.md) | ⬜ Todo | Replace campaign lifecycle, recovery and resume files with one log |
| [fn-152.4](.flow/tasks/fn-152-gomad-runner-storage-on-one-append-only.4.md) | ⬜ Todo | Integrate durable seed outcomes, scheduling and resume |
| [fn-152.5](.flow/tasks/fn-152-gomad-runner-storage-on-one-append-only.5.md) | ⬜ Todo | Commit both exploration strategies through one atomic round owner |
| [fn-152.6](.flow/tasks/fn-152-gomad-runner-storage-on-one-append-only.6.md) | ⬜ Todo | Replay corpus admission and eviction into the bounded live hash index |
| [fn-152.7](.flow/tasks/fn-152-gomad-runner-storage-on-one-append-only.7.md) | ⬜ Todo | Concatenate source-scoped shard logs after complete preflight |
| [fn-152.8](.flow/tasks/fn-152-gomad-runner-storage-on-one-append-only.8.md) | ⬜ Todo | Inspect log and corpus state and migrate storage consumers |
| [fn-152.9](.flow/tasks/fn-152-gomad-runner-storage-on-one-append-only.9.md) | ⬜ Todo | Retain integrated crash evidence and update current storage contracts |
| [fn-152.10](.flow/tasks/fn-152-gomad-runner-storage-on-one-append-only.10.md) | ⬜ Todo | Persist admitted work and terminal receipts with atomic policy stop/drain |

<a id="canonical-json-publication-cleanup-fn-153"></a>

## JSON and file-publication cleanup - [fn-153](.flow/specs/fn-153-gomad-retire-canonical-json-and-private.md)

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-153.1](.flow/tasks/fn-153-gomad-retire-canonical-json-and-private.1.md) | ⬜ Todo | Introduce narrow strict JSON decoding without a shared encoder |
| [fn-153.2](.flow/tasks/fn-153-gomad-retire-canonical-json-and-private.2.md) | ⬜ Todo | Centralize staged and streamed whole-file publication |
| [fn-153.3](.flow/tasks/fn-153-gomad-retire-canonical-json-and-private.3.md) | ⬜ Todo | Migrate execution records and I/O identity validation |
| [fn-153.4](.flow/tasks/fn-153-gomad-retire-canonical-json-and-private.4.md) | ⬜ Todo | Migrate preparation identities and preserve live-capability payloads |
| [fn-153.5](.flow/tasks/fn-153-gomad-retire-canonical-json-and-private.5.md) | ⬜ Todo | Replace World's generic encoder with a bounded domain-local stdlib codec |
| [fn-153.6](.flow/tasks/fn-153-gomad-retire-canonical-json-and-private.6.md) | ⬜ Todo | Migrate compatibility authoring without changing approvals' meaning |
| [fn-153.7](.flow/tasks/fn-153-gomad-retire-canonical-json-and-private.7.md) | ⬜ Todo | Migrate qualification evidence and shared report publication |
| [fn-153.8](.flow/tasks/fn-153-gomad-retire-canonical-json-and-private.8.md) | ⬜ Todo | Migrate qualification sets and generated manifest inputs |
| [fn-153.9](.flow/tasks/fn-153-gomad-retire-canonical-json-and-private.9.md) | ⬜ Todo | Migrate CLI JSON delivery and retain the public upgrade facade |
| [fn-153.10](.flow/tasks/fn-153-gomad-retire-canonical-json-and-private.10.md) | ⬜ Todo | Migrate choice exploration and Runner identity projections |
| [fn-153.11](.flow/tasks/fn-153-gomad-retire-canonical-json-and-private.11.md) | ⬜ Todo | Migrate simulation identities while preserving paired target formulas |
| [fn-153.12](.flow/tasks/fn-153-gomad-retire-canonical-json-and-private.12.md) | ⬜ Todo | Migrate replay composition, minimizer checkpoints and corpus semantics |
| [fn-153.13](.flow/tasks/fn-153-gomad-retire-canonical-json-and-private.13.md) | ⬜ Todo | Move streamed archive and validated patch publication to hostfs |
| [fn-153.14](.flow/tasks/fn-153-gomad-retire-canonical-json-and-private.14.md) | ⬜ Todo | Preserve the builder's staged stamp and launcher transaction |
| [fn-153.15](.flow/tasks/fn-153-gomad-retire-canonical-json-and-private.15.md) | ⬜ Todo | Migrate architecture effect fixtures before removing their source owner |
| [fn-153.16](.flow/tasks/fn-153-gomad-retire-canonical-json-and-private.16.md) | ⬜ Todo | Delete the generic canonical package and its obsolete owner entries |
| [fn-153.17](.flow/tasks/fn-153-gomad-retire-canonical-json-and-private.17.md) | ⬜ Todo | Regenerate final-input approvals, packs, goldens and identity outputs |
| [fn-153.18](.flow/tasks/fn-153-gomad-retire-canonical-json-and-private.18.md) | ⬜ Todo | Reconcile contracts and verify the integrated cleanup |

<a id="virtual-network-stalling-fn-154"></a>

## Virtual network stalling and simplification - [fn-154](.flow/specs/fn-154-gomad-simpler-virtual-network-with.md)

| Name / ID | Status | Description |
| --- | --- | --- |
| [fn-154.1](.flow/tasks/fn-154-gomad-simpler-virtual-network-with.1.md) | ⬜ Todo | Add explicit per-direction byte and virtual stall limits |
| [fn-154.2](.flow/tasks/fn-154-gomad-simpler-virtual-network-with.2.md) | ⬜ Todo | Propagate limits and admit held/timeout vocabulary in the generated network codec |
| [fn-154.3](.flow/tasks/fn-154-gomad-simpler-virtual-network-with.3.md) | ⬜ Todo | Preserve persistent stall timeout identity through local and process adapters |
| [fn-154.4](.flow/tasks/fn-154-gomad-simpler-virtual-network-with.4.md) | ⬜ Todo | Unify local connection mechanics behind a byte-bounded queue owner |
| [fn-154.5](.flow/tasks/fn-154-gomad-simpler-virtual-network-with.5.md) | ⬜ Todo | Hold partitioned bytes and release FIFO through effective topology changes |
| [fn-154.6](.flow/tasks/fn-154-gomad-simpler-virtual-network-with.6.md) | ⬜ Todo | Expire stalled connections through autonomous virtual-time work |
| [fn-154.7](.flow/tasks/fn-154-gomad-simpler-virtual-network-with.7.md) | ⬜ Todo | Bound pending partitioned dials with independent virtual stall intervals |
| [fn-154.8](.flow/tasks/fn-154-gomad-simpler-virtual-network-with.8.md) | ⬜ Todo | Remove delivery history and hashes with coherent record and codec migration |
| [fn-154.9](.flow/tasks/fn-154-gomad-simpler-virtual-network-with.9.md) | ⬜ Todo | Pin shared-connection and in-process fault conformance |
| [fn-154.10](.flow/tasks/fn-154-gomad-simpler-virtual-network-with.10.md) | ⬜ Todo | Execute process network cases through registered isolated Runner gates |
| [fn-154.11](.flow/tasks/fn-154-gomad-simpler-virtual-network-with.11.md) | ⬜ Todo | Verify framed streams and the divided payload replay guarantees |
| [fn-154.12](.flow/tasks/fn-154-gomad-simpler-virtual-network-with.12.md) | ⬜ Todo | Reconcile network contracts and verify the frozen integrated candidate |

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
| [fn-112.10](.flow/tasks/fn-112-gomad-determinism-assurance-and-test.10.md) | 🚧 In progress | Finish retained source acceptance for the soak gate and shared documentation |
| [fn-112.11](.flow/tasks/fn-112-gomad-determinism-assurance-and-test.11.md) | ✅ Done | Preserve watchdog classification when a killed target has no I/O terminal |
| [fn-112.12](.flow/tasks/fn-112-gomad-determinism-assurance-and-test.12.md) | ✅ Done | Resolve the native model compiler from the standard host-test entrypoint |
| [fn-112.13](.flow/tasks/fn-112-gomad-determinism-assurance-and-test.13.md) | ✅ Done | Execute watchdog diagnostic replay without requiring an exact I/O transcript |
| [fn-112.14](.flow/tasks/fn-112-gomad-determinism-assurance-and-test.14.md) | ✅ Done | Preserve parent cancellation classification when an exploration round finishes |
| [fn-112.15](.flow/tasks/fn-112-gomad-determinism-assurance-and-test.15.md) | ✅ Done | Make TestWatchdogDiagnosticReplayUsesCapturedInputs reliable |
| [fn-112.16](.flow/tasks/fn-112-gomad-determinism-assurance-and-test.16.md) | ✅ Done | Keep two retained successes with one outcome signature as distinct artifacts |

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
