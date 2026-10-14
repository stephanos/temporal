# fn153 final rotation source audit

Read-only requirements audit at `1ec25122320bd0ec10e909136c0bfb604678f54b` found no remaining implementation gap in the bounded fn153.20/.21 requirements. Reviewer and implementers are from the same GPT family. No tests or qualification ran in this audit. Root subsequently corrected the duplicated article in SPEC.md and retired public-filesystem-facade wording in README.md. The historical inventory at fe84b745d7 and its receipts remain historical, not relabelled current evidence.

## Requirement disposition

Paths below are relative to tools/gomad3.

| Requirement | Current source |
| --- | --- |
| WASM cache transaction | internal/backends/wasm/backend/provider.go stages and validates both exact files, publishes module before receipt, joins cleanup failures and retains publication disposition. provider_test.go and backend/integration_test.go cover composition and interrupted rebuilding. |
| World raw syntax | internal/model/world/replay.go uses bounded shared strict decode without redundant raw UTF-8 preflight. Domain validation and exact wire comparison remain; library errors are deliberately wrapped by `decode World replay plan:`. |
| Ordinary retained JSON | backend/provider.go and cluster/retained.go strictly decode without producer-spelling equality. Raw path/length/SHA authentication precedes decoding; the streaming producer and semantic/identity/presence/allocation controls remain. |
| Four WASM readers | wasi/protocol.go, qualification/spec.go, installation/installation.go and cluster/oracles.go use the shared decoder/tokenizer with caller bounds and first-value precedence. |
| Numeric durations | internal/model/strictjson/decode.go uses pinned FormatDurationAsNano(true); tests cover signed bounds, types and null. |
| Campaign pin | internal/runner/internal/campaign/choice_exploration_start_test.go permanently reproduces the old hash by subtracting exactly 3 × 2 × 64 reserved bytes. The current fd169c… pin follows the World binary envelope; initial-state identity is unchanged. |

All five original whole-file publishers use internal/host/fs:

| Publisher | Owner and preserved contract |
| --- | --- |
| Qualification report | internal/evidence/qualification/qualification.go: unique staged-derived path, 0600, retained publication path after post-rename durability failure. |
| Soak report | internal/backends/native/qualification/soak/soak.go: Replace, indented JSON/newline, 0644. |
| Source archive | internal/backends/native/toolchain/source.go: bounded streamed/checksummed stage, cancellation and cleanup. |
| Regenerated patch | internal/backends/native/toolchain/patch_regenerate.go: exact staged-file validation before publication. |
| Stamp/launcher | internal/backends/native/toolchain/build.go: both staged first; stamp sync/failpoint before launcher sync/failpoint; 0644/0755. |

Shared error/disposition controls remain in internal/host/fs/replace_test.go. Other inspected Rename calls are directory transactions, guest operations, shared platform primitives or private portable-plan staging, not overlooked ordinary replacement publishers.

Strictjson embeds decoder/identity sources and binds dependency path/version/content sum, with module-pin tests. WASI composes it with its embedded sources, volume/network, interrupted-choice, broker clock/World and assertion detector identities and fails closed on a missing decoder identity. Record inclusion/exclusion, nil/empty and pointer controls remain in identity_semantics_test.go. Livecap host/overlay constants agree at `sha256:98fa946ce41d360d5441fba6061b158b94739ece37d1e03a9512899cadc8adb3`. Exact decoder ownership/purity and host-filesystem architecture edges remain tested. Source agreement alone is not generated-validation or qualification evidence.

## Measurements

| Selection | Production Go | Test Go |
| --- | --- | --- |
| Retired canonicaljson package | −845 | −893 |
| Current strictjson package | +219 | +273 |
| Package-only net | −626 | −620 |
| Whole interval 0785ac6a93..1ec2512232 | +12,412 / −8,888 = +3,524 | +18,220 / −6,439 = +11,781 |

Whole-interval Rust is +784 / −361 = +423. These totals include fn154/fn166 and other joined work, not attributable fn153 savings. The correction-worker receipt separately records production +68/−44 and tests +293/−46. Using identical historical exclusions, direct JSON inventory changes from 124 files/314 sites at fe84b745d7 to 124 files/317 sites at 1ec2512232. No canonicaljson Go references survive.

Root owns fresh source inventory capture, final source/review/generated receipts and frozen current-source WASM qualification. Supported-host requirements and explicit new native gates remain open under their recorded owners; no skip or historical receipt supplies acceptance. Tracker path/duplicate-section cleanup does not change product or acceptance status.
