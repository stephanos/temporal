# Checkout revalidation source-progress review

Verdict: source-progress-acceptable.

The frozen correction protects a user's same-byte symlink replacement during approved adapter regeneration while retaining byte-based compatibility for regular files. This receipt covers the bounded checkout type drift admission under fn-113.2 R3. Original task acceptance remains open.

## Review identity and scope

Fresh independent host review using the requesting-code-review calibration. Writer and reviewer are both Codex family gpt-6.1-sol at high effort. Same-family review is disclosed. This is a source-progress assessment, with no formal impl-review, SHIP, task-completion or merge verdict.

Workspace is `/Users/stephan/Workspace/skunkworks/gomad/temporal`, branch `gomad`. HEAD and base are `717a2de678c743e0f47e07608f94dada3d315330`. The candidate is the working tree. I read the physical AGENTS.md, full Gomad README, MILESTONES.md, fn-113 spec and task admission, exact production diff, complete additive test, fixture helpers and surrounding staging, revalidation, journal, publication, Run and recovery paths. I inspected worker-handover.md, worker-observations.json, conductor-proof.json and meaningful retained logs. Source, index, HEAD, branch and lifecycle state were unchanged by this review. Only this receipt and two reviewer rerun logs were created.

Verified source SHA256 bindings:

| Input | SHA256 |
| --- | --- |
| Baseline transaction.go from Git and overlay | `7b1daf2868cac62ef0740c0ec63b143c49e2db8c84a34e85fb096e91dd7e65f8` |
| Candidate transaction.go | `1b710dec9c0bb7e8517d919682e853b97ff0bf76dfa9d70d3fc39a3cd6825986` |
| checkout_revalidation_test.go | `bb622e84650b111c6bf349a6265bbb13492ebb7b089876f82a359c65e5bde7ce` |

All thirteen current source/config bindings in worker-observations.json match their files or exact Git baseline. The historical initial RED test hash is disclosed separately and its earlier source revision is not the frozen candidate. The Go executable hash matches `1675694ef690db0f18fbe7046a886170904bede1d9db6ec96ae27945c1705c64`. The lint and errortype executable hashes match conductor-proof.json. All nineteen retained evidence-file hashes in that manifest match. The inspected conductor-proof.json SHA256 is `181e70e49136bbd3c69942bac28415ca06f2eeff43f84d75b6b57abd67507463`.

## Strengths

- `transaction.go:673` mirrors copyCheckout's existing `entry.Type().IsRegular()` policy. It executes after the appeared check and before fileDigest, so existing-path symlinks return the existing changed message and newly added symlinks preserve appeared precedence.
- `transaction.go:138` wraps refusal in BlockedError before writeJournal at line 141 and completeJournal at line 144. The exact diff adds only the three-line regular-entry refusal. fileDigest, Recover, locks, journal retirement, generators, comments, pins, guards, schemas and signatures remain unchanged.
- `checkout_revalidation_test.go:12` exercises public Run with the existing offline fixture, generator, staged verifier and afterStage/beforeApplyFile seams. Real symlinks are created. Separate Readlink checks establish retained link identity, external-byte checks protect the target and snapshot equality protects every other checkout file. Refusals assert empty publication/staged results, no writer entry and no remaining journal state.
- The same-byte regular replacement at test line 62 verifies a different inode and still succeeds. Both regular success controls require descriptor, adapter, fixture module and generated-output publication, with generated content bound to the published descriptor. Changed regular bytes, new symlinks and initial symlinks retain their existing refusal behavior.

## Issues

### Critical

None found in the admitted correction.

### Important

None found in the admitted correction.

### Minor

None requiring correction.

## Independent reruns and evidence checks

Both reviewer commands exported the following environment before invoking Go:

```sh
export PATH=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin:$PATH GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOWORK=off GOENV=off
```

Final-source command, captured in reviewer-focused.log:

```sh
{ time -p go -C tools/gomad3 test -tags test_dep -count=1 -v -run '^TestRunCheckoutRevalidation$' ./upgrade/adapterregen; } > .flow/artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-2/checkout-revalidation-progress/reviewer-focused.log 2>&1
```

Exit 0. Shell real 2.83 seconds; package 2.490 seconds. Six passing leaves, seven passing test records, one top-level test, zero failed tests and zero skips. Log SHA256 is `f1acd5ce01744a39f0ce42cc5d5672b9ba2768e7bdf74f4ae268a9d21b1a665b`.

Exact baseline-production overlay with the same frozen additive test, captured in reviewer-red.log:

```sh
{ time -p go -C tools/gomad3 test -tags test_dep -count=1 -overlay=/Users/stephan/Workspace/skunkworks/gomad/temporal/.flow/tmp/checkout-revalidation-base/overlay.json -v -run '^TestRunCheckoutRevalidation$' ./upgrade/adapterregen; } > .flow/artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-2/checkout-revalidation-progress/reviewer-red.log 2>&1
```

Exit 1. Shell real 3.08 seconds; package 2.612 seconds. Five passing leaves, one failing leaf, zero skips. The parent test also fails. The only failing leaf is same-byte-symlink-replacement. It reports nil error, Applied=true, eight published paths, writer entry and Readlink invalid argument after publication replaced the link. This independently confirms the missing type validation without reverting shared source. Log SHA256 is `54907615486fd77578a84b25dc2b4c44ce730e998f6259a652400dbb1aa0f673`.

I recomputed the worker JSON event counts. Baseline portable evidence has 24 top-level tests, 38 passing records and 34 leaves; final portable evidence has 25 top-level tests, 45 passing records and 40 leaves. Both have zero failures and zero skips. Shell real times are 18.50 and 40.54 seconds. Final worker RED has five passing leaves and one failing leaf at 7.45 seconds; GREEN has six passing leaves at 5.83 seconds. The retained conductor focused rerun has six passing leaves and zero skips at 3.34 seconds.

Retained conductor vet, errortype and check-only validate report exit 0 with real times 0.19, 0.23 and 5.35 seconds. Architecture evidence contains all four named passing tests at 170.16 seconds. Changed-line make lint-code-fast reports exit 0 at 3.81 seconds. Its processor log shows 302 configured residual findings before diff filtering and zero afterward, so it does not establish full lint cleanliness. Scoped baseline overlay and final lint each exit 1 and contain identical diagnostic blocks for unchecked lock.Release at transaction.go:89 and :219, with real times 1.83 and 0.59 seconds. No owned lint diagnostic or new suppression appears. Formatting lists no scope files and diff checking passes.

The initial missing /usr/bin/time launcher executed no baseline tests. The earlier diagnostic-only RED revision and absent metadata-path probes are disclosed. Initial RED failure counts agree with its log. The optional reviewer source search encountered an absent scripts/golangci glob, then inspected the actual Makefile/config directly; this supplied no verification result. Flow validation retains 21 valid specs and 190 tasks with two inherited uncovered-ID warning groups.

## Recommendations and assessment

Keep the original task acceptance, task1 dependency, native Darwin/full/default/functional/affected-consumer and formal gates open wherever unproved. Linux qualification remains deferred and unverified under fn128. Neither the existing revalidation/publication TOCTOU window nor independent Recover behavior is closed by this guard. The retained lint debt is outside this bounded source admission and remains red.

The exact three-line correction matches the admitted policy and preserves error precedence and regular-file compatibility. Public real-filesystem RED/GREEN controls and the independent baseline-overlay rerun establish the bounded defect and refusal before publication. The available portable evidence supports source-progress-acceptable only.

No reviewer commands remain live.
