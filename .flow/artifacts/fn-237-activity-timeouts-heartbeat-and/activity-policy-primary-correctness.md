I’ll read the task contract first, then trace the changed activity policy and replay paths for correctness and edge cases.
The transition logic matches the approved deadline priority and cancellation policy. I’m now checking whether the fairness and budget tests prove the required behavior, and whether the compatibility projection preserves the old lifecycle edges.
The compatibility verifier confirms all 15 retained artifacts are unchanged. Both focused Rust test commands stopped before compilation because the sandbox cannot write Cargo’s build lock; I’m checking the source and committed execution receipts for the remaining evidence.
No surviving correctness findings.

The deadline priority, cancellation races, exact expiry facts, fairness obligations and retained lifecycle projection match the task contract.

Verification: all 15 original artifacts verified; all 863 protected files match both the manifest and review base. Both focused Rust test commands were blocked before compilation by the read-only Cargo build lock. Committed execution receipts record passing focused and required-backend checks.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R1 | met | Completed prerequisite support consumed; protected artifacts preserved. |
| R2 | met | Deadline/cancellation implementation, policy matrices, exact replay, fairness and budget controls. |
| R3 | met | Exact retained-graph comparison and changed-edge negative control. |
| R4 | deferred | Live admission and evidence belong to fn-237.4. |
| R5 | deferred | Final design write-up belongs to fn-237.5. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict> - Ready to merge (no blocking introduced findings, all R-IDs met or deferred)