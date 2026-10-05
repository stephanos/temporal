# Source-inventory formatted write review

SOURCE_PROGRESS_COMMIT. The frozen one-line correction preserves the
source-inventory bytes and is ready for a source-progress commit with root's
terminal integrated lint evidence retained. No introduced P1, P2 or P3 finding
was identified. Original task11 qualification remains open. This fresh
independent review uses the same Sol model family as the writer, intentionally,
and supplies no formal SHIP/DONE verdict.

The reviewed admission HEAD is `4695a9ad18de1aa49e032dad82154f73635e9c8d` on
`gomad`. FINAL `inventory.go` SHA-256 is
`5df9e8cbf6f11f90ee804acf502b8b7dca45f83fdebcb88da9f4992253da53f4`.
The selected 1,052-path FINAL fingerprint is
`8541e2f5e20623ba38928ce13b9312b717ac39d7d7b9de57614e5f898cb9e134`.
The reviewer independently reconstructed FINAL from immutable Git BASE with
exactly the admitted replacement at `inventory.go:82`. The source diff contains
no other selected tracked change. Tests, comments, imports, signatures, domain
prefix, path/NUL framing, traversal, bounds, refusal order, capacity mappings,
consumer pins, APIs and generated output retain their HEAD bytes.

`sha256.Sum256(contents)` supplies a concrete `[32]byte` value with no custom
formatting methods. The retained Go 1.27.1 `fmt/print.go` formats a byte array
under `%x` as the bytes' lowercase hexadecimal string. Both `Sprintf` and
`Fprintf` call the same `doPrintf` formatter. The old string-to-byte conversion
preserves that buffer's bytes; the new call passes the formatted buffer directly
to one `Write`. Consequently both write the same 71 bytes, `sha256:` followed
by 64 hex digits, between the unchanged path and trailing NUL writes.
`sha256.New()` supplies the hash writer directly, without an injected writer
or callback. Its retained implementation returns the complete input length
with nil error, consistent with `hash.Hash`'s infallible-write contract.
Keeping `_, _` therefore preserves this concrete operation's error behavior.
This reasoning claims neither arbitrary-writer equivalence nor measured
performance.

The reviewer independently rebuilt the existing four-file fixture's canonical
stream with Node's SHA-256 implementation and obtained the unchanged literal
`sha256:624ffd10d3b0e4126993be4d4c60de5dba62a7bc08df9a1b9f4db1a0260c07b3`.
The worker's [handover](handover.md), [evidence](evidence.json), source freeze
and all 15 worker receipts were checked after the worker released its source
and cache lane. Every retained stdout/stderr hash matches its receipt, all
commands are terminal, before/after source fingerprints match their stage,
and current selected files and executable hashes match FINAL. The reviewer
also compared BASE/FINAL terminal test-event multisets independently.

BASE and FINAL inventory controls pass 8/8 and target controls pass 10/10,
including canonical reviews and typed capacity mapping. The adapter controls
have the same 1 pass and 2 failures at both stages. Both failing tests stop in
`pinnedModuleCache` because `.toolchain/bin/go` is absent, before inventory
hashing or pinned-module comparison. Complete module and rewritten-adapter
inventory proof therefore remains unavailable. Actual unfiltered package lint
changes from exactly one QF1012 to `0 issues.` with fix disabled. Architecture,
purity and exact edges pass 3/3; standalone errortype, formatting, exact
preservation and check-only validation pass. Generator input ownership excludes
the changed host file, and validation retains the source fingerprint. The
profile-check output is ordinary developmental-host evidence.

Root's [integrated lint receipt](root-integrated-lint/receipt.json) records
terminal Make exit 2, null signal and 99.518 seconds against stable FINAL inputs.
The actual gate covers 55 host packages with the original comparison revision
`951c5516e9e7b3066e7e069adda9565cfd68844c`, original configuration and fix disabled.
The reviewer independently checked all four root receipt/log hashes and stage
bindings, then compared the previous task9 integrated output with the complete
current diagnostic blocks. Exactly the inventory QF1012 block disappeared.
All 318 surviving blocks retain identical bytes and order, with zero introduced
blocks. Their measured counts are 252 errcheck, 3 exhaustive, 11 forbidigo and
52 staticcheck. The [delta receipt](root-lint-delta/receipt.json) agrees with
this independent 319-to-318 comparison. The integrated errortype stage remains
UNREACHED after golangci-lint failure. Standalone errortype success supplies only
its scoped proof. Root's fresh inventory recheck passes 8/8 without skips, and
its fresh preservation check passes; both retain stable FINAL inputs. Integrated
lint remains red, with its residual findings and original qualification open.

The execution host is stock Go 1.27.1 on developmental linux/arm64. Original
task10 admission dependency, task11/R17 acceptance, R18 preservation, matched
first-baseline, full/default/functional/affected-consumer/formal/native Darwin
requirements remain open wherever unproved. Static inspection of both supported
source sets does not supply native execution. Linux execution remains deferred,
unverified and nonblocking under fn-128. The reviewer ran no Go/cache/lint
commands and changed only this new review artifact.
