# Task9 evidence progress review

Assessment: `SOURCE_PROGRESS_COMMIT`. No outstanding introduced P1, P2 or P3 findings in this bounded evidence review. Task9 was in progress at review; the conductor owns subsequent lifecycle disposition. This assessment supplies neither formal SHIP nor DONE.

Reviewed at 2026-10-05 20:22:41 UTC in a fresh reviewer context, using the requested Sol/high reviewer tier from the same model family as the writer. Scope includes the new implementation captures, runner versions, source-selection driver, evidence and preservation records, and the actual conductor lint and final helper receipts. I read AGENTS.md, the Gomad guide and milestone ordering, flowctl usage, and the task through the CLI. The task's dated source admission authorizes these three source surfaces while retaining predecessor dependencies and original acceptance. I used read-only Git, source and artifact inspection, with no Go, test, lint, generator, cache, lifecycle, index or history mutations. This review file is my only write. The earlier independent review covered the committed BASE evidence; I examined the BASE process and literal-listing controls needed for this comparison.

## Observed results

| Capture | Actual result | Evidence bound |
| --- | --- | --- |
| public-bounds-red | Exit 1; 2 passed, 4 failed including parent, 0 skipped | Production matches all 987 BASE nested inputs. Three valid complete JSON listings with whitespace padding above the capacity incorrectly succeed. |
| adapter-focused-final | Exit 0; 35 passed, 0 failed/skipped | Final helper, new test and architecture hashes match current source. |
| root-adapter-focused | Exit 0; 35 passed, 0 failed/skipped | Same 990-input manifest as actual integrated lint; 2,557.170918 ms. |
| focused-command-packages | Exit 0; 57 passed, 0 failed/skipped | Existing hostexec, gocommand and capabilityreview capture/watchdog/descendant controls. |
| bounded-target-controls | Exit 0; 43 passed, 0 failed/skipped | Existing cleanup, preparation-digest and canonical projection controls. |
| architecture-source-sets | Exit 0; 3 passed, 0 failed/skipped | Architecture, signature and purity checks retain both supported source-set coverage. |
| validate-after-integration | Exit 0 | Actual check-only generator and pack commands retained. |
| scoped-lint-final | Exit 1 | Exactly two ST1005 sites in unchanged target/internal/build/context.go, lines 40 and 59. |
| standalone-errortype-final, format-final, diff-check-final | Exit 0 | Final source inputs match; formatting and diff output are empty. |
| root-integrated-lint | Make exit 2; null signal; 192,965.901925 ms | Actual original-base lint invocation with fix=false. Entire stdout equals task44's retained stdout byte for byte. |

The integrated gate observes 55 host packages and 319 findings, comprising 252 errcheck, 3 exhaustive, 11 forbidigo and 53 staticcheck findings. Its full stdout SHA-256 is `3fb126eb1e38a595c2812cdf0e01e9eb87704585f560ec6fb1adce5263e7ba8f`. Integrated errortype is unreached because make stops at lint; standalone errortype does not replace it. I independently checked raw stream hashes, command argv and cwd, executable identities, before/after inputs, termination status and timestamps for all 15 worker command records and both conductor records.

## Preservation and provenance

All 29 final public-process cases equal fresh BASE after only the retained volatile-input normalization. Concrete error types, ordered unwrap chains, errno/operation, ProcessState/signal/exit, context identity, stderr, quoting, digest and cleanup observations remain compared. All 18 observed started children are reaped and their observed GOPATHs removed. The normalizer preserves meaningful literals and replaces the exact executable/scratch spellings, generated missing-command PID, startup PID/time/GOPATH and ExitError PID.

Both real stock-Go source listings preserve literal fixture bytes, selected Go/foreign inventories, selected source hashes and BASE digests. Their raw JSON is equal after replacing only the fixture directory. The final fixture listings are 572 bytes versus BASE 634 bytes, reflecting the directory spelling. The fixture and measurement function in source-selection-final.go match the BASE driver; it omits the cached-module survey. The measured 4,075-byte maximum justifies sample headroom for the admitted 4 MiB refusal policy, not all 15 prepared pins or native platform qualification.

