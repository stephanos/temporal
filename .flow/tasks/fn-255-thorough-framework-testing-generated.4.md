---
satisfies: [R3, R7]
---
# fn-255-thorough-framework-testing-generated.4 cargo-mutants per framework crate; triage the first run

Touches: [model/umpire/.cargo/mutants*.toml, model/umpire/mise.toml, model/umpire/dev/xtask/src/mutants_scheduled.rs, model/umpire/dev/xtask/src/mutants_scheduled/**, model/umpire/dev/xtask/src/main.rs, model/umpire/dev/xtask/src/lib.rs, model/umpire/dev/xtask/tests/mutants*.rs, model/umpire/dev/gate/budgets.toml, model/umpire/dev/xtask/README.md, model/design/gate-speed.md]

## Description
Deliver per-framework-crate mutation commands, a bounded fixed smoke manifest, and scheduled campaign/triage reports (R3), using R7's authoritative work and time bounds.

**Size:** M
**Files:** mutation exclusions, framework tool pin, owned native command/module and controls, admitted budget entries, and command/evidence documentation.

### Prerequisites and ownership
- Consume committed fn-254.12 before execution or task closure. Smoke, per-mutant and scheduled limits come from its single committed authoritative framework table; remove duplicate provisional tables rather than introducing another budget authority. Set timeouts from measured test baselines.
- Default source ownership is limited to the listed configuration, mutation command/control and documentation paths plus the invocation's isolated mutant scratch source. The coordinator serializes shared pins, budget and dispatcher/module registration edits with .2/.3/.5/.6/.7.
- Real missed-mutant fixes may require production source or test edits outside this default scope. Before changing them, add the exact paths to task scope and acquire exclusive ownership from the coordinator. Never mutate a shared working checkout in place. Inspection can run in parallel; production fixes wait for ownership.

### Approach
- Pin cargo-mutants through the framework mise configuration. Exclude generated code, `Debug`/`Display` implementations, test support and Temporal code; prove exclusions against the pinned tool's actual inventory/schema.
- Register the explicit framework-native mutation command while preserving current native stage DAG, cancellation/completion custody, shared resource admission and actual test/TLC count authorities. Reuse the existing supervisor; do not create an independent scheduler. Bound nested test/compiler/JVM work as well as mutant jobs.
- Support production framework crates, starting with `umpire-front`, `umpire-ir`, `umpire-checker`, `umpire-tla`, `umpire-tlc` and `umpire-dsl`; document intentional support-crate exclusions. Use landed nextest integration and retain required baseline tests and doctests.
- At closure run a committed fixed smoke manifest with hard mutant-count, per-mutant timeout, output and total wall-clock caps admitted in the authoritative table. Record source/tool identity and exact selected mutant manifest. Full per-crate campaigns remain scheduled-only, outside quick/check/validate and outside closure prerequisites.
- Scheduled capped reports retain per-crate caught, missed, timed-out and unviable counts, exact identities, elapsed time, completed and remaining work, and reproducible resume inputs. Report partial work on timeout, cancellation and tool failure rather than losing already observed outcomes. Timeout and unviable results never count as caught.
- Triage observed missed mutants as test gaps (add a regression by default), equivalent mutants with a reason, or dead code to remove under exact exclusive ownership. Preserve observed missed identities and their dispositions. Full campaigns and exhaustive triage are not closure prerequisites; bounded smoke findings require triage.

### Investigation targets
- `model/design/gate-speed.md` and admitted fn-254.12 optimized test/count evidence.
- `model/umpire/front/src/`, `model/umpire/backends/native/src/` and the actual pinned mutation inventory.
- Current native supervisor/admission/count interfaces and retained candidate `work/fn2554-prep-mutants`; prepared source alone earns no execution evidence.

## Acceptance
- [ ] Committed config and native commands support framework production crates with documented exclusions; the fixed smoke manifest executes within authoritative work/time caps and records counts, identities, exact manifest and tool/source versions.
- [ ] Smoke findings are triaged; scheduled reports retain observed missed identities and dispositions, completed/remaining work and resume inputs without requiring full-campaign completion.
- [ ] Controls distinguish caught, missed, timed-out and unviable results, demonstrate capped partial results and preserve outcomes across failure/cancellation. Timeout never counts as caught.
- [ ] One fn-254.12 authoritative budget table supplies mutation work/time limits; command registration preserves the current native DAG, cancellation, resource admission and count authorities. Shared production/test fixes have explicit scope and exclusive ownership.

## Done summary
Blocked:
Execution and task closure wait for the committed fn-254-fast-validation-gates-and-a-real-inner.12 closure in umpire and admission of this task's work/time limits to its single authoritative budget table. Disjoint source preparation does not satisfy this gate. Flow task dependencies only accept same-spec IDs; after verifying that closure and admitted entries, the conductor may reset this block and admit the task normally.
## Evidence
- Commits:
- Tests:
- PRs:
