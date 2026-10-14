I’ll read the task spec first, then trace the diff against the existing command and composition APIs. I’ll review directly and keep verification focused on this task.

The agreement check reads the emitted literal and member tables directly, and the Outage tests pin the expected backend result. I’m checking two integration boundaries next: how the proof handles declared start states, and whether its completeness claim covers actions removed from the emitted catalog.

The focused build passed. I also confirmed a gap in the completeness check: removing an enabled action, all its rows, and the states reached only through it still produces a kernel-accepted agreement theorem. The check uses the emitted catalog itself to decide which actions must be present.

The same gap reproduces on `workerOutage`: a literal reduced from six states and four actions to three polling-only states and two actions still passes `decide +kernel`. This is a completeness gap in the proof contract; the emitted table itself passes its tests. Lint could not finish because the read-only sandbox blocks its temporary inventory file.

- **Severity:** P1
- **Confidence:** 100
- **Classification:** introduced
- **File:Line:** `model/Umpire/Command/ComposeProofs.lean:228`
- **R-IDs:** [R5]
- **Problem:** Completeness uses `literal.actions` as the expected action universe:
  `literal.states.all (sourceComplete members literal.actions literal.rows)`.
  Consequently, dropping an enabled action, its rows, and any exclusively reachable states escapes the check. I reproduced this using `workerOutage`: removing `workerStop`, `workerResume`, and the stopped-worker states leaves three states and two actions, yet `composedTableAgrees` returns `true` and `ComposedAgreement.ofChecked (by decide +kernel)` succeeds. The theorem therefore cannot certify that the walk preserved every enabled declared action.
- **Suggestion:** Supply the declared action candidates independently of the walk’s emitted catalog, preserving action identity. Define reachability and completeness against those candidates, and permit catalog removal only after proving an action is disabled at every reachable source. Add a negative test that removes an entire enabled action.

Focused build passed for all three task targets. Conformance and lint checks were blocked by sandbox-denied temporary-file creation; the chained golden check did not run.

## Requirements coverage

Coverage below addresses task .3’s portions; the caller composition remains assigned to task .5.

| R-ID | Status | Evidence |
|---|---|---|
| R5 | partial | Kernel checks, refusal test, and axiom pins pass; completeness gap above remains. Timing evidence is still unrecorded. |
| R7 | met | Outage verification and `veil default` differential pin pass. |
| R8 | met | Six reachable states pinned, below the enumeration bound. |
| R14 | met | New Outage Query appears in the passing differential sweep. |

Unaddressed R-IDs: [R5]

Classification counts: 1 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":1,"pre_existing":0},"unaddressed":["R5"]}
```

<verdict>NEEDS_WORK</verdict>