Exactly two of 987 tracked nested-module inputs change, the admitted helper and architecture document. The focused test is new. The public signature/comment, checked GOPATH cleanup and decode/projection remain preserved by the narrow transport transform. Shared hostexec/gocommand, generated inputs and pin consumers retain BASE bytes. Removing only the final exit-7 fixture/import and its two table entries reconstructs the earlier test SHA `8f46b106c45b470f2751f44d9015091e3d7fa1b2bbd09bd55d94757ccfd3be68`, which matches shared, architecture, process and listing receipts. Those checks remain applicable to unchanged production and existing consumers. Final helper, format, scoped lint and errortype receipts bind the final added tests.

The recovered adapter_source_set_test.red.txt SHA `4398a439879a34b3d75958ffc67264e23f827a8053302a580d7d6914130c3e1b` matches both RED snapshots. Independent literal-preserving comparison confirms that the public-boundary and child bodies differ only in whitespace and explicit struct-field separator formatting. Assertions, five cases, payloads, argv and cleanup checks are unchanged. The final source additionally imports and defines the separate expanded test coverage.

Three early receipts bind the original runner SHA `a83b1f0b65a1b2134bf45f9301ede83545f7c52b4a0a0225a0dc925f92fbb36e`. Recovered run-initial.cjs matches all six before/after bindings, resolving the provisional P3 reproducibility gap. Their Go executable hash is in the input snapshots; later receipts add explicit executable fields. Current run.cjs remains `8223d7987c1a8ca98a2f37f3dd30bbc8f47ad58865ec78429c4b0bb2011e017e`. Raw captures were preserved. The corrected network wording claims module/checksum downloads disabled, and the corrected handover describes complete padded JSON rather than incomplete data.

## Identity and remaining limits

Source BASE is `d2e0e035519f1385b9acf630a70152655b113f61`; HEAD is `34a958a61a6dac4315c3b5e912e1dcbe9a3e7665`. The intervening committed range changes only Flow files. Retained implementation execution spans 2026-10-05T20:03:48.969Z through the conductor's final helper receipt at 2026-10-05T20:19:15.607Z. Current source hashes match the final receipts at review close.

| Source | SHA-256 |
| --- | --- |
| target/adapter_source_set.go | cceeb09d76033404ff684bb9c57efd648dd909b230f0656da7a818173f3cd31d |
| target/adapter_source_set_test.go | 39613c9c1510def7aef69b191792f4d6590249e4ddb486f0918c07dc506a946c |
| ARCHITECTURE.md | 13f5fbeffe0f17a68073e00bb380facdd970f7131d6358e39af7ceedc4b39043 |

The conductor's 990 selected-input manifest is `bcd764726baa150e469b20d764a5b8c6b8d0caa400c11b17664e5ed0568091d8`; independently recomputed current inputs match both before/after snapshots. Its runner is `67678a91583b702a0117da5a4fda05db41d8eca5d8d4af0207e57f49132723da`. Actual integrated receipt SHA is `127a0defefb27713ad3a252b7adcb08c0110a8b9af1aa3a7e8c61f96ab5d2ba3`.

Execution used stock Go1.27.1 on developmental linux/arm64. The selected-input freeze covers the recorded tree and tools, not every repository, toolchain or environmental input. Injected precedence and deadline forwarding establish propagation; actual existing mechanism controls establish process behavior. Injected cleanup failure supplies no physical OS-close-fault proof. Exact pins, approval, regeneration and publication remain with fn113. Native Darwin, patched-toolchain, predecessor, matched-first-baseline, full/default/functional/affected, original preservation and remaining static/formal acceptance remain open where unproved. Native Linux remains unverified under fn128 and nonblocking for this source checkpoint.
