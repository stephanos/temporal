---
satisfies: [R8]
---
# fn-87-tighten-the-testpilot-protocol-glossary.1 Field-mapping equivalence harness over a descriptor snapshot

## Description
Build the oracle every later task leans on (R8, Edge Cases "No Verdict change", Early proof point): a checked-in snapshot of today's protocol descriptors and pre-migration fixtures, and a Go test that decodes each baseline fixture through the snapshot, maps it to the current protocol under a declared, named mapping, and compares it with the regenerated fixture. At this task the mapping is empty and every fixture must compare equal; each later task appends the steps its change declares. If the test cannot be written against a descriptor snapshot, stop and report (the spec's proof-point stop condition).

**Size:** M
**Files:** `common/testing/testpilot/internal/protocolmigration/{mapping.go,equivalence_test.go,mapping_test.go}` (new, test-only package), `common/testing/testpilot/internal/protocolmigration/testdata/baseline/{descriptors.binpb,fixtures/**}` (new), `common/testing/testpilot/internal/protocolmigration/README.md` (new, short: what the baseline is and how a task adds a step), `tools/umpire/internal/retiredvocabulary/{check.go,check_test.go}` (allowlist the baseline tree). No generator target: the snapshot is taken once, with the command recorded in the README.
**Touches:** [common/testing/testpilot/internal/protocolmigration/**, tools/umpire/internal/retiredvocabulary/check.go, tools/umpire/internal/retiredvocabulary/check_test.go]

### Approach
- Snapshot once at HEAD before any protocol edit: `mise exec -- protoc --include_imports --include_source_info --descriptor_set_out=.../descriptors.binpb -I proto/internal temporal/server/api/testpilot/v1/case.proto temporal/server/api/testpilot/v1/run.proto` (protoc 29.5, as `Makefile:620-626` pins). Record the command and the source commit SHA in the package README.
- Copy every pre-migration fixture byte-for-byte into `testdata/baseline/fixtures/`, mirroring repository-relative paths: the six `tests/testcore/testpilot/testdata/*-case.json`, the six conformance `case.json` and `expected.json`, and `correlated.json`.
- Decode baseline JSON with `dynamicpb` over a local `protoregistry.Files` built from the snapshot (`protodesc.NewFiles`), never `GlobalFiles` (the snapshot declares the same full names as the live package), with strict `protojson` (no DiscardUnknown). `Value.message_value` holds `google.protobuf.Any` payloads of Temporal API types, so the unmarshal resolver consults the snapshot types first and falls back to `protoregistry.GlobalTypes` for non-testpilot names. Per-file schema: `*-case.json` and conformance `case.json` are `Case`; `correlated.json` is a JSON array whose `case` and `runnableCase` are `Case` and whose `events[]` are `CorrelatedEvidence` (see `common/testing/testpilot/correlated_facade_test.go:22-50`); `expected.json` is the generator's own projection and the Verdict pin: it is compared as bytes, and a later task may change it only through a declared step over its JSON (for example localized rule ids in .14), never by regeneration alone.
- Mapping works on the generic JSON tree after strict old-descriptor validation: `Mapping` is an ordered list of `Step{Name, Requirement, Apply func(fixture string, tree any) (any, error)}`. Provide small helpers later tasks reuse: rename a message-typed key along a descriptor-known path, rename an enum literal, drop a field, rewrite a subtree. Steps name the R-ID they implement so the step list reads as the declared mapping.
- Compare: strict-decode the mapped tree into the generated `testpilotspb` types (`testpilot.DecodeCaseProtoJSON`, `common/testing/testpilot/case.go:11`), strict-decode the regenerated fixture the same way, and compare with `cmp.Diff(..., protocmp.Transform())` (go-cmp v0.7.0 is in `go.mod`). On difference, fail naming the fixture path and the first differing field path from the diff. Also require each baseline `expected.json` to equal its regenerated file byte-for-byte, and each `correlated.json` entry's `expected`/`incomplete` to be unchanged.
- One frozen snapshot and one mapping from it to the current protocol, edited by every structural task. Never re-snapshot mid-spec: chained snapshots would launder earlier drift. Non-rename steps must validate what they assume (a derived field equals its recomputed value, a dropped limit matches a declared loosened-bound entry, a removed guard equals the default expansion) instead of silently dropping data (ART-11 forbids ignore lists).
- Self-test (`mapping_test.go`): a deliberately mutated regenerated tree fails naming fixture and field; a step that errors fails naming the step; an unknown fixture file in either tree fails (the fixture set must match one-to-one, so a deleted or added fixture is not silently skipped).
- Retired vocabulary: the baseline spells names later tasks retire. Add an allowance for `common/testing/testpilot/internal/protocolmigration/testdata/baseline/` to `allowedNegativeFixture` (a path-prefix entry is acceptable because the tree is frozen and holds nothing else; list the rationale in a comment) (`tools/umpire/internal/retiredvocabulary/check.go:671`), with a check_test case proving a retired token outside that prefix still fails. Mapping source code spells old names by concatenation (`"Run" + "Status"`), the convention `check.go` itself uses, so no source file needs an allowance.

### Investigation targets
**Required** (read before coding):
- `common/testing/testpilot/case.go` — `DecodeCaseProtoJSON`, the strict decode to reuse
- `common/testing/testpilot/correlated_facade_test.go:20-50` — `correlated.json` shape
- `common/testing/testpilot/conformance_test.go:175-199` — descriptor closure walk pattern
- `tools/umpire/cmd/umpire-gen-case-runtime-conformance/generate.go:416,544-560` — which fixtures exist and where
- `tools/umpire/internal/retiredvocabulary/check.go:106-160,671-710` — scan and allowlist

**Optional:**
- `tests/testcore/testpilot/fixture_table_test.go:15-40` — fixture enumeration
- `Makefile:602-638` — how conformance and authoring checks are wired

### Key context
- `make umpire-check-regression` runs `go test ./common/testing/testpilot/...`, so the new package is gated without Makefile changes.
- The generated Go descriptors carry no `source_code_info`; the snapshot includes it so task .4's comment check can reuse the same protoc invocation pattern.
- Other sessions may commit concurrently; take the snapshot from a clean `git status` of the protocol and fixture trees.

## Acceptance
- [ ] `testdata/baseline/descriptors.binpb` and a byte-identical copy of every pre-migration fixture are checked in; the README records the protoc command and source SHA
- [ ] `go test -count=1 -tags test_dep ./common/testing/testpilot/internal/protocolmigration/` passes with an empty mapping: every Case, conformance Case and correlated entry decodes strictly through the snapshot and compares equal to the current fixture; every `expected.json` is byte-identical
- [ ] self-tests prove a field difference fails naming fixture and field, a failing step fails naming the step, and an added or missing fixture fails
- [ ] the retired-vocabulary gate allows retired tokens only under the baseline prefix (check_test proves both sides); `make umpire-check-retired-vocabulary` green
- [ ] `make lint-code` (after `go clean -cache`) shows no new issue in touched files versus the 161 baseline


## Done summary
Added the fn-87 equivalence oracle in `common/testing/testpilot/internal/protocolmigration`. It contains a frozen protoc 29.5 descriptor snapshot and byte copies of all 19 fixtures, both taken at d9c7577309. Every fixture decodes strictly through the snapshot, is mapped under `Declared` (empty today), and matches the regenerated fixture in the generated types, and every `expected.json` is byte-identical. The retired-vocabulary gate now allows retired tokens only under the baseline prefix.

- Self-tests in `mapping_test.go` fail for a differing field, naming the fixture and field path (Case, `expected.json`, and correlated rows). They also fail for a failing step, naming the step, and for an added or deleted fixture. `TestMappingHelpers` covers the four helpers: `RenameField`, `RenameEnumLiteral`, `DropField` (requires a check) and `RewriteMessages`.
- Decision: `Step.Apply` keeps the spec's `func(fixture string, tree any) (any, error)` signature. JSON objects in the tree are `*Object` values that carry the snapshot message name they encoded, so steps can target a message by its baseline name after earlier renames. This does not change the spec's scope.
- Decision: the repository `.gitignore` ignores `testdata/`, and `.gitignore` is outside this task's Touches. The baseline tree was committed with `git add -f`, and the README records this. Follow-up (reviewer P2): add `.gitignore` negation entries for this tree.
- Follow-up (reviewer P3): Any-payload annotation keeps the `google.protobuf.Any` name, and no test covers an external payload.
- Gates: the regression gate failed twice on unrelated live-test flakes before passing on the third run with 9 live identities. The first failure was a namespace-delete DeadlineExceeded in TestTestpilotUmpireRunRunsACheckedInCaseAgainstAnyEndpoint. The second was evidence ordering in TestTestpilotTypedNexusOperationsCase plus the same namespace-delete failure. The baseline run was green before any edit. lint-code shows 161 issues and lint-model 163 errors, both matching the inherited baselines.
- The commit range also contains 094108eac3 "wip", which another session committed with only `.flow`/`.plans` files.

stage: impl-review - ran (claude backend, SHIP on first round)
## Evidence
- Commits: 094108eac3aa7e90bf8d19c21dee73994ba0e846, ba4306b889a95a4224cb8088dd401cf9c366d9d8
- Tests: go test -count=1 -tags test_dep ./common/testing/testpilot/internal/protocolmigration/, go test -count=1 -tags test_dep ./tools/umpire/internal/retiredvocabulary/, make umpire-check-retired-vocabulary, CC=/usr/bin/cc TMPDIR=$(cd "${TMPDIR:-/tmp}" && pwd -P) make umpire-check-regression, go clean -cache && make lint-code GOLANGCI_LINT_FIX=false (161 issues, baseline 161), make lint-model (163 errors, baseline 163)
- PRs: