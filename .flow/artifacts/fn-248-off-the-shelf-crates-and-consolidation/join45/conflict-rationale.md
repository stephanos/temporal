This is isolated merge preparation on baseline 956c017c32e0f009a475c9d0fa664b06f74d9e0a. No Flow lifecycle or review ran, and neither original source branch was changed.

Commit A, f9507adf06378fb9d4af80f6e7ebd88088da4673, merges original work/fn2484-macros first. The merge was conflict-free. Every entry in task4/source-sha256.json matches A, including the recorded manifests/lock; merge-a-source-comparison.json records the comparison. Existing task4 source-bound evidence is retained as such; no separate A-only new gate pass is claimed. Later combined checks exercise the integrated source and its baseline dependencies.

The second merge, from original work/fn2485-types, had conflicts in two files:

- catalog/build/src/wire_schema.rs: keep task4's quote/prettyplease generation, prost module sanitizer and canonical type naming. Resolve Node.rust by consulting task5's shared WKT primitive mapping first and retaining task4's type_name fallback otherwise. A wrapper descriptor's `.value` mutates `*parent` through quote tokens; ordinary fields retain member access. Nonprimitive WKTs use the native prost-types namespace.
- temporal/annotate/syntax/src/realization.rs: consult the same primitive mapping before task4's heck type-name fallback. The auto-merged native scalar presence access and prost-types imports remain. Darling parsing, explicit strict values, duplicate/source-span controls and source-ordered manual grammar families are retained without modification.

Three test-consumer corrections are required by the combined intents:

- task4's newly introduced wire_generated.rs imports descriptors through runtime::prost_types instead of the removed runtime::google namespace.
- API services.rs retains every method/action, request/response, literal and absence assertion, comparing lexed Rust tokens instead of old whitespace. Prettyplease's tuple trailing comma is represented explicitly; string literal spaces remain significant.
- API wire_schema.rs follows the canonical non-raw member spelling for a non-keyword field. Its old partial compact output assertion now checks the complete typed decode/mutate/encode arm. The long decode call's formatter-added trailing comma is represented explicitly. A proc-macro2 dev-dependency and its single lockfile dependency edge provide the Rust lexer; no registry package/version or guest runtime dependency is added.

The actual formatting was inspected in services-format-observation.log and wire-format-candidate.rs before updating the checks. These are generated-source observations from existing semantic test fixtures, not canonical artifacts. No .stderr or snapshot payload changed during this preparation. No global snapshot acceptance, wasm rebuild, canonical guest publication, full Model gate or all-Go suite ran.

The failed iterations remain recorded: missing native import (exit 101), obsolete service formatting pin (101), one locked invocation before refreshing the intentional API dev dependency edge (101), obsolete raw-member pin (101), and the full-arm pin before inspecting the long decode call's trailing comma (101). Each was corrected on the same coherent merged tree; the source branches remain intact.
