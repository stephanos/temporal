# R19 mutation-coverage correction

The worker corrected the two mutation descriptions in `aggregate-evidence.mjs` and `evidence.json`, plus the matching paragraph in `handover.md`. The actual red/green tests are `TestPrepareCompositionOrderAndCleanupIsolation` and `TestPrepareCustomPreparerSkipsAdaptersAndValidates`, both in `./internal/preparation`. Those commands execute no portable-plan assertions. Independent portable-plan coverage remains in `source-existing-owner`, `current-callers-owned-cache` and `source-runner-portable-final`, with the exact relevant test names now stated in the handover.

This addresses the wording defect identified as `finding-97545ddf2f4a7566a3ebfbe1a81e06ce` in the conductor's first formal review. The worker issues no review or acceptance verdict. Root owns the fix commit and single-session re-review.

Candidate HEAD remains `34b398ecdadc9d79c64dec8d30dd9999b29ed66f`. Initial aggregation exited1 because its historical guard accepted only admission `b4602685b3184387cf2d713178095247f0c11d8f`. Root admitted exactly those two HEADs for this documentation verification. The corrected guard accepts only them and keeps every historical evidence base fact at b460. The original setup failure is retained in [review-fix-aggregation-guard.log](review-fix-aggregation-guard.log).

Aggregation then exited0 with the same forty command receipts, twenty coverage groups and ten task-8 source gaps. Read-only Node verification exited0 and proved that `evidence.json` differs from the committed original only in the two corrected disposition strings. All 120 other original artifact files are byte-identical to the committed candidate; their canonical path/hash-set digest remains `358d70bb00b98aa6c86f1b662888d49e003f13376fe53a8cf7106b0f71b3997b`. All 1,034 frozen source entries and 820 current control entries verify against their unchanged manifests. Frozen graph remains `c8bfba0aed3ba328aae4ffb7ad317b3cd051e379a66b54dabc6701676ac335e9`.

The first `git diff --check` exited0. A redundant final read-only invocation hung in `futex_wait` for at least 66 observed seconds at the verified repository cwd. The worker terminated only that owned process (PID 1703759, session 66926) and observed exit143; it supplies no additional pass. Original raw logs, receipts, hashes, source/tests/fixtures/controls, every red outcome and native deferral remain unchanged. No Go, build, lint or generation command ran. All worker commands are terminal. Verification outcomes and elapsed observations are retained in [review-fix-verification.json](review-fix-verification.json).

| Summary | Original SHA-256 | Corrected SHA-256 |
| --- | --- | --- |
| `aggregate-evidence.mjs` | `669cc462e047bc54789ce2aee7e44c9cb0e764c83f1c3f1370487ce0d060fd31` | `de336c4c6e470f2c693b0bb8004128e54c6990c0b4874f16ffff795a2485cb27` |
| `evidence.json` | `315e46d5295909fd242750f1441769b43c727b79e92d616ba6a1367a36ac2f71` | `f4714243602954d6c5bb8689366ce31aa3838d3f75199d92ac058c00b856e812` |
| `handover.md` | `829041504aeabb382f755449fbbbe46fe25b5b63232b473331f6c21b538c7eb2` | `b6f2310d5ac3540fc6daf6dadbfa01ce1f3a324c01c6ba3e8647754ba08fa43d` |
