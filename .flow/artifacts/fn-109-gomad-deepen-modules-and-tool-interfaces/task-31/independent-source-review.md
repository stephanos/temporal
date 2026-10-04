# Task 31 independent source review

Assessment is **SOURCE_PROGRESS_COMMIT_ONLY**. No actionable introduced source defect was found in the frozen five-file dirty diff against base and HEAD `39fc19c4618322b6a939f5b603d9ba69aab00b9b` on `gomad`. Task 31 remains in_progress. This review supplies no formal SHIP verdict or full/native qualification.

Requested reviewer and writer routes are gpt-6.1-sol at high, the same configured family. Actual executing model metadata is unknown. Root already reported `Tier: session (jev-unavailable(no_key))`; this fresh-context reviewer did not repeat the judge or bridge. The requesting-code-review and verification-before-completion skills governed independent review and fresh evidence. The repository prose contract governed reporting.

## Strengths

- `store.go:485-512` separates operation and cleanup until the context owner selects the original primary. Open failure acquires nothing; successful acquisition defers exactly one Close. The inner defer finishes before the post-context check. An operation error skips that check. Nil cleanup returns the exact primary object, sole cleanup returns its raw error, and dual failure joins primary first. No new IsDir, mode or path validation changes os.Open/Sync acceptance. The artifact package has only the one caller of this private helper.
- `target_pool.go:211-252` observes root and file cleanup at their original acquisition boundaries. LIFO attempts file Close before root Close, once each, with no retry. Each nonnil cleanup clears the named File result, assigns raw cleanup on prior success, or joins after an existing primary. Nil cleanup leaves literal successful metadata and exact errors unchanged. The original mode/size, copy/context, count/hash and successful metadata body is byte-preserved. The unchanged caller at lines 66-72 returns zero File, empty sharing and its existing wrapper on any verifier error.
- The unchanged Store and pool transaction bodies retain their separate owners. Store staging cleanup at `store.go:112` removes a failed verifier link; a pool winner remains owned independently. The no-replace artifact rename at line 179 precedes store-root syncing at line 227, so a newly observed later cleanup error grants no rollback or winner deletion. Pool rename and syncing at `target_pool.go:199-206` retain that same distinction. No additional publication retry, removal, manifest ordering or pruning policy was introduced.
- Real-file controls at `store_test.go:16` and `target_pool_test.go:72/109` check exact cancelled sentinel and raw PathError identity, literal successful metadata, bytes and mode. The existing damaged-pool matrix at line 235 gains store/staging and exact original pool-winner survival assertions at lines 290-301. Original assertions remain intact. The three original opened-handle defers at `publication_test.go:42/136` and `target_pool_test.go:322` now report Close errors through t.Error at the same lifetime.

## Issues

Critical: none introduced.

Important: none introduced within the admitted source-progress scope.

Minor: none introduced.

Actual unfiltered lint remains red on four inherited findings, all retained in `review-lint.log`. They are `manifest_copy.go:59` forbidigo panic, `opened_test.go:257/292` exhaustive reflection switches and `publication.go:39` ST1005 World error capitalization. The audited baseline has ten findings. Exactly six mapped errcheck findings are resolved and zero findings are introduced. No filtered, suppressed or waived-green result is claimed; historical whole-Gomad 419 remains historical.

## Verification and preservation

Seven fresh receipts use cached stock Go 1.27.1 on developmental linux/arm64, GOWORK off, GOTOOLCHAIN local, GOPROXY off, empty GOFLAGS and absent seeds. Whole artifact, focused directory/verifier/publication/pool, retained/private/public-copy and actual root boundary tests exit 0 with 77, 42, 31 and 15 RUN entries respectively. All five named root boundaries pass, including public signature negative fixtures and both external Runner consumer controls. Errortype and static diff/gofmt exit 0 with empty logs. Pinned unfiltered lint exits 1 with the four diagnostics above. `TestRecordAndArtifactHaveSeparateOwners` is absent and was not run.

The read-only audit in `review-run-gate.py` verifies all fifteen worker and seven fresh receipts against exact commands, cwd, environment, tool/config hashes, all 23 artifact-source pre/post hashes, elapsed/start/end times, timeouts, child exits and raw-log hashes. It runs the hash-bound exact root command from `root-source-checks.json`. That command strips audit.py's sole evidence.write_text and asserts its computed result equals the saved JSON. Worker evidence and handover bytes remain unchanged. The exact command and successful result are persisted in `independent-source-review-checks.json` for root's pre-Git reaudit.

The reverse recipes recover all five original files exactly from BASE. The corrected characterization recipe restores the original opened.Close defer, removes originalEntry and weakens only the newly added pool assertion to len(entries)!=1, reproducing `8e572579136ae7a097412fef3c671bda612f47fb21d7ab3ef6d539ecf88fe3fd`. The final stronger assertion stays intact. Every other product byte in the protected 1,039-input set matches aggregate `3e5fbd96cfea07cce2f2c2296a0e5064820f3167a17f0eb7c06f353dfa1e5e4c`. Public CopyPayload, private helpers, clone/reflection/invariants, canonical construction and resource accounting are unchanged against this task's base. This is no substitute for the still-open matched original first-baseline identity proof.

Generator input lists and validation recipes were inspected. They select version, protocol, boundary, compatibility and qualification owners outside these five paths. The protected aggregate binds those inputs, so the audited worker `final-validation.json/log` is reused without regeneration or an unchanged broad rerun. Both mutation recipes were reconstructed and their failure/restored receipts audited; no mutations were replayed.

## Recommendations

Root should run the persisted read-only audit before Git changes, then commit this bounded source progress. Use `review-run-gate.py` instead of running worker audit.py directly, since the latter rewrites worker evidence.

Keep genuine first-Close OS faults, simultaneous operation/cleanup faults, verifier metadata zero on genuine cleanup fault and post-Sync cancellation timing explicitly unexecuted. Those branches are source-inspected only. Cached Unix Root.Close returns nil, and the nil-wrap/second-Close mutants establish control sensitivity without supplying first-Close fault proof.

## Assessment

Ready for **SOURCE_PROGRESS_COMMIT_ONLY**. Conditional error merging preserves primary identity and order while observing cleanup failures at the existing lifetimes; fresh controls and exact byte recovery support the admitted change. Original R13/R18/R19, task12/predecessors/task21, matched original first-baseline fixed identities, complete/full/formal/both patched-native and affected consumer/integration/qualification gates remain open. No source, Git, index, Flow, worker evidence, config, dependency, pin or runtime edits were made by this reviewer. All seven owned gates and audit commands are terminal; no delegates or live command handles remain.
