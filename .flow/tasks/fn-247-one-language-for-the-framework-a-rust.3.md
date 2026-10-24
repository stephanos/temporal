---
satisfies: [R4]
---
# fn-247-one-language-for-the-framework-a-rust.3 Python removed from prerequisites and docs; DESIGN records the one-language rule; model-time

## Description
After .2. Remove Python from `model/GUEST_BUILDS.md`, the Makefile preflight and `model/README.md`; add the `xtask` and `process` crates to the crate map; in `model/DESIGN.md` record the one-language rule (Rust for Model and framework, Go only where the host demands it), the `libc` promotion under R18 with the fn-211.3 precedent, and the supervisor's platform contract; paste `make model-time`. MILESTONES remains root-owned. Full gates: `make model-check`, `UMPIRE_BACKENDS=require make model-check`, `make model-time`.

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
