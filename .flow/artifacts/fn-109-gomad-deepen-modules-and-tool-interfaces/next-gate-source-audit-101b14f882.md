# Patch regeneration cleanup correction

Recommend one R18/R19 correction in `tools/gomad3/toolchain/patch_regenerate.go`, with additive `patch_cleanup_test.go` regressions. Its seven unchecked cleanup sites belong to the public `RegeneratePatch` operation. Keep the fifteen remaining `build.go` sites for a separate owner. This receipt is source analysis at `101b14f882195422c31f35afc259cb25050b31e3`; no reproduction, Go command, lint command, native gate or source edit was performed by this scout.

The root's fresh baseline is `.flow/tmp/next-gate-101b14f882/base-full-lint.log`, SHA-256 `13d603ea5285e3092a8f5656f2675d829125cfa4e618af74c795b2df1c67fb0b`. It retains 273 findings, including 216 errcheck, 2 exhaustive, 11 forbidigo and 44 staticcheck, and exits 2 before errortype. Removing exactly these seven findings would leave 266; measure the actual delta. The requested research tier was gpt-6-astra at high; this scout cannot verify its actual model. The conductor retained its tier decision in `.flow/tmp/next-gate-101b14f882/scout-state.json`.

## Exact owner and lifetime

All line references below name the frozen source revision.

| Source | Existing resource lifetime | Correction boundary |
| --- | --- | --- |
| `patch_regenerate.go:68` | Regeneration work tree survives validation and output publication, then `RemoveAll` runs on return. | Capture a genuine cleanup error without undoing publication or removing an earlier primary error. |
| `patch_regenerate.go:121` | Candidate VERSION reader closes after return-value formatting in `validateCandidateVersion`. | Keep this defer boundary and report Close failure. |
| `patch_regenerate.go:255` | Candidate source reader closes after destination Close and copy-error formatting in `copyFile`. | Keep this defer boundary and report Close failure. |
| `patch_regenerate.go:282` | Output temporary pathname is removed after validation, git apply-check, rename and directory durability work. | Report Remove failure; suppress expected missing pathname only after successful rename. |
| `patch_regenerate.go:284,288,292` | Temporary file closes immediately on Chmod, Write or Sync failure, before contextual error formatting. | Check each Close exactly once, preserving close-before-format order. |

Use local named results and checked deferred cleanup, following `source.go:240` and `source.go:261`. On nil cleanup, return the existing primary error object unchanged. On a genuine additional error, retain the primary first and make both identities discoverable. Preserve `copyFile`'s existing `errors.Join(fmt.Errorf("copy patch candidate file: %w", copyErr), closeErr)` at line 263, including its current nil-copy-error formatting behavior. Correcting that behavior would expand this scope.

Publication remains `Chmod(0644)`, Write, Sync, Close, validatePatch, `git apply --cached --check`, Rename, directory Sync/Close (`patch_regenerate.go:283-311`). A cleanup or directory-durability error after Rename can accompany a published patch. Preserve that state and exact canonical patch bytes, one-context-line policy, descriptor/allowlist validation, output defaults and comments. Add no callback injection or cleanup framework.

## Meaningful public regressions

The existing `writeRegenerateFixture` and `writeRegenerateFixtureWithSource` at `patch_test.go:625-653` supply a synthetic checksummed archive and matching descriptor. They avoid the unavailable pinned source archive without changing repository pins. Existing tests invoke public `RegeneratePatch` at `patch_test.go:196` and preserve old output on rejection at line 310. Use the standard-library `testing` style because `tools/gomad3/go.mod` supplies only x/mod.

