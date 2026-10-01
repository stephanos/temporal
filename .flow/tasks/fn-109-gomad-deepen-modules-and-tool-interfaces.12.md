---
satisfies: [R13]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.12 Separate detached Artifact references from owned opened handles

## Description
Stage 4, R13 (S1). One `artifact.Artifact` value serves both as a published/detached reference and as a live handle holding an `os.Root`, with an exported mutable `Manifest`. Split them so a detached value can never carry or alias a live resource. This is the second intentional public Go change of the spec.

**External ordering:** start only after the fn-108 R6/R7 tasks are done and verified: `fn-108-gomad-reduce-code-size-without-removing.5` (shared assessment, R6) and `.6` (retention and artifact-input composition, R7) in `tools/gomad3/runner`. Re-anchor the line references below against the post-fn-108 source first. flowctl cannot record a cross-spec task edge, so check `flowctl tasks --spec fn-108-gomad-reduce-code-size-without-removing` before `flowctl start`. fn-108 R7 owns artifact-input composition; this task changes the result/handle types only.

**Size:** M
**Files:** `tools/gomad3/artifact/{store.go,open.go,publication.go}` and tests, consumers in `runner/` and `qualification/set/execution.go`, `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/go-interface-changes.md`.
**Touches:** [tools/gomad3/artifact/**, tools/gomad3/runner/**, tools/gomad3/qualification/set/**, tools/gomad3/cmd/gomad/internal/cli/cli_test.go, .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/go-interface-changes.md]

### Approach
- Today: `type Artifact struct { Path; Manifest record.ExecutionRecord; StoredBytes; root *os.Root }` (`store.go:49-54`). `Store.PublishArtifact` (`store.go:65`) and `PublishArtifact` (`publication.go:33`) return it without a root; `OpenArtifact` (`open.go:19`) returns it with one; `Close` is on the pointer (`:70`), `Detached()` copies and drops the root (`:79`), and `OpenPayload` / `ReadPayload` / `CopyPayload` take the value (`:83`, `:191`, `:214`).
- Target shape: publication and snapshot return a detached reference (path, manifest copy, stored bytes); opening returns an owned handle (pointer) whose manifest is private, with payload access as methods or functions taking the handle. A snapshot deep-copies the manifest so mutating it cannot change the handle. Use-after-close returns an error.
- Append the exact declarations changed, consumers and migrations to `go-interface-changes.md` before editing. Consumers: `runner/replay_operation.go` (26 references, `preflight` `:421`, `simulationCapabilityForArtifact` `:284`, `choiceCapabilityForArtifact` `:355`, `readWorldPayloads` `:555`), `runner/runner.go`, `runner/minimize_operation.go`, `runner/inspect.go`, `runner/internal/corpus/corpus.go`, `qualification/set/execution.go`, `cmd/gomad/internal/cli/cli_test.go`.
- Payload access keeps validating inventory membership, mode, size and hashes against the pinned directory (`validateDirectory` `open.go:122`, `readValidatedFile` `:268`, `hashValidatedFile` `:293`, `openValidatedFile` `:311`); none of that validation may move to callers or be skipped for the handle.
- Publication semantics (staging, no-replace rename, manifest last, validated reuse) are untouched.

### Investigation targets
**Required:**
- `tools/gomad3/artifact/open.go` (whole file), `artifact/store.go:21-190`, `artifact/publication.go:16-145`
- `tools/gomad3/runner/replay_operation.go:79-130,279-360,421-570`
- `tools/gomad3/runner/internal/corpus/corpus.go` (artifact use)
- existing `tools/gomad3/artifact/*_test.go`

### Quick commands
```bash
cd tools/gomad3
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep ./artifact/...
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep ./runner/... ./qualification/... ./cmd/gomad/...
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep . -run 'TestRecordAndArtifactHaveSeparateOwners|TestPackageArchitecture'
```

### Constraints
- No `git add`, commit, stash or worktree: the user owns commits. Record `"commits": []` in the `flowctl done` evidence and say so in the summary.
- No new third-party dependency. `tools/gomad3/go.mod` requires only `golang.org/x/mod`, so testify is unavailable inside `tools/gomad3`: follow the existing `t.Fatalf` style with whole-value comparisons there. In the root module (`tools/gomad3sim`, `tools/gomad3integration`) use `require` with `Equal`/`EqualValues`.
- Preserve existing comments with their owning code, CLI grammar/defaults, canonical bytes for fixed supplied identities, and error precedence/classification.
- This host is `darwin/arm64`. `linux/amd64` gates cannot run here: list them as incomplete in the done summary, never claim them.
- fn-105 D12/D14 replay-divergence dispositions stay unchanged. Attribute a failure to those owners with retained evidence instead of relaxing an expectation.
- Run tests with `-tags test_dep`. Baseline the Quick commands before editing so a pre-existing failure is not attributed to this task.
- Evidence and decision records go under `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/`.
## Acceptance
- [ ] Detached Artifact references and owned opened handles are distinct types; a detached value holds no live resource and the opened manifest is private.
- [ ] Tests cover published and detached references, open/close, use after close, directory replacement under an open handle, snapshot mutation not affecting the handle, and valid payload access.
- [ ] Unlisted payloads, wrong mode/size/hash and symlink or path substitution remain rejected with the existing errors.
- [ ] All repository consumers are migrated and `go-interface-changes.md` records the declarations, consumers and migrations.
- [ ] Publication bytes, manifests and store identities for fixed inputs are unchanged.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
