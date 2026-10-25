I’ll read the task spec first, then trace the changed parsers and generators through their callers and focused test evidence.
The parser changes preserve the manual grammar boundaries and add duplicate-attribute controls. I’m now checking that the new name conversion stays consistent across prost-generated types, catalog inference, and Temporal realization.
The committed focused and UI logs pass, including the generated-setter compile/run control. I found a possible diagnostic gap in duplicate `request`/`response` handling and am checking whether real macro expansion points to the second occurrence, as the spec requires.
The duplicate-direction control points to the correct span. I did reproduce a naming regression: valid protobuf names such as `_42` and `Self_` lose the escaping prost needs, causing catalog inference to reject them or emit an invalid Rust path. The base implementation preserved those names.
- **Severity:** P2
- **Confidence:** 100
- **Classification:** introduced
- **File:Line:** `model/umpire/catalog/syntax/src/derive.rs:675`
- **R-IDs:** [R1]
- **Problem:** Case conversion drops prost’s required identifier sanitization. A valid protobuf message `_42` becomes `42`, causing `#[mirror(response = "example._42")]` to fail with `invalid protobuf name`. `Self_` becomes `Self`, producing E0433; package `self_` similarly becomes invalid `self`. These names worked with the base implementation. Temporal descriptor collection repeats the unsanitized type conversion at `realization.rs:477`, while the build generator handles these escapes.
- **Suggestion:** Apply consistent prost-compatible sanitization after case conversion across inference and descriptor collection. Add compile controls for numeric-prefixed types, `Self_`, and keyword-derived package names.

Existing parser controls pass. Direct macro compilation reproduced the failures above; successful compile controls could not finish because the sandbox forbids temporary-file writes.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R1 | partial | Darling and quote/prettyplease replacements implemented; naming regression above violates prost alignment. Remaining R1 work belongs to tasks .5/.6. |
| R2 | deferred | Owned by task .2. |
| R3 | deferred | Owned by task .3. |
| R4 | deferred | Owned by task .1. |
| R5 | deferred | Joined full validation belongs to task .7. |

Unaddressed R-IDs: []

Classification counts: 1 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":1,"pre_existing":0},"unaddressed":[]}
```

<verdict>NEEDS_WORK</verdict>