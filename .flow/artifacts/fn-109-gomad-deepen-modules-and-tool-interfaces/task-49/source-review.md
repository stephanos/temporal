# Task49 independent source review

Accept the frozen adapter-cache correction as verified source progress. No actionable Critical, Important or Minor finding was identified. This review grants no formal SHIP or completed qualification.

Fresh review under requesting-code-review read AGENTS.md, the complete Gomad README, MILESTONES, task49, applicable R18/R19, the primary source audit, worker handover/observations and conductor-checks. Requested reviewer is gpt-6.1-sol/high, in the same GPT family as the writer. Actual host model metadata is not exposed. Tier is session with jev-unavailable(no_key); the explicit project reviewer selection governs the request.

## Source and preservation

BASE is `8486dcb98d2b15e1f985d5bb1e79ae7da81a0d5a`; HEAD remains BASE with the uncommitted task49 candidate. Production SHA256 is `bb99674446c997ad48fad63050ea8ce275f1de860b3e6f7c5a70e5faddce6d25`; new test SHA256 is `e46f0b8272d4878c1321c5cf8c94349d0e2852b907b398d9311362bc34d7cdcc`. Both remained fixed before and after the independent tests.

At adapter_registry.go:385-405, the existing deferred scratch lifetime now checks actual RemoveAll failure. Healthy cleanup leaves the original primary object untouched. A sole cleanup failure returns its concrete OS error directly, and an additional cleanup failure joins after the primary. Explicit returns still select the original prepared values or zero values before the defer executes. Replacement validation, Rename, inventory verification and relocation remain unchanged at :406-441. Published replacements survive scratch failure. Registry prepare at :257-271 and the public consumer at :319-322 stop before target/evidence/modfile publication on the helper error; cleanup retains infrastructure classification.

The entire new test file exercises the existing implementation callback and real filesystem. Mode-000 nonempty scratch produces ReadDir permission denial and actual RemoveAll openfdat permission failure on UID1000. Direct-owner tests pin concrete error type and primary-first pointer identity; registry tests pin zero target, nil evidence, absent gomad.mod/sum, source module bytes, published cache bytes/modes/relocation and healthy retry. Validation controls preserve root/inventory errors, corrupt cache bytes and non-directory obstruction behavior. Literal inventory framing and file SHA256 independently recompute to the stated digest; the stable basename is fixed rather than derived from the owner under test. Existing tests/comments, pins, public signatures and source-selection policy have no source diff.

Task49 retains task48 as its dependency. Task21 retains all 26 old dependencies and adds only task49. Its Acceptance-through-EOF bytes exactly match BASE with SHA256 `244837d75f8cb29005d38e463af06b58d33d63c5211b51cca5c1250c16fa8330`. MILESTONES changes only the task49 row and retains delivery order.

## Independent execution

Pinned stock Go1.27.1 ran offline on developmental linux/arm64, UID1000, with GOTOOLCHAIN=local, GOPROXY=off, GOSUMDB=off, GOWORK=off and GOENV=off. The worker overlay contains exactly one production mapping. Its mapped bytes compare equal to `git show BASE:tools/gomad3/deterministicio/adapter_registry.go`, SHA256 `657ea0a8b44726ea13baaa4b65dcafcb210f450a5bf60f319bcc1914027401c4`.

| Command | Exit / real seconds | Observed result | Log SHA256 |
| --- | --- | --- | --- |
| `go -C tools/gomad3 test -tags test_dep -count=1 -json -overlay=/Users/stephan/Workspace/skunkworks/gomad/temporal/.flow/tmp/next-gate-8486dcb98d/worker/base-overlay.json ./deterministicio -run '^TestAdapterCacheCleanup'` | 1 / 0.15 | Five fault subcases fail; eight control subcases pass; zero skips | `06ee126493202722649c8dd2173f26571f71b506a0802922cb5b5544536dceb6` |
| `go -C tools/gomad3 test -tags test_dep -count=1 -json ./deterministicio -run '^TestAdapterCacheCleanup'` | 0 / 0.13 | Four top-level tests and 13 subcases pass, 17 passing test records; zero skips | `4b8fcc6566f9cc9c6a885eb7a67354472597a2959160b43648e06489f1ea83f2` |

Logs are `.flow/tmp/next-gate-8486dcb98d/reviewer/base-regression.log` and `final-regression.log`. These reproduce the five final-test BASE faults independently. The worker's retained initial four-fault TDD log and final five-fault BASE log use different test-source tuples; their hashes match observations.json. The reviewer did not witness the historical editing sequence or reconstruct the initial test source.

## Retained gate evidence and limits

All five conductor log hashes, all 13 worker log hashes and the baseline full-lint hash match conductor-checks.json. Root architecture has six passing top-level tests, validation passes, and changed-line fast lint passes across 55 host packages and reaches errortype. The retained worker vet/direct errortype pass. The reviewer independently reran the header-position-only lint block comparison. Full lint changes exactly 266 to 265, removing only this unchecked RemoveAll; all 265 residual blocks match, with 208 errcheck, 2 exhaustive, 11 forbidigo and 44 staticcheck. Full lint still exits2 before its errortype stage. Scoped lint changes two to one with the unchanged profile.go:211 forbidigo. The original wide Quick selection is unchanged and retains the same 12 failing test names before/final; those required checks remain red.

Worker handover's conductor checks were pending at handoff; conductor-checks.json now supplies their later receipts. Proposed additional uncached/no-selected-adapter controls were not newly executed and are disclosed. Private fixture evidence establishes neither public pinned adapter qualification nor native platform acceptance. Native Darwin/full-host/full/default/integration/functional/affected-consumer, matched-first-baseline, bounded10/100, formal and predecessor requirements remain required and open wherever unproved. Native Linux stays deferred and unverified under fn128; linux/arm64 qualifies no platform.

All reviewer commands are terminal. Go/cache lane released to the conductor. Root owns lifecycle, any fixes and the separate verified-progress commit.
