---
satisfies: [R13]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.12 Separate detached Artifact references from owned opened handles

## Description

Owner amendment (2026-10-04): this task transfers every remaining native Linux execution, Linux pack/report/replay and Linux-specific qualification-documentation requirement to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). Native execution/full/affected gates still owned here apply to Darwin. Missing transferred Linux proof cannot block this task. Static coverage of both supported source sets, shared implementation, preservation, review and other non-Linux requirements remain unchanged. Retained scope: Implementation, both-source-set static coverage, R18 preservation, admission dependencies, lint, formal review and Darwin/full/affected gates. See the [transfer manifest](../artifacts/linux-scope-transfer-2026-10-04.md). Historical progress below retains its original meaning and is not current-candidate proof.

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

Current native-execution acceptance is Darwin-only here. The corresponding Linux clauses and any older missing-Linux completion rule are transferred to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). All other acceptance below remains in force.

- [ ] Detached Artifact references and owned opened handles are distinct types; a detached value holds no live resource and the opened manifest is private.
- [ ] Tests cover published and detached references, open/close, use after close, directory replacement under an open handle, snapshot mutation not affecting the handle, and valid payload access.
- [ ] Unlisted payloads, wrong mode/size/hash and symlink or path substitution remain rejected with the existing errors.
- [ ] All repository consumers are migrated and `go-interface-changes.md` records the declarations, consumers and migrations.
- [ ] Publication bytes, manifests and store identities for fixed inputs are unchanged.

## Done summary
Blocked:
Blocked: R13 is implemented and reviewed (SHIP, two rounds). Only native darwin/arm64 and linux/amd64 gates remain, and they belong to task 21.

Done (commits 3a3c31b0a7 and 6a52807659 on gomad-fn109, base 48c95c0c97):
- `artifact.Artifact` is now only a detached reference. It loses its private `root`, keeps the same exported fields, and has no Close.
- `OpenArtifact` returns an owned `*artifact.Opened`. Its path, manifest, stored bytes and pinned root are private. `Close` is idempotent and nil-safe.
- `Manifest()` and `Snapshot()` (which replaces `Detached()`) return deep copies through the reflective `cloneManifest`.
- `OpenPayload`, `ReadPayload`, `CopyPayload` and `TargetSharing` (which replaces `TargetSharingOf`) are now handle methods. Their bodies, check order and error texts are unchanged. Use after close fails with `artifact is not open`.
- All consumers are migrated: runner replay, minimize, inspect, resume, corpus, campaign retained evidence and qualification/set. The declarations, consumers and migrations are recorded in go-interface-changes.md.
- Migration fix: `runner.preflight` never closed a rejected artifact's root, because its deferred close saw the zeroed named result. It now closes it.

Local evidence (linux/arm64, developmental only, with the uncommitted shim; details in task-12/local-evidence.json):
- Fixed-input publication bytes, manifests and store identities are byte-identical to base across 12 store configurations, checked with a scratch test.
- Eight new artifact tests cover published and detached references, open/close, use after close, directory replacement, snapshot isolation, valid access, and the rejection matrix (unlisted, bound, mode, size, hash, symlink, escape). A mutation check showed they catch a shallow snapshot and a close that keeps the root.
- Quick commands: exit 1 at baseline (122s, 21 failures) and after (133s). The run after has the same 21 failures plus 4 timing tests that pass on an isolated rerun.
- Architecture tests: exit 0. go vet and gofmt: clean.
- Full `make test-host`: exit 2 (244s). Every failure is in the baseline or in the task-11 shim/pin-drift set, in packages that do not depend on artifact, apart from one watchdog timing test that passes when rerun alone.
- golangci-lint: not run, because the repository binary is a darwin build.

Remaining native gates: on darwin/arm64 and linux/amd64 hosts (task 21), full `make -C tools/gomad3 test-host`, the task Quick commands and scoped golangci-lint.
## Evidence
- Commits:
- Tests:
- PRs:

## Linux ownership blocker (2026-10-04)

Linux ownership amendment (2026-10-04): all native Linux execution obligations moved to fn-128. Missing transferred Linux evidence no longer blocks this task. Source-owned acceptance remains incomplete for Implementation, both-source-set static coverage, R18 preservation, admission dependencies, lint, formal review and Darwin/full/affected gates. Keep the task blocked for those independent requirements, with current-source evidence required by its original acceptance. See the scoped Description/Acceptance and .flow/artifacts/linux-scope-transfer-2026-10-04.md.
