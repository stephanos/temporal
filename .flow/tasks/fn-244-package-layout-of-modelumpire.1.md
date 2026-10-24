---
satisfies: [R1, R3]
---
# fn-244-package-layout-of-modelumpire.1 Rust crates into role directories; umpire-tla; annotations merged into catalog; facade

## Description
One serialized freeze, after fn-238 and fn-242 close and before fn-237.1 starts (no other lane may edit model/umpire crates meanwhile). Move the Rust crates into the target layout in the spec: author/{umpire facade, dsl, dsl/macros, suite}, front, ir/derive, backends/{native, tla, tlc, apalache}, protocol/{proto, runtime}, catalog/{syntax, macros, build}, dev/{testing, fixtures}. Rename umpire-lift to umpire-tla and move the shared subprocess supervision into it so umpire-apalache drops its umpire-tlc dependency; merge umpire-annotations into umpire-catalog; add the facade and use it in the shared fixtures. Update every consumer (model/temporal, tools, Makefile, scripts, boundary scanner, guest build inputs), the crate map in model/README.md, Cargo descriptions, MIGRATE.md, AGENTS.md and the module map. Rebuild the guest; run slice and gaps. Goldens byte-identical except listed path-embedding snapshots; add a scoped leftover-name check.
## Acceptance
- [ ] Crates in the target directories; umpire-tla; annotations merged into catalog; apalache independent of tlc; facade used by the shared fixtures
- [ ] Crate map, descriptions, MIGRATE.md, AGENTS.md and module map updated; leftover-name check passes
- [ ] Goldens byte-identical except listed snapshots; guest rebuilt; default and required model gates, slice and gaps pass
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
