I’ll read the task specification first, then inspect the diff and focused validation evidence for contract mismatches.

The focused test evidence covers the parser changes and generated wire setters. I’m checking naming and descriptor paths next to see whether all three consumers apply prost’s naming rules consistently.

Fresh runs of both required Cargo commands are blocked by the sandbox’s read-only build lock. The committed validation record matches all 14 current source hashes and records both commands passing after integration. I’m using that evidence and checking the cached parser tests separately.

I found an introduced naming mismatch: catalog type inference strips underscores without applying prost’s identifier sanitizer. Valid protobuf names such as `Self_` and `_42Foo` now produce invalid Rust paths. The build generator handles both cases correctly, but catalog inference and realization paths do not.

- **Severity**: P2
- **Confidence**: 100
- **Classification**: introduced
- **File:Line**: `model/umpire/catalog/syntax/src/derive.rs:675`
- **R-IDs**: [R1]
- **Problem**: Case conversion omits prost’s identifier sanitization. A valid protobuf message named `Self_` becomes `Self`, producing E0433; `_42Foo` becomes `42Foo` and fails with “invalid protobuf name.” Both previously retained their correct Rust names. The same omission exists in `realization.rs:477`, while the build generator correctly handles these cases through `type_name`. Package conversion similarly changes `self_` into the invalid path component `self`.
- **Suggestion**: Apply consistent prost-compatible type and module sanitization across these consumers. Add compile controls for `Self_`, `_42Foo`, and package `self_`.

Fresh Cargo verification was blocked by the read-only build lock. Committed validation hashes match all 14 current source files and record passing focused/UI checks; 14 cached tests also passed. Cached macro probes reproduced the naming errors above.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R1 | partial | Darling and quote/prettyplease replacements implemented; prost naming contract fails as described above. Remaining crate replacements belong to tasks .5/.6. |
| R2 | deferred | Owned by task .2. |
| R3 | deferred | Owned by task .3. |
| R4 | deferred | Owned by task .1. |
| R5 | deferred | Joined full gate owned by task .7. |

Unaddressed R-IDs: [R1]

Classification counts: 1 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":1,"pre_existing":0},"unaddressed":["R1"]}
```

<verdict>NEEDS_WORK</verdict>