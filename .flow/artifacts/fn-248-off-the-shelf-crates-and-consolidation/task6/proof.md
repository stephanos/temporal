The top-level ITF envelope now uses serde derive with deny_unknown_fields. Required fields still use the strict Json visitor; optional params use a presence decoder so explicit null refuses. An object-root guard preserves refusal of positional arrays, while nested value/type validation, budgets and independent native replay are unchanged.

Baseline: green (14 controls; live backend disabled). The final affected-scope command passed 28 tests (17 controls, five decode, six replay), clippy passed with warnings denied, and package formatting passed. The new preservation controls also passed against the original parser. Deliberate null-collapse and positional-array mutants each failed the intended assertion (exit 101) and were restored; their logs are retained.

Two exact live selectors required checksum-verified Apalache 0.62.4: vector witness replay and payload/World/outcome replay. Both passed their existing tampering negatives. Their emitted TLA+, tool outputs, current ITF witnesses and replay reports are retained under live/. No snapshots changed.

R18: serde_derive 1.0.229 and syn 3.0.9 already exist in Cargo.lock; only umpire-apalache enables serde/derive, with no lockfile or package additions. Actual wasm32-wasip1 normal/build feature-edge resolution for all six guests is identical before/after; neither serde nor Apalache belongs to those guest closures. This is metadata evidence, not a wasm build proof. Host feature inventories and exact commands/exits/source hashes are in validation.json and its adjacent logs. Final7 owns joined guest builds/publication, required full validation and any central conflict rechecks.

Tier: session (jev-unavailable(no_key))
stage: impl-review - skipped(policy: parallel-wave - conductor owns the gate)
