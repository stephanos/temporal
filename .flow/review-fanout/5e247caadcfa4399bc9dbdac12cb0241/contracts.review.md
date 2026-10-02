I’ll read the task specification first, then review the diff for agreement between the code, documentation, and stated contracts.
The runtime diff extracts the same instance check into a shared helper. The main change is documentation; I’m checking its claims about instance types, expansion, and Producer folding against the implementation. I’m also reviewing the fn-94 planning files included in this range.
The focused tests could not start because the read-only sandbox prevents Go from creating its build directory. Source inspection so far supports the helper extraction and Rule-instance docs. I’m finishing checks of the Producer’s folding exceptions and requirement coverage.
No blocking introduced findings. The shared helper preserves both callers’ behavior, and the Rule-instance documentation agrees with the admission, evaluation, and Producer code.

Focused tests could not start: the read-only sandbox prevented Go from creating its build directory. No test pass is claimed.

## Requirements coverage

Coverage is scoped to this task’s R9 and R10.

| R-ID | Status | Evidence |
|---|---|---|
| R9 | partial | GOV-02 drafts preserve approved text; listed documentation updated. ORDER delivery entry remains pending close. |
| R10 | partial | Instance-value conformance entry and expanded-size unit coverage exist. Closing gates could not be verified here. |

The pending close record is process-only and does not block this verdict.

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>