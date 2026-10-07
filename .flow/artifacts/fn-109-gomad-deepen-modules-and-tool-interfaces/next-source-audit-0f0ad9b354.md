# Next source correction at 0f0ad9b354

Research-only inspection of `0f0ad9b35451056c85788efd12dbda8ec2bdf60f`, using the project research tier, gpt-6-astra at high. No Go, lint, native execution, production edits or task-state changes were performed. The conductor's integrated portable receipt remains separate evidence.

## Recommendation

Continue existing fn-113.1 R2 with a missing checksum on a **rule-only module**. R2 requires every unknown pin to remain actionable. The original task's Approach and Acceptance require unknown status and nonzero disposition; README lines 1103-1107 state the missing-sum contract. This follows delivery item 2 in MILESTONES, without creating another owner.

`upgrade/pinimpact/pinimpact.go:285-303` computes the candidate rule match, handles uncertainty only when `candidateActivation.unknown`, then filters out any rule whose baseline identity does not match. When activation is resolved and both baseline and candidate lack a rule module's zip checksum, the candidate match is unknown but the baseline filter emits `not-selected`. Default output suppresses the rule, permitting `Invalidated=false`. `modules.go:166-168` already supplies the correct `module_sum_missing` diagnostic; the defect is evaluation ordering.

The shipped `internal/compatibilitypack/packs/golang-x-sys-v047-darwin-arm64.json:1` proves rule membership need not equal activation membership. It activates only on `golang.org/x/sys@v0.47.0`, but includes `golang.org/x/term@v0.45.0`. The reviewed activation-sum correction does not cover this branch. Current `pack_sums_test.go` separately exercises independent rule exclusions only while an activation sum is missing.

## First regression and controls

Use public Evaluate, Encode and Render with real temporary module files. Reuse `loadPack`, `moduleFiles`, `withoutZipSum` and `goModResolver` from the existing pinimpact tests. Load the unchanged x/sys pack; derive the x/sys requirement from its activation and the x/term requirement from its rule (the existing `packModule` helper searches activation only). Require just these two modules at their checked identities. Remove only x/term's zip-sum line from both snapshots, retaining its `/go.mod` line and x/sys's complete identity. Do not modify pack governance, platform, pins or source inventories.

Static branch prediction, awaiting TDD execution: current code omits x/term in default output and reports it `not-selected` with IncludeAll. Expected behavior is exactly one unknown x/term rule with its existing missing-sum reason and `Invalidated=true`. The two x/sys rules remain unaffected. The absent x/crypto rule remains not-selected.

Controls must retain exact canonical/human bytes for fully resolved cases: all exact; baseline missing but candidate repaired; genuine rule absence; rule version, checksum or replacement mismatch; known activation exclusion. Compare candidate-only missing with both missing and baseline rule absent/bumped. Preserve baseline/candidate validation precedence, input snapshots and the existing activation-uncertainty matrix. A known activation exclusion must continue to take precedence over rule uncertainty.

Minimal production change belongs only in `evaluatePacks`: before its baseline filter, recognize candidate rule uncertainty when activation is known to match; retain the existing uncertainty branch and its reason precedence. Use the candidate rule's existing reason. No new resolver, API, profile calculation, selection policy or platform handling is needed.

Extend the public CLI file-proxy fixture in `cmd/gomadtool/pin_impact_test.go:22` with these two real module identities, `.mod`/`.info` records and matching `/go.mod` hashes. Its stock-Go graph listing remains offline. Assert text/JSON status 1 instead of 0, with unchanged module files. Later authorized focused command: `go -C tools/gomad3 test -tags test_dep -count=1 ./upgrade/pinimpact ./cmd/gomadtool -run 'TestMissingPackRuleSum|TestPackRuleSumControls|TestRunPinImpactMissingPackRuleSum|TestMissingPackActivationSum|TestPackActivationSumControls|TestKnownPackActivationExclusion|TestPackRuleExclusion'`. Retain actual RED before editing, then scoped and existing portable controls through the conductor's serialized lane.

## Other acceptance

Fn-109.3-.6 cannot be declared portable-only completions from the current maintenance receipt. Each task's ownership amendment retains R18 preservation, admission dependencies, lint, formal review and Darwin/full/affected gates; fn-109 R19 at spec lines 368-376 requires complete native Darwin and consumer evidence. Their historical checked acceptance and review are not current-source completion proof. Unchecked adapterregen lock release still lacks a lawful deterministic public failure fixture. Profile/source-inventory pack matching would require broader platform/evidence decisions and is not this recommendation. Original native comparators, full gates, formal acceptance and deferred fn-128 ownership remain unchanged; this receipt grants no SHIP verdict.
