---
satisfies: [R5]
---
# fn-108-gomad-reduce-code-size-without-removing.4 Publish upgrade dossiers through hostfs.Replace

## Description
Stage 1c (R5): publish the upgrade dossier through the shared host-filesystem replacement primitive instead of the private temp-file/rename sequence in the upgrade module.

**Size:** S
**Files:** `tools/gomad3/upgrade/upgrade.go`, `tools/gomad3/upgrade/upgrade_test.go`, `tools/gomad3/architecture_test.go`, `tools/gomad3/ARCHITECTURE.md`
**Touches:** [tools/gomad3/upgrade/**, tools/gomad3/architecture_test.go, tools/gomad3/ARCHITECTURE.md]

### Approach

- Replace the body of `publish` (`upgrade/upgrade.go:511-541`) after the `json.MarshalIndent(dossier, "", "  ")` + trailing `'\n'` step with one call to `hostfs.Replace(path, contents, 0o644)` (`internal/hostfs/replace.go:11`), wrapped as an upgrade-dossier publication error. Encoding, indentation, trailing newline, mode `0o644`, destination and parent-directory creation (`0o755`, done by the primitive) stay identical.
- The call site at `upgrade.go:226` runs before the gate-failure returns, so a failed gate still publishes; keep that ordering.
- `hostfs.Replace` is stronger than the current code: it syncs the file and the directory and reports a failed temp-file cleanup. Those surface as the same kind of returned infrastructure error from `Run`; dossier classification (`Qualified`, gate results) is untouched. Document this in the upgrade-dossier paragraph at `ARCHITECTURE.md:674-681` in one or two sentences.
- **Architecture rule:** `architecture_test.go:453` allows the `upgrade` owner to import only `qualification`, `toolchain`, `hostexec`. Importing `internal/hostfs` fails `TestPackageArchitecture` until `"hostfs"` is added to that list. Add exactly that one entry (every other host-effect owner already has it) and state the widened edge in the evidence; do not relax any other rule.
- Artifact no-replace publication (`artifact` package) keeps its own owner; do not route it through `hostfs.Replace`.

### Investigation targets

**Required:**
- `tools/gomad3/upgrade/upgrade.go:140-245`, `:511-541`
- `tools/gomad3/internal/hostfs/replace.go`, `replace_test.go`
- `tools/gomad3/architecture_test.go:429-466` (`ownerMayImport`), `:263-289`
- `tools/gomad3/upgrade/upgrade_test.go:25-165` — `TestRunPublishesCheckedUpgradeEvidence`, `TestRunPublishesUnqualifiedEvidenceWhenCorpusIsMissing`, `TestRunPublishesFailedGateEvidence`

### Quick commands

```bash
cd tools/gomad3
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -tags test_dep ./upgrade/... ./internal/hostfs ./cmd/gomadtool
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -tags test_dep . -run 'TestPackageArchitecture|TestExactModuleEdges|TestUpgradeOrchestrationIsAboveToolchain'
```

### Key context

Equivalence pin: add (or extend an existing upgrade test with) an assertion that the published file equals `json.MarshalIndent(dossier, "", "  ")` plus `"\n"`, has mode `0o644`, replaces a pre-existing dossier at the same path, and that a publication failure (for example an unwritable or file-blocked parent directory) returns an error while the prior complete dossier is still intact. The temp-file prefix changes from `.upgrade-dossier-*` to `.safefile-*`; confirm nothing (CI upload globs, `.gitignore`, tests) matches the old prefix.

### Standing constraints (every fn-108 task)

- The user owns commits: no `git commit`, `git add`, `git stash`, and no worktrees. Leave changes in the working tree and report the paths.
- No new dependencies (Go modules or external tools). `tools/gomad3` is a nested module pinned to go1.27.1; `tools/gomad3sim` and `tools/gomad3integration` belong to the root module.
- Preserve existing comments: keep them with the logic they describe when code moves, and delete a comment only together with the dead code it documents. Do not compress formatting.
- Public Go names/signatures/fields/defaults, CLI commands/flags/exit statuses, schemas, canonical bytes, `HostError.Reason` values and failure precedence stay unchanged (spec "API Contracts", R8).
- Host is darwin/arm64. linux/amd64 gates cannot run here: list them as "not run (no host)" in the evidence, never as passed.
- Focused tests run from `tools/gomad3` as `env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -tags test_dep <packages>`. `.toolchain/bin/go` is the patched toolchain; `make -C tools/gomad3 toolchain` rebuilds it (needs go.dev access). New test assertions use `require` with whole-value equality.
- Evidence (commands, platform, results, remaining failures) goes under `.flow/artifacts/fn-108-gomad-reduce-code-size-without-removing/`. A defect found on the way is recorded for its existing owner, not fixed here.

## Acceptance
- [ ] `publish` uses `hostfs.Replace`; no private temp-file/chmod/write/close/rename sequence remains in the upgrade module.
- [ ] Payload bytes (two-space indentation, trailing newline), mode `0o644`, destination path and replace-over-existing behaviour are asserted by test and unchanged; a failed gate still publishes the dossier.
- [ ] Write/close/rename/cleanup/sync failures return an infrastructure error and cannot leave a partial file in place of the prior dossier.
- [ ] `ownerMayImport` gains only the `upgrade` → `hostfs` edge; `TestPackageArchitecture` and the upgrade/hostfs/gomadtool tests pass.
- [ ] `ARCHITECTURE.md` notes the stronger cleanup/sync reporting; nothing staged or committed.


## Done summary
`upgrade.publish` now writes the dossier through `hostfs.Replace(path, contents, 0o644)`; the private temp-file/chmod/write/close/rename sequence is gone. Payload bytes, trailing newline, mode, destination, replace-over-existing and publication before the gate-failure return are unchanged and pinned by tests. Nothing is staged or committed.

- Errors are now `publish upgrade dossier: <hostfs stage>: ...`; the wrapped OS error is unchanged. File sync, directory sync and temp-file cleanup failures are newly reported (documented in one sentence in `tools/gomad3/ARCHITECTURE.md`). A directory-sync failure returns after the rename, with the complete new dossier in place.
- `ownerMayImport` gains only the `upgrade` -> `hostfs` edge; without it `TestPackageArchitecture` fails.
- Tests written before the switch and green against both publishers: golden bytes, mode and replacement (`TestPublishWritesFixedDossierBytes`), failed-gate replacement (`TestRunReplacesExistingDossierAfterFailedGate`), create/mkdir/rename failures with the prior dossier intact and the publication error ahead of the gate error (`TestRunReportsPublicationFailureAndKeepsPriorDossier`), and a real partial-write failure under `RLIMIT_FSIZE` in a child process (`TestRunKeepsPriorDossierWhenWriteFails`, unix). A direct `os.WriteFile` mutant fails all four.
- Not injected: close, file-sync, directory-sync and cleanup failures. They need a seam in `internal/hostfs`, which is outside this task's Touches; follow-up for the hostfs owner.
- Size: `upgrade/upgrade.go` 515 -> 496 code lines (production Go -19). `go doc -all` of `upgrade` is identical to the baseline.
- Gates on darwin/arm64, all exit 0: focused `go test` for `./upgrade/... ./internal/hostfs ./cmd/gomadtool`, the three architecture tests, `gofmt`, `go vet`, `make -C tools/gomad3 validate`. Not run: full `make -C tools/gomad3 test` (conductor instruction, fn-108.2 edits concurrently; `gate classify` reports FULL, no receipt written) and every linux/amd64 gate (no host; a `GOOS=linux` vet compile check passed).

Evidence: `.flow/artifacts/fn-108-gomad-reduce-code-size-without-removing/task4-evidence.md`, `task4-review.md`, `task4-working-tree.diff`.

baseline: green (focused Quick commands, before any edit)

stage: impl-review - ran (raw codex bridge on working-tree diff; commits forbidden) (model: gpt-5.6-sol) [round 1 SHIP, no findings]
stage: plan-sync - skipped(config: planSync.enabled != true)

GATE_SKIPPED lines: none.
## Evidence
- Commits:
- Tests: env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep ./upgrade/... ./internal/hostfs ./cmd/gomadtool, env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep . -run 'TestPackageArchitecture|TestExactModuleEdges|TestUpgradeOrchestrationIsAboveToolchain', GOWORK=off .toolchain/bin/go vet -tags test_dep ./upgrade/... ., GOOS=linux GOARCH=amd64 GOWORK=off .toolchain/bin/go vet -tags test_dep ./upgrade/ (compile check only), make -C tools/gomad3 validate, NOT RUN: make -C tools/gomad3 test (conductor instruction; concurrent fn-108.2 edits), NOT RUN: linux/amd64 gates (no host)
- PRs: