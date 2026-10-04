---
satisfies: [R1]
---
# fn-124-shrink-and-simplify-the-umpire-go.1 Delete the umpire0 and model0 archives and the empty umpire-gen-lean-api command

## Description
Implements R1 of the spec. Use `git rm -r` for tools/umpire0, model0 and tools/umpire/cmd/umpire-gen-lean-api (git rm is a git command, not a raw rm). Then fix every reference: common/testing/testpilot/dependency_boundary_test.go, tools/umpire/model/{ownership,isolation,nexus_close_baseline}_test.go, tools/umpire/lower/lower_test.go, Lean/model0 comments in lower/internal/producer/{producer,build,localize}.go and model/internal/checker/canonical.go (a comment that explains a rule keeps the rule and loses the citation), the producer's 'must match the Lean producer' constraint wording, and indexes or live docs that point at the archives (dated research may keep historical mentions). The untracked tools/gomad/ and tools/umpire3/ are owner files: leave them. Gates: module build and vet, full Go tooling suite, Testpilot tests, lint-code-fast, model gate.
## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
