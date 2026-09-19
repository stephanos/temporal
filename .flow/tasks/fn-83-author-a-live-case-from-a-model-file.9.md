---
satisfies: [R9]
---
# fn-83-author-a-live-case-from-a-model-file.9 Store generated Testpilot JSON indented for review

## Description
Store every generated Testpilot JSON artifact indented, so a reviewer can read a Case or a conformance corpus change as a line diff. Today 13 checked-in files are single-line compact ProtoJSON: the six functional fixtures under `tests/testcore/testpilot/testdata/*-case.json`, the six `common/testing/testpilot/testdata/case-runtime-conformance/<class>/case.json`, and `correlated.json` (188 KB on one line). The `expected.json` files beside them are already written with `json.MarshalIndent(…, "", "  ")`.

**Size:** S
**Files:** `tools/umpire/cmd/umpire-gen-case-runtime-conformance/generate.go` (one persisted-form helper applied to every file it publishes), `generate_test.go`, the 13 regenerated fixtures, `tests/testcore/testpilot/README.md` (fixtures are canonical ProtoJSON *indented for review*), any test or doc that asserts renderer stdout equals fixture bytes
**Touches:** [tools/umpire/cmd/umpire-gen-case-runtime-conformance/**, tests/testcore/testpilot/testdata/**, common/testing/testpilot/testdata/case-runtime-conformance/**, tests/testcore/testpilot/README.md]

### Approach
- Format in the Go generator, the single writer of all 13 files. One helper runs `json.Indent(&buffer, rendered, "", "  ")` and appends exactly one trailing LF; apply it to functional fixtures, conformance `case.json`, and `correlated.json` before they enter the artifact map. `json.Indent` preserves key order and string escapes, and matches the `expected.json` layout.
- Keep `Testpilot.ProtoJSON.canonical` compact. It stays the single encoding policy (field names, enums, presence); indentation is presentation of the stored file. Do not use the protobuf library's `pretty := true`: it wraps at `lineWidth` 80, so a one-field change reflows whole blocks. Do not import `Umpire.Json` into `Testpilot`.
- Keep the determinism check on the renderer's compact bytes (render twice, compare), then indent.
- Staged validation (`validateArtifacts`, `validateFunctionalArtifacts`, `validateGeneratedArtifacts`) must reject a staged file that is valid JSON but not in persisted form, so a hand-edited or compact file fails `make umpire-check-case-runtime-conformance` with a message naming the file, not only the `diff -ru`.
- fn-83.3 may add a check that `umpire-case --render <id>` output equals the checked-in fixture. Where such a check exists (Go or Lean, test or acceptance wording in `.flow/tasks/fn-83-*.md`), compare after applying the same persisted form, or have the comparison normalize with `json.Indent`. Grep for it; do not assume.
- Every reader already decodes semantically (`testpilot.DecodeCaseProtoJSON` via `protojson`, `json.Unmarshal` for `correlated.json` and `expected.json`); hashes are over proto or binding bytes, never file bytes. Confirm no new byte-level reader landed in fn-83.3 before regenerating.

### Investigation targets
**Required:**
- `tools/umpire/cmd/umpire-gen-case-runtime-conformance/generate.go` — `runGeneration`, `renderStable`, `renderCorrelatedArtifacts`, `runFunctionalGeneration` (or its fn-83.3/fn-83.4 successor), the three validators
- `tools/umpire/cmd/umpire-gen-case-runtime-conformance/json.go` — `marshalExpected`, the existing indented form to match
- `Makefile` `umpire-gen-case-runtime-conformance` / `umpire-check-case-runtime-conformance`
- `common/testing/testpilot/case.go` — `DecodeCaseProtoJSON`, `PackCaseProtoJSON`

**Optional:**
- `model/Testpilot/ProtoJSON.lean` — the compact policy that stays
- `common/testing/testpilot/correlated_facade_test.go`, `internal/verification/correlated_test.go` — `correlated.json` readers

### Key context
- A parallel session owns fn-83.3's edits to the generator and fixtures; start only after fn-83.3 is committed, rebase onto it, and regenerate through `make umpire-gen-case-runtime-conformance` rather than hand-editing JSON.
- fn-83.4 rewrites the functional half of the generator to read `umpire-case --list`; it depends on this task so the two do not edit `generate.go` concurrently. The helper must be the one place both halves call.

## Acceptance
- [ ] All 13 generated Testpilot JSON files are indented with two spaces and end in exactly one LF; `wc -l` on each is greater than 1
- [ ] One generator helper produces the persisted form, and every published Case, conformance and correlated file goes through it
- [ ] A generator unit test pins the persisted form (key order kept, escapes untouched, trailing LF) and a staged-validation test rejects a compact but otherwise valid fixture, naming the file
- [ ] Any renderer-versus-fixture byte comparison from fn-83.3 compares against the persisted form
- [ ] `make umpire-gen-case-runtime-conformance` is idempotent; `make umpire-check-case-runtime-conformance`, `go test -tags test_dep ./tools/umpire/... ./tests/testcore/testpilot/... ./common/testing/testpilot/...` and `make umpire-check-live-tests` pass


## Done summary
All thirteen generated Testpilot JSON artifacts are stored indented.

- `tools/umpire/cmd/umpire-gen-case-runtime-conformance/json.go`: `persistedForm` compacts then
  re-indents with two spaces and appends exactly one LF. Compacting first makes the form idempotent
  whatever whitespace the input carried, and both passes keep key order and string escapes exactly.
  `requirePersistedForm` rejects a staged file that is valid JSON but stored otherwise, naming it.
- `generate.go`: every published artifact goes through the one helper -- conformance `case.json`,
  `correlated.json`, and the functional Case fixtures -- and all three validators
  (`validateArtifacts`, `validateFunctionalArtifacts`, `validateGeneratedArtifacts`) enforce it.
  The determinism check still compares the renderer's compact bytes before the file is stored;
  `Testpilot.ProtoJSON.canonical` is untouched and stays the single encoding policy.
- `generate_test.go`: the persisted form is pinned (key order kept, `A` and `\n` escapes
  untouched, `x/y` unescaped, idempotent, exactly one trailing LF), and a compact-but-valid staged
  fixture is rejected by filename.
- `tests/testcore/testpilot/README.md` records the stored form and its single writer.
- 13 regenerated files: 6 functional (`async-nexus` 1004 lines, `typed-nexus` 2025, `worker-outage`
  752, `typed-unary` 585, `get-system-info` 155, `synthetic` 148), 6 conformance `case.json`
  (122-153 lines each), and `correlated.json` (10924 lines, previously one line of 188 KB).

No renderer-versus-fixture byte comparison from .3 exists in code (it was a manual verification
recorded in that task's evidence), so nothing needed normalizing.

Verified: `make umpire-gen-case-runtime-conformance` is idempotent (second run leaves the tree
clean), `make umpire-check-case-runtime-conformance` passes,
`go test -tags test_dep ./tools/umpire/... ./tests/testcore/testpilot/... ./common/testing/testpilot/...`
is green, and `make umpire-check-live-tests` passes across 6 identities.

Review: SHIP after one NEEDS_WORK round. The P0 was real and mine: `go build
./tools/umpire/cmd/umpire-gen-case-runtime-conformance` drops its executable in the working
directory and `git add -A` swept a 10 MB binary into the first commit. It is removed and `/umpire-*`
is now ignored.
Pinned reviewer `claude:claude-fable-5-1:high` is account-limited for this session, so both rounds
ran on `claude:claude-sonnet-4-5:high` -- a same-family fallback, not an equivalent cross-family
review.

stage: impl-review - ran, 2 rounds (model: claude-sonnet-4-5, high; fable pinned but account-limited)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 4c7676d859, ba7d223eda
- Tests: make umpire-gen-case-runtime-conformance (idempotent), make umpire-check-case-runtime-conformance, CC=/usr/bin/cc go test -count=1 -tags test_dep ./tools/umpire/... ./tests/testcore/testpilot/... ./common/testing/testpilot/..., make umpire-check-live-tests (6 passing identities)
- PRs: