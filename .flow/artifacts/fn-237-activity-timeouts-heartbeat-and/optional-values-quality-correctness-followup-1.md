AXIS: correctness

## Quality Audit — Correctness follow-up: 58027c27b8..47cf0beea6

### Summary

- Files changed: 20 · Remaining findings: 1 · New regressions: 0 · Critical 0 · Should Fix 1 · Consider 0 · Ship: ⚠️ Fix first
- The coordinated assignment replacement is fixed; another form of the original CFG bypass remains.

### Should Fix

- **model/umpire/backends/tlc/src/module.rs:186** (Conf 75): Validation ignores constant continuation lines. Insert `__umpire_world_remaining = 2` immediately after `CONSTANT __umpire_world_remaining = 0` in the main CFG. `constant_bindings` retains only the original line, so binding and cross-family checks pass; TLC accepts the continuation and its scalar constant table keeps the later value. Because this parameter is unused, agreement and replay remain unchanged. Reject noncanonical CFG lines or parse complete constant sections before comparing assignments.

### Evidence

- **walked**: Rust filtering, decoding, cross-family checks and runner forwarding; TLC 1.7.4 bytecode in `ModelConfig.parse` and `SpecProcessor.makeConstantTable` establishes continuation parsing and last-assignment replacement.
- **cited**: Retained red/green logs support the coordinated replacement fix. No builds or tests ran during this audit.

### Regression Checks

- Extracted native conversion preserves its function body; moved test modules retain their assertions. Observed differences are formatting and redundant closure braces.
- Add a continuation-assignment mutation control. Full gates and primary review remain pending.
