---
satisfies: [R4]
---
# fn-108-gomad-reduce-code-size-without-removing.3 Prepare the modernc memory adapter through the rewritten-module owner

## Description
Stage 1b (R4): express the modernc memory adapter through the existing rewritten-module owner instead of its private copy of the same preparation and anchored-rewrite loop. The adapter, its pinned version, both platforms and every digest stay.

**Size:** S
**Files:** `tools/gomad3/deterministicio/memory_adapter.go`, `tools/gomad3/deterministicio/memory_adapter_test.go`
**Touches:** [tools/gomad3/deterministicio/memory_adapter.go, tools/gomad3/deterministicio/memory_adapter_test.go]

### Approach

- Owner to reuse: `prepareRewrittenModule`, `rewrittenModule`, `sourceRewrite`, `anchorRewrite`, `rewriteAdapterSource` in `deterministicio/adapter_rewrite.go:17-136`. Pattern to follow: `deterministicio/sprig_adapter.go` (constants, a `[]sourceRewrite` variable, a one-call `prepare*` function).
- `prepareModerncMemory` (`memory_adapter.go:29-76`) and `rewriteModerncMemory` (`:78-110`) duplicate that owner step for step: identity check, module-cache resolution, original inventory pin, source read, anchored replace with exactly-once anchors, source/replacement digest pins, `copyAdapterModule`, replacement inventory pin, `BuildAdapter` evidence.
- Move the three anchor/replacement pairs verbatim (byte for byte, including the `//go:linkname` lines) into a `[]sourceRewrite` with `path: memoryMmapPath`, `sourceSHA256: memoryMmapSourceSHA256`, `replacementSHA256: memoryMmapReplacementSHA256`. Describe the module with `cacheElements: {"modernc.org", "memory@" + memoryVersion}`, `replacementDirectory: "modernc-memory"`, `preparedPackage: memoryModulePath`, and the existing inventory and prepared-source-set pins. Keep every constant and `memoryPreparedSourceSetSHA256` unchanged.
- `prepareModerncMemory` keeps its name and signature (registered at `deterministicio/profile.go:167`).
- `memory_adapter_test.go:27` and `:47` call `rewriteModerncMemory` directly. Retarget those two tests at `rewriteAdapterSource` with the memory rewrite spec so they keep asserting the same behaviour (only anonymous allocator mappings are modeled; a changed `mmap_unix.go` is rejected). Do not drop either test.

### Investigation targets

**Required:**
- `tools/gomad3/deterministicio/adapter_rewrite.go`
- `tools/gomad3/deterministicio/memory_adapter.go`, `memory_adapter_test.go`
- `tools/gomad3/deterministicio/sprig_adapter.go` — smallest existing user of the owner

**Optional:**
- `tools/gomad3/deterministicio/adapter_rewrite_test.go:105-290` — table tests over rewritten modules; add the memory adapter to those tables only if that removes a duplicated memory-specific test without losing an assertion

### Quick commands

```bash
cd tools/gomad3
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -tags test_dep ./deterministicio -run 'Memory|Rewrit|Profile'
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -tags test_dep ./deterministicio
make -C . validate
```

### Key context

- Equivalence pin: `TestPrepareModerncMemoryRecordsExactPrivateReplacement` and `TestModerncMemoryPreparedPackageSourceSetIdentity` compare the produced bytes, inventories and source-set identity with the pinned digests. They must pass with no pin edited. A pin that needs changing means the rewrite is no longer byte-identical: stop and report.
- Diagnostic wording changes from "modernc memory ..." to the owner's "modernc.org/memory ..." form, and the owner adds a regular-file check on the source. Both remain preparation failures. Confirm no test or doc asserts the old wording (planning found none) and note the change in the evidence.
- The linux/amd64 prepared-source-set pin cannot be exercised on this host; record it as not run.

### Standing constraints (every fn-108 task)

- The user owns commits: no `git commit`, `git add`, `git stash`, and no worktrees. Leave changes in the working tree and report the paths.
- No new dependencies (Go modules or external tools). `tools/gomad3` is a nested module pinned to go1.27.1; `tools/gomad3sim` and `tools/gomad3integration` belong to the root module.
- Preserve existing comments: keep them with the logic they describe when code moves, and delete a comment only together with the dead code it documents. Do not compress formatting.
- Public Go names/signatures/fields/defaults, CLI commands/flags/exit statuses, schemas, canonical bytes, `HostError.Reason` values and failure precedence stay unchanged (spec "API Contracts", R8).
- Host is darwin/arm64. linux/amd64 gates cannot run here: list them as "not run (no host)" in the evidence, never as passed.
- Focused tests run from `tools/gomad3` as `env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -tags test_dep <packages>`. `.toolchain/bin/go` is the patched toolchain; `make -C tools/gomad3 toolchain` rebuilds it (needs go.dev access). New test assertions use `require` with whole-value equality.
- Evidence (commands, platform, results, remaining failures) goes under `.flow/artifacts/fn-108-gomad-reduce-code-size-without-removing/`. A defect found on the way is recorded for its existing owner, not fixed here.

## Acceptance
- [ ] `prepareModerncMemory` delegates to `prepareRewrittenModule`; no memory-specific copy of the anchored-rewrite loop remains.
- [ ] All memory digests, version, sum, inventory pins and both platform source-set pins are textually unchanged; replacement bytes, inventory identities and `BuildAdapter` evidence match the pins on darwin/arm64.
- [ ] Changed version/sum, source drift, missing or duplicate anchors, a non-regular source file and a changed replacement inventory each still fail preparation; the two retargeted tests keep their original assertions.
- [ ] `./deterministicio` tests and `make -C tools/gomad3 validate` pass; the diagnostic-wording change and the unrun linux/amd64 pin are recorded.
- [ ] Net production lines in `memory_adapter.go` decrease; nothing staged or committed.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
