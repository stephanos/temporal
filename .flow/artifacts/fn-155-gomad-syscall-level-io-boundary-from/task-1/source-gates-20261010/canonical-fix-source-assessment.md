# fn-155.1 canonical correction source assessment

No introduced issue was found in the bounded three-file source delta. The frozen correction removes the retained canonical-regeneration discrepancy while preserving every patch path and source hunk. This independent assessment supplies source-review evidence only. Root owns integration and acceptance.

The reviewer used a fresh context, read `AGENTS.md`, `tools/gomad3/README.md`, the operative `MILESTONES.md`, and fn-155.1's current task contract. Requested reviewer tier was `gpt-6.1-sol` at high effort. Dispatch reported `Tier: session (jev-unavailable(no_key)), explicit AGENTS preference retained`; requested model identity is not execution telemetry. Writer and reviewer were requested from the same model family.

## Candidate and preservation

Base HEAD remains `a226b92b48f5a1580860f3f851afd7284d1851c9`. The assessment covers only these frozen product files against HEAD.

| File | SHA-256 |
| --- | --- |
| `tools/gomad3/toolchain/runtime/go1.27.1.patch` | `2dca04787cefd5427b5d8e86169ca7f309133a0324b50a01fc3179a2ed746885` |
| `tools/gomad3/choice/internal/wire/wire_generated.go` | `b3fa8d8317d53c95628c4e49f91bbb0c0ad001c0bf061d381cca014ac2c2346a` |
| `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadchoicewire/wire_generated.go` | `8a84253cce6e91b18f8d428de0a32915fdfdcf1e96e569a96855977fce976bce` |

The original 1,091-file product manifest is `.flow/tmp/fn155-source-gates-20261010.7uEg0rKW/product-before.sha256`, sealed `496762e45e3d07cdd884ff507e655604233644dff189f58bf350018106e69ca9`. The frozen manifest is `.flow/tmp/fn155-canonical-fix-20261010.7JxdKVdm/product-frozen.sha256`, sealed `f0c8797aa14d4ad543eb9880bcf66b179f6ae6d2d9d452c015e5978a2e6d384d`.

Independent `sha256sum -c --quiet` verification of the frozen manifest exited 0. Comparing both manifests shows exactly the three rows above changed. `git diff --name-only HEAD -- tools/gomad3 cmd/tools/lintcode` lists exactly those three files; `git diff --cached --name-only` was empty. Descriptor, live-capability generated outputs, `runtime/gomad.go`, tests, assertions, baselines, generator implementation, templates, lint configuration and policy retain their original product bytes. Unrelated existing workspace changes remain outside this assessment.

## Attempt to refute the canonical explanation

The retained initial failure is the exact-byte comparison at `toolchain/patch_test.go:400`, documented by `worker-summary.md` and `evidence.json` in this directory. Initial regeneration failed while context-representation materialization and all six cleanup cases passed. The diagnostic driver invokes the existing `ExtractSource`, `MaterializePatch` and `RegeneratePatch` operations with private output and work paths.

The retained `checked-vs-canonical.diff` is sealed `fffcf6c4cfb5182d83545b9d121984257c01d3846f632b61e2d6dcd1d6fe7e81`. Its six changes shorten 10-character Git object abbreviations to their existing seven-character prefixes for `runtime/netpoll.go`, `syscall/syscall_darwin.go`, `syscall/syscall_linux.go`, `syscall/syscall_unix.go`, `syscall/zsyscall_darwin_arm64.go` and `syscall/zsyscall_linux_amd64.go`, all under `src/`. HEAD carries 19 other seven-character index headers; the candidate carries 25 seven-character headers.

Independent comparison after removing only index-header lines confirms all other patch bytes are identical. Candidate-versus-retained `canonical.patch` comparison exited 0. Independent `diff -rq` of `/tmp/fn155-source-gates-20261010.yUjbK4VS/diagnostic/checked/go` and its `regenerated/go` sibling exited 0. These checks support the six-header explanation for this retained failure and exclude source-hunk or materialized-tree changes in this correction.

The unchanged regenerator fixes one context line but does not pass an explicit `--abbrev` argument. Seven characters describe the retained canonical output under its recorded Git environment, not a new universal guarantee across ambient Git abbreviation configurations. Changing generator policy would exceed this correction's scope.

## Generator inputs and compatibility

`internal/gomadtool/generation/protocol/protocol.go:594` includes the complete patch bytes among the seven choice-implementation inputs. Its length-prefixed SHA-256 formula produces `c2473abc8556ee7e851681dec7801c3390c152e8101475152cbe32b054e9fb81` for HEAD and `c4aded67729619fa86fdb41fd1e04fe2d869407ccbfde7154f0d7380ed34a93a` for the candidate. An independent read-only Perl SHA-256 calculation reproduced both values. The two changed Go byte arrays match the new digest exactly.

Only `choicewire.go.tmpl` consumes `ImplementationDigest`; the runtime and test templates do not embed it. This accounts for precisely the host and overlay codec constants changing. The live-capability input list at line 564 excludes the patch and both changed generated codecs. Its included `runtime/gomad.go` and all other inputs remain byte-identical. The product-manifest comparison excludes other generated-output drift.

Equal materialized patched Go source does not preserve execution identity here. `toolchain/buildkey.go:48` hashes the complete patch and overlay bytes, and `choice/trace.go:100` binds the generated source digest and toolchain build key into choice implementation identity. `choice/tape.go:445` rejects mismatched execution identities. The corrected inputs therefore require a new matching build and identity-bound artifacts; this assessment grants no cross-candidate replay compatibility or old-build qualification. Recorded wire layout, version, magic values and non-identity codec bytes remain unchanged.

## Observed worker evidence and limits

The reviewer launched no Go, test, build, lint, compiler, formatter or generator process. The following completed worker receipts and corresponding logs were read under `.flow/tmp/fn155-canonical-fix-20261010.7JxdKVdm/`.

| Receipt | Observed result |
| --- | --- |
| `protocol-generate.receipt.json` | Exit 0; 1.984 seconds; log seal `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` |
| `validate-toolchain.receipt.json` | Exit 0; 3.128 seconds; log seal `c9621da06eafa3000fbd15da760654996246d90221b77efc15d21a3c30fe7aef` |
| `archive-tests.receipt.json` | Exit 0; 67.273 seconds; both selected archive tests PASS with zero skips; log seal `c7bfdddc6f8e6352c40d3ef018ce2b52a96f1f8c317763060ecd4785dbea9fc5` |
| `patch-policy.receipt.json` | Exit 0; 0.508 seconds; selected positive and negative patch tests PASS |
| `generation-choice-wire.receipt.json` | Exit 0; 0.807 seconds; all-endpoint generation check, input-identity and selected codec tests PASS |

Independent hashing confirmed the first three log seals. Worker receipts establish ordinary stock-Go source checks on linux/arm64, including the regression that previously failed. Root must retain the worker's final before/after candidate binding and any remaining command results. No terminal lint receipt was available when this assessment read that log; historical RED50 lint evidence remains unresolved by this source verdict.

Missing supported-platform patched build-key, compiler/runtime, boundary-off and boundary-on native execution evidence remain open. fn-155.1 explicitly owns its first native proof on darwin/arm64 or linux/amd64; the existing fn-128/fn-149 transfers do not waive it. fn-155.1 remains `in_progress`, and fn-155.2/.8 remain gated. This assessment supplies neither formal implementation-review acceptance nor full-host or full-task completion.

The reviewer wrote only this report through `apply_patch` and made no product, index, commit, lifecycle, native-revival, PR, push or CI change.
