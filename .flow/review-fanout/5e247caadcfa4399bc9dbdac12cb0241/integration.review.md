I’ll read the task spec first, then inspect the diff and trace how the changed runtime code interacts with its callers.

The runtime change extracts the same rule-instance check into one shared helper; both callers still invoke it at the same point. I’m checking the documentation against the existing preparation and evaluation paths, and reading the future-refactor plans included in this range.

The focused tests could not start because the read-only sandbox prevents Go from creating its build directory. Code inspection still shows the helper preserves both callers’ behavior, and the terminology changes match the task’s Rule/Rule-instance distinction.

No blocking findings. `ir.HasRuleInstances` preserves both callers’ predicates and validation order. The documentation matches the existing preparation, evaluation, and Producer paths.

Focused tests could not start: the read-only sandbox denied Go’s build-directory creation. Closing gates were not rerun.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R9 | partial | GOV-02 drafts and Rule-instance documentation are present; ORDER delivery bookkeeping remains for closure. |
| R10 | partial | Instance-value rejection fixture exists; gate success could not be independently verified here. |

These remaining closure and verification items are nonblocking under the supplied review rubric.

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>