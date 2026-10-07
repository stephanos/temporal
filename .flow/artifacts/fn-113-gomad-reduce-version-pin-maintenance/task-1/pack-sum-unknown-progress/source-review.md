# Missing pack activation sums source review

Assessment is `source-progress-acceptable`. No actionable P0-P3 finding was identified in the frozen three-file correction. This verdict covers fn-113.1 R2 source progress. Original native comparisons, Darwin/full acceptance and formal implementation review remain open, so it supplies no merge approval, formal SHIP or task-completion verdict.

## Reviewed identity

Fresh independent reviewer used `gpt-6.1-sol` at high. Writer and reviewer are both Codex family. Review followed AGENTS.md, the complete Gomad README, MILESTONES.md, the amended task scope, requesting-code-review skill and template, and the Flow prose contract. Base and HEAD at review were `a45ebab979f4a97f9237bab651cdb90e0dc6b42b`; the implementation was uncommitted.

| Source | SHA256 |
| --- | --- |
| `tools/gomad3/upgrade/pinimpact/pinimpact.go` | `f948c266c891ce0a04af14b109b98503df6518c422877fbcb1e3e9c0472c154e` |
| `tools/gomad3/upgrade/pinimpact/pack_sums_test.go` | `50cce5e8b252c6fc87515a3471f6f7bb3cfd2994826c76a1b7a620d4d1d5218e` |
| `tools/gomad3/cmd/gomadtool/pin_impact_test.go` | `1a89c21bac99eec790fc5c083a57bcde892493079f3d4f60b21760786cebb30f` |

The reviewer checked these hashes before and after the independent tests, verified the base production hash against Git, and verified every retained log hash and source hash in [conductor-proof.json](conductor-proof.json).

## Strengths and preservation

`activates` retains the first unknown identity while continuing to a definite exclusion. The rule evaluator propagates activation uncertainty only when the rule identity matches or remains unknown. This preserves definite absent, replaced, version and checksum exclusions even when another activation checksum is missing. A validated external pack exercises a rule module separate from its activation module. Baseline uncertainty alone keeps the repaired candidate's prior not-selected disposition.

The public Evaluate/Encode/Render regressions retain `/go.mod` checksums and check actionable unknown pins, useful `module_sum_missing` reasons and immutable input files. The CLI regression uses the real GoResolver with a file proxy and checks status 1 in human and JSON modes. The exact-base RED reproduces hidden rules, empty reasons and CLI status 0. The separate proposed-fix RED demonstrates why known rule exclusions must be checked before reporting activation uncertainty.

Known-exclusion controls assert literal statuses before comparing uncertain inputs with resolved controls. The reviewer independently compared all 19 base/final complete canonical and human digest pairs, which match. The external-rule test asserts literal dispositions and logs reasons; its assertions alone do not prove complete reason-byte preservation. Inspection of those outputs and the production branches supports the required stale, not-selected and specific invalidation reasons. Existing unresolved-graph reason bytes have an exact assertion. The admitted mixed-input changes restore the resolved exclusion instead of the former order-dependent empty unknown or hidden not-selected result.

The source diff introduces no production seam, API, schema, dependency, replacement-policy, pin, generated-input, runtime/toolchain, native-guard or existing-comment change.

## Verification

Independent reviewer command, on developmental linux/arm64 with stock Go 1.27.1, exited 0 in 0.853 seconds. [reviewer-green.log](reviewer-green.log) retains six top-level tests, 40 passing records and 36 terminal leaves with no failures or skips.

```sh
PATH=/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin:$PATH GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off GOWORK=off GOENV=off go -C tools/gomad3 test -tags test_dep -count=1 ./upgrade/pinimpact ./cmd/gomadtool -run '^(TestMissingPackActivationSumIsUnknown|TestPackActivationSumControls|TestKnownPackActivationExclusionWinsOverMissingSum|TestPackActivationGraphUnknownRetainsReason|TestPackRuleExclusionWinsOverUnknownActivation|TestRunPinImpactMissingPackSumIsUnknown)$' -v
```

The worker's portable command passes 29 top-level tests and 82 records. Final scoped pinimpact lint reports zero findings. Fresh conductor expanded lint retains 129 errcheck findings in both the exact-base-production/final-test overlay and frozen final source. The reviewer compared their entire diagnostic text after removing timing footers and found equality. The earlier worker expanded lint covers a proposed revision only. Mandatory fast lint passes after filtering 302 residual configured findings. Full lint remains red. Conductor vet, errortype, check-only validation and four architecture checks pass; formatting and diff checks pass. Setup-only failed lint attempts remain explicitly classified in the receipts.

## Open acceptance

`TestFixtureBumpMatchesBuildRejections`, `TestSameVersionWithChangedSum` and `TestReplacedModules` remain unchanged and unproved. The reviewer ran no native preparation, runtime or full-host qualification. Required Darwin/full/formal acceptance remains open. Linux remains deferred and unverified under fn128. The reviewer changed only this review artifact and generated its independent test log, leaving source, index, HEAD, Flow state and unrelated `.turbo` files untouched.

## Evidence encoding recheck

The final evidence-only recheck preserves `source-progress-acceptable`. The reviewer read the amended receipts and independently decoded `portable.log.gz` to SHA256 `13f9960ba0721948d9f0f24eb2b14445cdb71c580d391116a82db5011ba740f5`, matching the original log and its recoverable `.flow/tmp/pack-sum-unknown-progress/portable.log` copy byte-for-byte. Every updated conductor-proof evidence hash and all three frozen source hashes verify. The staged whitespace check passes. Compression preserves the original negative-test output, including its trailing space; source and test inputs remain unchanged, so no Go rerun was warranted.
