# Private goroutine-field compaction

Source checkpoint for fn-110.2, based on `1b0bc277589d141aca8b534b03135ab3e57fc050`.
The original U3 comparator stays 32,652 bytes; the immediate candidate is
34,148 bytes. Qualification and R8 remain open until their actual gates pass.

Shorten only these private runtime `g` field declarations and their selectors:
`gomadIdentity` → `gomadID`, `gomadChildOrdinal` → `gomadChild`,
`gomadTimerOrdinal` → `gomadTimer`, `gomadSimulationDomain` → `gomadDomain`,
`gomadSimulationTransport` → `gomadSimIO`. Keep their complete order and types,
identity labels, ordinal arithmetic, hashing, transport accounting and comments.
The function/linkname `gomadSimulationDomain` and global
`gomadSimulationTransportSyscalls` retain their exact names.

Edit verified materialized upstream source with apply_patch, then regenerate
through the existing governed command. The overlay owns the corresponding
selector edits. Generated protocol identity mirrors and the choices-only
diagnostics identity fixture may refresh only as consequences of changed inputs;
preserve every other fixture field and all original tests/guards.

Before implementation, retain a failing measurement of unnecessary alignment
edits in otherwise unchanged upstream `g` fields, using real canonical outputs
and the existing pinned archive test helpers. Afterward retain U1/U3 counts,
zero-fuzz source equivalence, alpha-renaming plus pinned-gofmt equivalence across
all patched files and the overlay, unchanged field layout and exact allowlists,
both-source-set inventories, generation/validation, and focused regressions.
Use task-local additive measurement tests, leaving shipped tests unchanged.

Only one implementation lane may mutate runtime/patch/generated source or run
Go/cache commands. Parallel audit lanes are read-only or own disjoint task-local
evidence. The conductor owns Flow lifecycle, review, staging and commits. Preserve
the two unrelated `.turbo` files. Linux qualification stays with fn-128; stock
Linux/arm64 checks do not qualify Darwin or Linux/amd64. Keep raw evidence small,
reference previous receipts rather than copying them, and leave bulk scratch
materializations and source snapshots under ignored `.toolchain/fn-110/`.
