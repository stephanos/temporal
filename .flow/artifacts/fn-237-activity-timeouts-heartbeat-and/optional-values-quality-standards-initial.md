## Quality Audit — Standards axis: `e2f25dc292..58027c27b8`

### Summary

- Files changed: 92 · Findings: 4 · Should Fix: 4 · Consider: 0 · Blocking: none possible (standards axis).
- Evidence: **executed** Git blob line counts; **walked** declaration reuse, module boundaries and test organization. No builds/tests run.

### Should Fix

- **model/umpire/backends/native/src/interpret.rs:834** (Conf 100): File-size crossing rule: 974→1,022 lines increases interpreter maintenance surface — extract scalar import/conversion helpers into a private module.
- **model/umpire/backends/tla/src/emit.rs:954** (Conf 100): File-size crossing rule: 932→1,005 lines; inline optional tests push emission beyond the threshold — move that test module into a sibling test file.
- **model/umpire/dev/testing/src/backend.rs:414** (Conf 100): File-size crossing rule: 994→1,083 lines mixes backend comparison and extensive controls — move the existing test module into a separate file.
- **model/README.md:255** (Conf 100): File-size crossing rule: 996→1,014 lines enlarges the central guide — move a cohesive authoring section into a linked guide.

### Protected-path filter

- Protected-path filter: 2 size-only evidence candidates dropped; retain the baseline log and preservation manifest.

### What's Good

- **walked:** Optional admission reuses `declare::finite` and `declare::literal`; module dependencies and native validity documentation follow the approved contract.
- **executed:** All 827 preserved artifacts match their manifest at the range tip and working tree; all 15 original compatibility artifacts match their pinned originals and authoritative hashes.
