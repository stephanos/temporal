# Retained-success source checkpoint

Both original retained-success tests now reach and pass their unchanged shared-inode, full-byte capacity, same-output artifact and disk/journal/summary assertions. The integrated source adds only two call-local preparation assignments in `retention_test.go`. All remaining source acceptance stays open.

Root integrated isolated checkpoint `c668243e0ecab6e4080aa7dad0810ccc2cedb08f` as PRIMARY `fd9cfc4026db966a877597a9658a0af589f18aea`. Actual implementation BASE is `c506713ce063759c5d24129d775d2fefc6314618`; the earlier admission's BASE is explicitly reconciled in the frozen handover. Candidate SHA-256 is `b289d29192e3b6de5ff23d1ed8e33627b795c570d32ae727d09f82ea2f6cf9a6`. Removing precisely the two admitted lines reconstructs the complete BASE file, SHA-256 `35a5599194d809a364751743318dc3c9e7304810bbeb8fd819e69dc8d3f14194`.

The isolated worker packet retains 102 sealed members plus its seal, SHA-256 `863e5ef514d69f0920a0f4decf0d8bcc77f26d814767709a77683fb212bb1c97`. Its precommit handovers intentionally retain empty commit arrays. This note maps the later checkpoint without rewriting those handovers, their failed/inconclusive attempts or any sealed member. Both fresh worker reviews accepted bounded source/evidence claims with Critical 0, Important 0 and Minor 0. Their complete reports are retained outside the isolated seal in PRIMARY task-69.

Root executed the prepared comparison wrapper once on clean frozen HEAD `c668243e0ecab6e4080aa7dad0810ccc2cedb08f`. Wrapper session35593 terminated with exit 0 because the comparisons were valid; both aggregate gates remained nonzero. Root waited for all four commands and released its exclusive Go/build/lint/vet/generator lane before dispatching worker70 in a separate checkout. Source acceptance for fn-109.69, fn-109.63 and fn-112.10 remains OPEN/RED. No formal implementation review, SHIP or Done occurred.

| Actual root command | Exit | Seconds | Result |
| --- | ---: | ---: | --- |
| Ordinary Runner, `test_dep`, count1, JSON | 1 | 102.160 | 673 named outcomes, 394 PASS / 267 FAIL / 12 SKIP |
| Original-base `make lint-code-gomad3`, fixes disabled | 2 | 18.956 | RED50, all complete blocks unchanged |
| Darwin/arm64 Runner, conformance and execution vet, CGO=0 | 0 | 2.784 | Static supported-source selection |
| Linux/amd64 same packages and flags | 0 | 3.266 | Static supported-source selection |

The immediately preceding combined68 execution was `524c092a3f6cbf5834895ae5ba7821d6fa610924`, with 673 actual named outcomes, 392 PASS / 269 FAIL / 12 SKIP. Exactly `TestRunCountsASharedTargetInFullAgainstTheSuccessByteLimit` and `TestRunRetainsSameOutputSuccessesWithMatchingDiskAndJournalCounts` changed FAIL to PASS. Every other 671 named outcome is unchanged, with zero missing, added or synthesized names. The full 50 lint header/source/caret blocks are identical, with zero introduced or removed blocks and no source-line relocation. Integrated errortype remains unreached at the failing golangci Make recipe. Separate analyzer and changed-line fast-lint successes establish no aggregate-green result.

| Frozen binding | SHA-256 |
| --- | --- |
| Explicit 20-member root postcapture seal | `71b566f5f17961cb370c3791a8008116241503b0cc31f4b00f9fa406c06fa0b5` |
| Source-before and source-after, 1,265 entries | `406b032fddbc2289319419c13a4c9f7be33cd15491039e747ed12260412dee84` |
| Actual 19-tool inventory | `ffee63e3a581ac37ffa97b7af4eec395132d29dfacd519bcc2d24a60f5383a97` |
| Run binding | `a9e9e28681a1db072ec390a1ee1ad2388d5e1c3118c937c9ed99e5d64b78da97` |
| Ordinary raw log | `73ceada3c52c462a03f655c0be752b2c191d7a3925d0f30f624ef48aa0c438d9` |
| Actual outcome comparison | `25652b2082340d186580699e413af072fe1817f6b3527759f13d35435f91d975` |
| Full lint comparison | `f834ed400744ebecd8ee964f8a7f05818b96d0c17f5de34d72127528401c2290` |

Root independently checked all 20 sealed members, all 19 current tools and every one of the 1,265 actual source/comparison inputs. All checks exited 0. Every command binds actual source and tool bytes before and after execution, with equal digests. The wrapper prebound its effective Go settings and environment comparison; the later seal supplies only postcapture evidence. This note and the full integrated reviews are outside that fixed 20-member domain.

Both fresh integrated reviews accepted bounded source progress with Critical 0, Important 0 and Minor 0. The source reviewer independently reconstructed the whole BASE file, checked 1,129 PRIMARY/frozen product and executable-routing inputs, parsed all named outcomes and compared all complete lint blocks. Its full report is [independent-behavior-review.md](independent-behavior-review.md), SHA-256 `793f117b1681dda5de2af7ce73d43913eb4cc8b4cdb468c90d9829644aa32408`. The evidence reviewer independently checked both 20-member seals, all 1,265 inputs, all 19 tools, 1,126 executable-routing inputs, the frozen 103-file worker packet, every root receipt, metadata prebindings, named outcomes, full lint blocks and all 21 diagnostic-source identities. Its full report is [independent-evidence-review.md](independent-evidence-review.md), SHA-256 `a5a962cdd60a6df64178b2f94cd98a0ccb07213796d5f897da42f77f4b07dfc8`.

The authoritative PRIMARY owner spec remains SHA-256 `851151bc3b5ea0ac9bfda873f108a593653a9becbb66323d241244955274fd2c`. Frozen historical owner SHA `0866b495ef6150bd0341aa904358f5de49ab4977e88da25c67e9df1531106f8a` and historical milestone bytes supply no superseding authority. Root preserved all unrelated dirty user files and staged only its own paths or exact milestone hunks.

Stock linux/arm64 execution remains developmental. Supported-source vet supplies no full native test-host result. Selected environment, reused caches and executable hashes leave complete cache contents, tool installations, nonselected inherited settings and C headers/libc unqualified. Native fn-128/fn-149 remains deferred and unverified. No runtime/replay/soak qualification, determinism bound, native revival, CI, PR or push authority follows.