1. **Actual output-temporary Remove failure with cancellation.** Pass a test context wrapping `context.WithCancel`. Its Err method checks pathname state, not an invocation count. When the unique `.gomad3-patch-*` file exists in the test's output directory, rename it into a test-owned holding path, create a nonempty directory at its former pathname, then cancel the embedded context and return its Err. Before this point the real git/gofmt commands run normally. The callback is reached immediately before the final git apply-check through `patch.go:321`, `hostexec/command_unix.go:23-33` and `hostexec/command.go:62-65`; the temporary is already closed and has passed validation (`patch_regenerate.go:295-301`). BASE should return the contextual cancellation while dropping the real `ENOTEMPTY` from line 282. The corrected source must expose both identities with the primary first. Assert that old output bytes remain, the obstruction and its child remain, work cleanup completes, and the hook fired exactly once. Test cleanup removes only those explicitly owned paths. Establish this RED before production edits; the audit has not executed it.
2. **Unchanged-error control at the same boundary.** Cancel when that pathname appears without replacing it. Assert the same literal `verify regenerated patch against pristine source: context canceled` message, `errors.Is(context.Canceled)`, and the direct unwrap shape when cleanup succeeds. The output remains the prior bytes and the temporary disappears. Also retain a pre-cancelled context control and existing invalid-version/no-change/addition/deletion/prohibited-path controls.
3. **Work-tree cleanup failure where the host permits it.** A test context or configured Gofmt can make an explicitly owned work-tree directory unreadable/unwritable after extraction; a nonprivileged process should then observe actual `EACCES` on RemoveAll. Restore modes during test cleanup. UID 0 can bypass permissions, so a permission case must prove the syscall fault or explicitly skip. An alternative root-capable causal probe can rename the dedicated work-root parent to a holding path and replace its old pathname with a self-referential symlink at the final context callback, producing an ancestor `ELOOP` when RemoveAll traverses the original work path. Restore that test-owned parent before teardown. This remains a proposed probe, and its syscall/result must be observed before claiming coverage. Keep any otherwise-successful post-publication cleanup case separate from cancellation cases; no deterministic post-publication callback is exposed by this source.
4. **Healthy publication and retry controls.** Run deterministic exact patch generation twice, retain literal bytes/mode and materialization equivalence, ensure no temporary/work state remains, then repeat after the failed attempt using a fresh healthy context. Retain output-directory/rename refusal controls. A public normal success must remain nil despite the temporary pathname naturally disappearing after Rename.

The VERSION-reader Close, candidate-reader Close, three private temporary Close failures, and combinations involving them have no demonstrated fault realization in this receipt. Do not claim that handling their returns proves those faults executed. Context callbacks used by the regression must be confined to existing operation boundaries; a general interception hook, call-count trigger, asynchronous pathname race or privilege-dependent false pass is unnecessary.

## Commands and remaining acceptance

Run BASE controls and the additive regression before changing production, then freeze source for final checks. Serialize Go/cache work under the conductor. Suggested commands use the established pinned stock Go1.27.1 and `test_dep` contract.

```sh
go -C tools/gomad3 test -tags test_dep -count=1 ./toolchain -run 'Test(Regenerate|PatchCleanup|Validate|Materialize|Build)'
go -C tools/gomad3 vet -tags test_dep ./toolchain
go -C tools/gomad3 test -tags test_dep -count=1 . -run 'Test(PackageArchitecture|PureModulesHaveNoHostEffects|ExactModuleEdges|PublicPackagesDoNotExportTypeAliases|DomainModulesDoNotExportWireFraming|RunnerExecutionInjectionIsPrivate)$'
make -C tools/gomad3 validate
make lint-code-fast GOLANGCI_LINT_BASE_REV=101b14f882195422c31f35afc259cb25050b31e3 GOLANGCI_LINT_FIX=false
make --trace lint-code-gomad3 GOLANGCI_LINT_BASE_REV=951c5516e9e7b3066e7e069adda9565cfd68844c GOLANGCI_LINT_FIX=false
```

Retain unfiltered configured toolchain lint and direct `GOFLAGS=-tags=test_dep errortype -test=true ./toolchain` from the nested module with the installed binary. Existing pinned-archive tests may skip and remain unproved. Keep required native `GOFLAGS='-tags=test_dep -count=1' make -C tools/gomad3 test-host`, `make -C tools/gomad3 test-builder`, full `make -C tools/gomad3 test`, integration, functional smoke, core and affected qualification under their original owners.

`MILESTONES.md:33` permits further source progress after an integrated reviewed predecessor while acceptance stays open. Task47 supplies that predecessor; a separate corrective owner supplies this seven-site change. Task21 (`.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.21.md:9`) consumes evidence and implements nothing. R18/R19 (`.flow/specs/fn-109-gomad-deepen-modules-and-tool-interfaces.md:818-819`) preserve errors, transaction guarantees, comments, fixed-identity bytes and all original evidence requirements. Fn-110.4 retains canonical regeneration and native source-archive qualification ownership; fn-109.10 retains installation ownership. Keep the original formal review, native Darwin/full/affected, matched-first-baseline, 10/100 bounded measurements and predecessor requirements open wherever unproved. Fn-128 remains deferred and native Linux remains unverified.
