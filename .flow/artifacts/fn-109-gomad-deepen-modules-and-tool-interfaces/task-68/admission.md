# Explicit scripted Choice Exploration calls

Root admits exactly three test-call attachments selected in [post-task66-next-slice.md](../../fn-112-gomad-determinism-assurance-and-test/task-10/runner-preparation-design/post-task66-next-slice.md), SHA-256 `b2e54c6b48f714e53e83e4f0f129c025346c6db9d06da7899e56c576e0934033`. BASE is `95b6abcad1cdfa37a7eb4287f4233130a46c59ba`; its product inputs equal reviewed primary `41727a2ce2ac6a59562db7f62613c06d7f30502b` and observed candidate `f699252450b8e67f1edb50ed8e4cff4cb6e644c0`.

The combined-66/67 packet records 673 named Runner outcomes: 389 pass, 272 fail and 12 skip. Exactly ten admitted originals improved; every other original stayed unchanged. Original-base configured lint retains 50 complete findings after exactly two task-67 spin corrections, with zero introduced findings. The three selected tests still fail at unsupported-host preparation. No future behavioral pass is inferred from that diagnostic.

This corrective owner supports open fn-109.63 and fn-112.10 source requirements. It does not depend on completion of still-red fn-109.66/.67. It grants no later simulation/storage admission or dependency shortcut. Root selects the grounded recommendation under the user's autonomous instruction and retains integration, source review, lifecycle and acceptance ownership.

## Product boundary

Only `tools/gomad3/runner/runner_test.go` may change, with one assignment after final configuration and before the existing call in each function:

- `TestRunChoiceExplorationExecutesRootAndEveryNonSelectedRank`
- `TestRunChoiceExplorationDivergingPrefixRetainsCompletedRound`
- `TestRunChoiceExplorationExpandsCompleteTargetFailures`

Use `configDependencies = scriptedPreparationDependencies(t, config.Preparer, configDependencies.executor)`. Preserve the outer divergence executor, final preparer, matching target/arguments, copied prepared bytes and verification. The helper and all existing setup, imports, comments, assertions, fixture data, channels, deadlines, cleanup, trace records and error paths remain unchanged. Removing only these three new assignments in the named functions must reproduce BASE byte-for-byte, including all sixteen preexisting task-65/66 attachments.

The original assertions remain real: root and rank-1 prefix requests, two immutable round commits and corrupt-segment rejection; four divergence-case attempts, three successes, one divergence, two committed rounds and no candidate artifact; and two failed executions with two retained artifacts under PolicyAll. No additional seam or test is needed for the admitted fixture attachment.

No production, public API, helper, default configuration, bootstrap decoding, replay/resume/minimize, guidance, simulation, storage, runtime, adapter, toolchain, generation or lint-policy change is admitted. No blanket fixture migration, encoded-output compatibility requirement, target metadata coercion, assertion relaxation or inferred executor dispatch is authorized.

## Verification and failure boundary

Before editing, retain the three unchanged-source preparation failures using pinned Go/tools, test_dep and count=1. Bind actual source, tools, environment, argv, exit, elapsed time and immutable raw outputs. A compile failure, missing tool or premature timeout supplies no meaningful preparation RED.

After exactly three attachments, observe all original behavioral assertions and retain the task-65/66 originals, preparation forwarding/error/stage/default/isolated controls, real preparation error/cancellation/local-phase controls and public profile guard. A newly reached original failure is new evidence for a separately admitted correction, never permission to alter this scope.

Retain formatting, exact byte preservation, affected vet/errortype, configured unfiltered Runner lint and required repository fast lint with pinned tools and no fixes. Generated validation and both supported source-set static checks require current execution or exact equality of their actual consumed inputs. Reference unchanged prior evidence rather than copying bulk histories; no whole-fingerprint PASS extrapolation is allowed.

Root serializes shared Go/build/lint/generator execution. At the next frozen batch, compare all 673 ordinary named outcomes against combined-66/67, admitting only the three selected originals' FAIL-to-PASS transitions and no other changed or missing original. Compare complete original-base lint blocks against RED50, requiring no introduced or removed finding. Reconciliation against older RED52 permits only task-67's two already-reviewed removals. Preserve actual aggregate nonzero exits and integrated errortype reachability.

Retain fresh independent integrated source review and a separate verified-progress checkpoint. Task-owned source requirements stay open while red; this is not formal SHIP or completion. Native full-host/runtime/soak evidence remains deferred and unverified under fn-128/fn-149. This admission grants no native revival, CI, PR or push authority.
