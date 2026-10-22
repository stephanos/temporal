I’ll read the task spec first, then inspect the diff and relevant callers for contract and consistency issues.
The new definitions appear to preserve branch and payload scope. I’m checking whether downstream consumers still handle expressions that now arrive as local references, particularly the vector-bound validator.
I found a compatibility regression: a cached length such as `let n = self.values.len(); if n < 2 { push_back(...) }` loses its identity in the vector-bound validator and is now refused. The existing growth tests miss this case. Cached phased and vector tests passed; fresh Cargo verification was blocked by the read-only build lock.
- **Severity**: P2
- **Confidence**: 100
- **Classification**: introduced
- **File:Line**: `model/umpire/front/src/body/vector.rs:293`
- **R-IDs**: [R3]
- **Problem**: The new boolean-only summaries lose scalar length identities. A previously admitted handler containing `let n = self.values.len(); if n < 2 { self.values.push_back(7); }` now refuses with “requires an authored bounded length guard.” The initializer becomes `Local`, but comparison recognition still requires a literal `Length(Var(field))` operand. Before this change, local substitution preserved that shape. This contradicts the task’s representation-only compatibility contract, and the new boolean-sharing test misses it.
- **Suggestion**: Preserve memoized length identities when resolving comparison operands, without expanding definition trees. Add a regression covering the cached-length handler and an equivalent unbounded negative control.

Verification: cached phased/vector binaries passed all 11 tests without required TLC. Fresh Cargo verification was blocked by the read-only build lock. Committed evidence records a successful required-TLC full gate.

## Requirements coverage

| R-ID | Status | Evidence |
| --- | --- | --- |
| R1 | met | Existing World/start behavior preserved in golden comparison. |
| R2 | met | Existing input behavior preserved; payload-scope assertions updated. |
| R3 | partial | Linear-growth and compilation controls added; vector admission regresses above. |
| R4 | deferred | Activity retry checkpoint belongs to task .3. |
| R5 | deferred | Retired here; moved to fn-237. |
| R6 | deferred | Live admission work belongs to tasks .12/.5. |
| R7 | deferred | Final documentation belongs to task .6. |

Unaddressed R-IDs: []

Classification counts: 1 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":1,"pre_existing":0},"unaddressed":[]}
```

<verdict>NEEDS_WORK</verdict> - Fix the introduced vector-admission regression.