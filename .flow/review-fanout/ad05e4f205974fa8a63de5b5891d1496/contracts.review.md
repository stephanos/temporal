I’ll read the task specification first, then review the diff and related code for contract and consistency issues.
The diff adds the per-composition theorem and pins the Outage query on both backends. I’m checking whether the theorem independently validates completeness, including rows or actions the generated table might omit.
The three focused Lean targets pass. I found a possible gap: completeness checks only actions retained in the emitted catalog, so deleting an action and all its rows may still pass. I’m testing that case directly. Lint is blocked by the sandbox’s read-only filesystem.
- **Severity:** P2
- **Confidence:** 100
- **Classification:** introduced
- **File:Line:** `model/Umpire/Command/ComposeProofs.lean:228`
- **R-IDs:** [R5]
- **Problem:** Completeness trusts the emitted action catalog: `sourceComplete members literal.actions literal.rows`. Removing an enabled action together with all its rows therefore removes the obligation to check that action. I reproduced this using `workerOutage`: deleting `workerResume` from the indexed catalog and deleting its rows still passes `composedTableAgrees` and proves `ComposedAgreement.ofChecked` using `decide +kernel`. Consequently, the check cannot fulfill the promised rejection when the walk omits an enabled action.
- **Suggestion:** Supply the declared action candidates independently of the emitted literal, and check completeness and reachability against those candidates. Add a negative test deleting an entire enabled action and its rows.

The three focused Lean targets passed. The counterexample also compiled successfully. Conformance checking and lint stopped at temporary-file creation because the sandbox is read-only; the chained golden check consequently did not run.

## Requirements coverage

Scoped to task .3; the `nexusCaller` portions belong to later work.

| R-ID | Status | Evidence |
|---|---|---|
| R5 | partial | Kernel theorem, axiom pins, and refusal test pass, but completeness has the omission gap above. |
| R7 | met | Outage query pins `verified-within-limits`; both backends agree with `veil default`. |
| R8 | met | Six reachable states pinned; 6 × 4 actions is below 16,384. |
| R14 | met | New Outage query appears in the passing differential sweep. |

Unaddressed R-IDs: [R5]

Classification counts: 1 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":1,"pre_existing":0},"unaddressed":["R5"]}
```

<verdict>NEEDS_WORK</verdict>