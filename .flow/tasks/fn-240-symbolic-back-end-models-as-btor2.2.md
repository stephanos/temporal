---
satisfies: [R2]
---
# fn-240-symbolic-back-end-models-as-btor2.2 Pinned rIC3 admission and runner: build from a pinned source, IC3 for proofs, BMC to native depth plus one, lifecycle and verdict kinds

## Description
Add model/scripts/fetch-ric3.py shaped like fetch-apalache.py: pinned rIC3 version and source checksum, private staging, build with cmake and clang (document the shim recipe from model/design/native-symbolic/run.sh), atomic publish into model/build/ric3 under a lock, symlink refusal, concurrent installers converge. Check and record the rIC3 license before admission. Runner in umpire-symbolic: discover the binary, run ric3 check --cex --ui=false with engine ic3 (proof), bmc --end N (N = native BFS depth plus one, reuse umpire-apalache's native_bound), optional kind; bounded timeout, output drained, children reaped on timeout and cancel, one work directory per invocation under the build root. Verdict enum Safe, Unsafe{witness text}, Bounded{depth}, Timeout, NotRun{reason}; Refusal for admission and execution failures. Fake-tool tests for wrong version, failed launch, timeout, child cleanup, symlink escape, concurrent install; a real Kettle check under UMPIRE_BACKENDS=require.

## Acceptance
- fetch-ric3.py installs the pinned ric3 into model/build/ric3 deterministically and refuses a mismatched checksum or symlinked root.
- Runner returns Safe for Kettle's safe claims under ic3 and Unsafe with a witness for never_beeps; Bounded under bmc; Timeout and NotRun are distinct and never blessed as a pass.
- Fake-tool tests cover version, launch failure, timeout, child cleanup, symlink escape and concurrent installation.
- No files written outside model/build; default make model-check does not build or download rIC3.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
