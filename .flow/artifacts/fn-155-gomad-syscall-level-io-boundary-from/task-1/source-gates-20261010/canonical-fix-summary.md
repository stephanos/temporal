The canonical pinned-archive regression now passes after correcting only six patch index headers and regenerating the two choice-wire implementation digests. Source checks pass; the required original-base fast lint remains RED with the same 50 findings. Native acceptance remains open.

Task: fn-155-gomad-syscall-level-io-boundary-from.1
Status: in_progress
Tier: session (jev-unavailable(no_key)), explicit AGENTS preference retained.
stage: impl-review - skipped(policy: host-deferred - conductor owns the gate)

The existing regression failed before the fix at `patch_test.go:400`; its test, equality expectation and generator policy remain unchanged. The retained governed canonical scratch patch and the corrected checked patch both seal `2dca04787cefd5427b5d8e86169ca7f309133a0324b50a01fc3179a2ed746885`. The six index abbreviations changed from ten characters to seven; all source paths and hunks are identical. The retained checked-versus-canonical materialized trees compare equal over 15,618 files. Existing ambient Git abbreviation dependence remains outside this correction.

The protocol generator's seven choice inputs include the complete patch bytes. Regeneration therefore updates only the host and overlay codec constants to implementation digest `c4aded67729619fa86fdb41fd1e04fe2d869407ccbfde7154f0d7380ed34a93a`. The twelve live-capability inputs and their generated outputs remain sealed, including unchanged `runtime/gomad.go`. The changed identity requires a new matching patched build and identity-bound artifacts; equal source trees do not establish old-artifact compatibility.

All commands used pinned stock Go 1.27.1 and its native linux/arm64 compiler/gofmt, offline modules, CGO disabled and private `/tmp/fn155-canonical-fix-20261010.fhBo1dF7` for TMPDIR/GOTMPDIR. The complete environment, timeout-wrapped argv, numeric exits, elapsed times, logs and log seals are in [canonical-fix-evidence.json](canonical-fix-evidence.json) and `.flow/tmp/fn155-canonical-fix-20261010.7JxdKVdm/`.

| Check | Terminal observation |
| --- | --- |
| Protocol generation | Exit 0; 1.984 seconds; only corresponding codec digests changed |
| `make -C tools/gomad3 validate-toolchain SHELL=/bin/sh` | Exit 0; 3.128 seconds |
| Two exact pinned-archive tests | Exit 0; 67.273 seconds; regeneration PASS 28.43 seconds, context materialization PASS 38.19 seconds; zero skips |
| Exact patch policy/negative selection | Exit 0; thirteen named tests PASS; zero skips |
| Exact generation/host choice-wire selection | Exit 0; eight named tests PASS; zero skips |
| Original-base `make lint-code-fast` | Exit 2; 417.770 seconds; root and integration scopes report zero issues, Gomad reports 42 staticcheck and 8 forbidigo findings |
| Host source-set vet, darwin/arm64 | Exit 0; 0.406 seconds; two authorized host packages only |
| Host source-set vet, linux/amd64 | Exit 0; 0.407 seconds; same two packages only |

Lint used canonical base `951c5516e9e7b3066e7e069adda9565cfd68844c`, the retained lint/errortype tools and original-base classifier, with no task-only narrowing. Its ordered fifty three-line blocks seal `034b5959d8f6689fefbd9215234326d66b3809e204cf628c62b244e256398eea`, independently recomputed here. Root verified byte-for-byte equality with retained task-73 aggregate output and the unchanged original RED50 digest. No lint fix or test/assertion/timeout/policy change was made.

The two vet checks cover only `choice/internal/wire` and `internal/gomadtool/generation/protocol` source sets, executed on linux/arm64 with the qualified target GOOS/GOARCH values and `test_dep`. Their success is not full overlay/native static acceptance. Current-candidate patched-runtime clock/draw inventories remain unavailable because `.toolchain/build-key` is absent and the tests require a built GOROOT. No key was fabricated, no inventory skip was manufactured, and no native compiler/runtime or boundary-off/on gate ran. The retained full source-tree equality and unchanged overlay logic except the identity constant preserve a bounded relationship to prior source checks, not current native qualification.

The final 1,091-file product manifest matches the frozen candidate seal `f0c8797aa14d4ad543eb9880bcf66b179f6ae6d2d9d452c015e5978a2e6d384d`. Compared with the pre-fix manifest, exactly three rows changed. Tools, archive, descriptor, live-capability outputs and original RED/cleanup/diagnosis packet hashes still match. Root separately verified all 95 protected preexisting user/evidence paths, with zero mismatches, in `.flow/tmp/fn155-canonical-protected-check-20261010.json` (SHA-256 `9e1406599d3f164f7667687d092d3f660bd72c0e22de2e292579420f080488a6`). These are explicit bounded preservation sets, not a full-repository guarantee. Unchanged helper and cleanup suites were not rerun.

Root accepted the fresh [bounded source assessment](canonical-fix-source-assessment.md), sealed `1b3683fafb8c32533cc590cc5135a8a473f3ca600d9e123ca43da325014180e2`. It found no introduced issue in the exact three-file delta and independently recomputed identities and preservation. Requested writer/reviewer were the same model family; actual execution telemetry is unverified. This is source assessment only, not a formal implementation-review or SHIP verdict.

HEAD remains `a226b92b48f5a1580860f3f851afd7284d1851c9`, the index is empty, and the three-file candidate remains uncommitted for root. All owned commands are terminal, including foreground lint session 67226; the final process snapshot contains no attributable Go/compiler/gofmt/lint/generator process. No lifecycle, claim, native revival, PR/push or CI change occurred. Root owns subsequent acceptance and commit. fn-155.1 remains in_progress, and fn-155.2/.8 remain gated.
