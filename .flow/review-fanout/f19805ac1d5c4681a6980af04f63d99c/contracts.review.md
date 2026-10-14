I’ll read the task specification first, then check the diff against the implementation and focused verification commands.
The diff updates the backend documentation and adds the missing descriptor-generation step to both CI jobs. I’m checking the search-limit and replay claims against the code, and the new CI step’s build prerequisites.
No blocking findings. The updated backend, replay, deduplication, and witness-key documentation agrees with the implementation.

Verification attempted: plan-index check, inventory check, and the focused CI-workflow test. All were blocked by the read-only sandbox’s temporary-file restrictions.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R13 | deferred | R22 selected adoption; defer-only absence checks do not apply. |
| R20 | partial | Both jobs now generate the missing descriptor. A completed cold CI timing remains pending, explicitly recorded in `UMPIRE4_ORDER.md`. |
| R21 | partial | Delivery order records the successful 23-file rollback drill; the task evidence still lacks its diff stat. This paperwork gap is nonblocking under the review contract. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>