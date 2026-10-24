I’ll read the task specification first, then inspect the diff and focused verification evidence for correctness issues introduced by this change.
The Rust process module and Python supervisor moved byte-for-byte, and TLA’s compatibility reexport remains in place. I’m checking callers and path assumptions for effects of the move, then running the focused backend tests.
No blocking findings in this extraction checkpoint.

Verified both moved implementations are byte-identical to base, all 752 recorded artifact hashes match, compatibility reexports remain intact, and Cargo changes only add the local process crate and consumer edges. Recorded baseline/post corpus outputs match exactly.

The Darwin policy test passed. The focused Cargo command could not run because the sandbox prevented creating Cargo’s build lock. Committed logs record 159 backend tests and six Python tests passing before and after extraction.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R1 | partial | Ownership extraction and compatibility preserved; Rust supervisor explicitly deferred to task .4. |
| R2 | deferred | Tool migration assigned to later tasks. |
| R3 | deferred | Controls retained and mapped; Rust ports assigned to later tasks. |
| R4 | partial | Both ownership maps updated; remaining documentation and final measurements deferred to task .3. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict> - Ready to merge this extraction checkpoint.