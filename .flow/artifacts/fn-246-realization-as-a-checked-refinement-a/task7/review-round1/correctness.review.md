I’ll read the task specification first, then trace the changed code and focused test evidence against its acceptance criteria.
The runtime changes mostly move existing behavior into capability traits. I’m checking the generated adapters for validation or dispatch changes, especially malformed evidence and callbacks that now use the shared filler.
The focused tests could not start because Cargo needs to create files and this sandbox is read-only. I’ve traced one potential contract violation: evidence admission refusals are converted to “no match,” which can publish Unexplained and freeze the run. I’m checking that path and finishing the adapter review.
- **Severity**: P2
- **Confidence**: 100
- **Classification**: introduced
- **File:Line**: `model/umpire/protocol/runtime/src/runtime.rs:310`
- **R-IDs**: [R2]
- **Problem**: `Err(_) => EvidenceMatch::Reject` discards evidence admission refusals. If snapshot projection returns Reach but typed evidence admission returns MalformedRequest or Unsupported, `reach()` returns `Ok(None)`. Judge then publishes Unexplained and commits a frozen run. Invalid evidence therefore becomes a model mismatch instead of a refusal, violating the task’s evidence validation contract.
- **Suggestion**: Preserve admission errors through the runtime’s Result path, keeping explicit predicate rejection separate. Add a control where projection succeeds and evidence admission fails; verify that Judge returns the original refusal without changing state, pending requests or retained history.

Focused verification could not run: Cargo failed before compilation because its build directory is read-only.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R1 | deferred | Singleton prefix controls retained; frontier and current-format integration belong to later tasks. |
| R2 | partial | Capability assembly, fillers and consumer migrations implemented; evidence refusal propagation fails as described above. |
| R3 | deferred | Catalog corrections included; shared-layer completion belongs to other epic tasks. |
| R4 | deferred | Probe accounting updated; binding replacement and live comparisons remain separately owned. |
| R5 | deferred | Protocol refinement is outside task .7. |
| R6 | deferred | Executor capability and strategy enumeration are separately owned. |
| R7 | deferred | Mapping measurements and authoring documentation are separately owned. |

Unaddressed R-IDs: [R2]

Classification counts: 1 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":1,"pre_existing":0},"unaddressed":["R2"]}
```

<verdict>NEEDS_WORK</verdict> - Preserve evidence admission refusals before publishing a judgment.