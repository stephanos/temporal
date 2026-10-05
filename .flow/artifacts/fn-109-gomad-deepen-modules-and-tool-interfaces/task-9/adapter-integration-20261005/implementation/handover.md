# Task9 bounded adapter listing source progress

Adapter authors now receive an overflow error instead of a digest when either listing stream exceeds 4 MiB. The BASE regression used complete valid JSON padded above that capacity; the migrated helper refuses overflow before decoding retained bytes. The public helper delegates through the private `adapterPreparedSourceSetSHA256With` and existing `Compatibility` runner, with 4 MiB per stream and the existing 15-minute watchdog. The measured BASE maximum is 4,075 bytes across 22 initial and six corrected real listings. This sample establishes over 1,029 times headroom, while complete prepared-pin qualification remains open.

Task `fn-109-gomad-deepen-modules-and-tool-interfaces.9` remains `in_progress`. BASE is `34a958a61a6dac4315c3b5e912e1dcbe9a3e7665`. The conductor owns commits, lifecycle, integrated lint and fresh source/evidence reviews. Worker commits are empty.

| Evidence directory | Exit | Observation |
| --- | --- | --- |
| public-bounds-red | 1 | Unchanged production accepts all three overflowing valid-JSON prefixes. Two exact-capacity cases pass; three cases and their parent fail. |
| adapter-focused-final | 0 | 35 test results, five top-level tests, zero failures/skips. |
| focused-command-packages | 0 | 57 results, 33 top-level tests, zero failures/skips; existing capture/watchdog/cancellation/descendant controls. |
| bounded-target-controls | 0 | 43 results, 16 top-level tests, zero failures/skips. |
| architecture-source-sets | 0 | Three architecture/public-signature/purity tests cover both supported source sets. |
| validate-after-integration | 0 | Check-only generator validation ran before broader checks. |
| standalone-errortype-final, format-final, diff-check-final | 0 | Final test/source hashes bound; formatting and diff outputs empty. |
| scoped-lint-final | 1 | Two unchanged ST1005 sites in `target/internal/build/context.go:40,59`; introduced 0, resolved 0. |

The final helper suite covers exact capacity and limit+1 on each stream, dual overflow, caller context/deadline, raw argv and empty/relative Dir, ordered environment overrides, default timeout/grace, GOPATH lifetime, import override, Go ordering/deduplication, exact import-comment suffix, malformed/trailing/missing-field listings, Go/foreign read failures, and execution/overflow/watchdog/infrastructure precedence. Raw exit 7 is obtained from a controlled child. Injected failure propagation supplies no genuine OS-close-fault proof.

`verification.json` records the exact inverse production transform, preserved public comment/signature, unchanged GOPATH cleanup and decode/source projection, the seven-line owning documentation addition, and all 987 tracked nested-module hashes. Only the admitted helper and architecture document changed; the focused test file is declared new. Every other tracked nested input, including hostexec/gocommand, pins, generated files and the four deterministic-I/O regeneration/verification consumers, retains its BASE bytes. Fn113 keeps exact pin, approval, regeneration and publication ownership. The conductor's MILESTONES emoji and both user `.turbo` files remain outside worker edits.

The unchanged public process probe matches all 29 fresh BASE cases under only the existing recorded normalization, with 18 children reaped and 18 GOPATHs removed. Literal fixture bytes, selected Go/foreign inventories, hashes and both BASE digests match. Raw listing JSON matches after replacing only the fixture directory; the path length explains 634 versus 572 bytes. `source-selection-final.go` retains the original fixture/measurement code while omitting the redundant cached-module survey.

Architecture, generator, shared-mechanism, probe and source-selection receipts precede the final test-only raw-nonzero fixture and two table entries. Their production and existing tests stayed identical. Final helper tests, unfiltered lint, standalone errortype and formatting bind the final added test SHA. Each receipt retains argv, cwd, HEAD, environment, UTC timestamps, monotonic elapsed time, actual exit/signal, stream hashes and before/after inputs. Early receipts also retain the Go executable hash in their input snapshots. All worker sessions are terminal.

The original predecessor, matched-first-baseline, preservation, full/default/functional/affected, formal/native Darwin and remaining static acceptance stays open wherever unproved. The 15-adapter/two-platform exact-pin test remains behind its unavailable `.toolchain` gate and was not bypassed or rerun. Linux remains unverified and nonblocking under fn128. The conductor will execute integrated `make lint-code-gomad3` against the original baseline; its previous 319 findings and unreached integrated errortype remain historical until that observation.

Defect route:
- prior fixes: task40's integrated mechanism and task39's cleanup reused; current helper history confirms this omitted direct listing. External PR/tracker searches were not performed under the conductor's bounded offline source admission.
- diagnosis: actual BASE accepts valid JSON padded above 4 MiB on either stream; exact-capacity controls succeed, eliminating malformed input as the cause.
- introduced by: skipped; no known-good bounded public-helper revision supplied.
- base: behavioral RED before production edits; final: same public boundary assertions GREEN. A reproduction commit was not made because the conductor is the sole committer.
- live: no separate live application surface; real child and unchanged public process probe supply execution evidence.

stage: impl-review - skipped(policy: bounded source admission; original gates red/unavailable; conductor owns review)
Tier: session (jev-unavailable(no_key)), pinned implementer retained.

Pointers: `evidence.json`, `verification.json`, `source-listing-comparison.json` and each listed command directory. Existing BASE scripts and raw captures remain unchanged.
