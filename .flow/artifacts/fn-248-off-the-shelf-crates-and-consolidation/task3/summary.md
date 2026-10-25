Added allocation-free, exhaustive immediate-child iterators and consolidated the four recursion-only walks. Per-variant semantic lowering and the independent interpreter/compiled oracles are unchanged.

Traversal controls cover every variant, empty and four-element vectors, ordered mutation, immediate-only iteration, each infallible child position, all arithmetic operators, rename override stopping, rewrite replacement preorder, and Var-only idempotent priming with World/Input/Outcome/Local preservation.

baseline: green (IR 2 + TLA 4 library tests; interpreted 12 passed, 1 timing ignored).
verification: IR 4 + TLA 6 library tests; interpreted 13 passed, 1 timing ignored; required new-test selectors collected 2/1/1/1 passing tests; focused clippy with warnings denied and rustfmt check passed. See source-receipt.json for exact commands, logs, exits and final source hashes. Red precursor controls failed on the missing child APIs before implementation.

No snapshot diffs. Required TLC agreement and the covering R5 Model pipeline remain with joined task7; no canonical guest publication or full Model gate ran in this lane.

stage: impl-review - skipped(policy: parallel-wave - conductor owns the gate)
Tier: session (jev-unavailable(no_key))
