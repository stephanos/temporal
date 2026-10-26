---
satisfies: [R4]
---
# fn-257-extract-the-standalone-umpire.4 Prepare sibling Temporal consumers and forwarding commands

## Description
Wire actual server/system consumers to the exported interface; disjoint provisional source preparation may begin after task 1, but final admission requires task 3.

**Size:** M
**Files:** model/Cargo.toml/lock, root and nested Go modules/imports, model scripts/xtask/Make/CI/docs, committed guest ownership metadata.
**Touches:** [Makefile, go.mod, go.sum, .github/**, model/Cargo.*, model/scripts/**, model/dev/xtask/**, model/temporal/**, model/design/native-symbolic/**, common/testing/testpilot/**, tools/canary/**, tools/umpire/**, AGENTS.md, model/README.md]

### Approach
- Prepare in an owning task worktree, preserving all existing candidates. Follow model/Cargo.toml's shared path declarations and the existing external framework command bridge.
- Replace framework paths with exact sibling crate/schema/tool locations; preserve all actual raw/SDK consumer coverage, freshness and caller-context contracts.
- Use a parent go.work selecting server, generic host, Kettle and Temporal extension modules. Do not overwrite an existing workspace; retain unrelated entries.
- Update CI to fetch both recorded revisions in the same sibling layout. Missing source must yield setup guidance rather than network fallback.
- Migrate the retained runnable model/design/native-symbolic probe: its Cargo path dependencies, build.rs fixture path and src/main.rs path import must use the sibling too; preserving a deferred probe does not exempt it from source ownership cutover.
- Final admission also consumes task 7's cross-repository quick interface.
- Defer removal of in-tree generic source to task 6.

### Investigation targets
**Required:** model/Cargo.toml; model/temporal/hosts/go/go.mod; go.mod; model/dev/xtask/src/gate.rs; model/scripts/proto-go.sh; Makefile.

### Quick commands
When validation resumes, inspect cargo metadata manifest paths, go env GOWORK and go list -m -json; execute actual forwarding commands and scoped consumer gates.
## Acceptance
- [ ] Actual Cargo and Go consumers resolve the agreed sibling source paths and canonical identities.
- [ ] Existing model commands, guest freshness, setup failure handling and paired CI use the same interface.
- [ ] Source preparation preserves unrelated workspaces, candidates and all retained Temporal obligations.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
