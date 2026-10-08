---
satisfies: [R8]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.19 Enforce package coverage, host-effect and public-signature rules in the architecture checks (fulfils fn-105.4 D4)

## Description


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

Owner amendment (2026-10-04): this task transfers every remaining native Linux execution, Linux pack/report/replay and Linux-specific qualification-documentation requirement to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). Native execution/full/affected gates still owned here apply to Darwin. Missing transferred Linux proof cannot block this task. Static coverage of both supported source sets, shared implementation, preservation, review and other non-Linux requirements remain unchanged. Retained scope: Implementation, both-source-set static coverage, R18 preservation, admission dependencies, lint, formal review and Darwin/full/affected gates. See the [transfer manifest](../artifacts/linux-scope-transfer-2026-10-04.md). Historical progress below retains its original meaning and is not current-candidate proof.

Stage 6, R8 (F7). The architecture test lists a fixed set of roots and ignores standard-library and external imports, so a new top-level package or a host effect in a pure module goes unnoticed. This task is the single implementation owner of fn-105.4 (D4, origin brief `.flow/tasks/fn-102-gomad-architecture-consolidate.5.md`); close fn-105.4 by reference afterwards. It runs after the refactors so the rules describe the final owners.

**Size:** M
**Files:** `tools/gomad3/architecture_test.go`, negative fixtures under `tools/gomad3/testdata/architecture/` (new), possibly a small private checker package; record validation/preservation tests for the timezone effect; Runner inspection and target capability evidence projections, pinimpact's pack-directory intent and its pack-refresh consumer for confirmed public-type leaks. Retain intentional migrations in `go-interface-changes.md`.
**Touches:** [tools/gomad3/architecture_test.go, tools/gomad3/testdata/**, tools/gomad3/internal/gomadtool/**, tools/gomad3/internal/compatibilitypack/schema*.go, tools/gomad3/record/**, tools/gomad3/runner/inspect*.go, tools/gomad3/runner/replay_operation_test.go, tools/gomad3/target/capability*.go, tools/gomad3/upgrade/pinimpact/**, tools/gomad3/cmd/gomadtool/compatibility_pack_refresh*.go, tools/gomad3/world/*.go, tools/gomad3/world/process/session*.go, tools/gomad3/world/process/terminal*.go, tools/gomad3/README.md, tools/gomad3/ARCHITECTURE.md, .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/go-interface-changes.md]

### Approach
- Gaps: `listHostPackages` (`architecture_test.go:371-396`) passes thirteen explicit patterns to `go list`; `TestPackageArchitecture` skips every import outside the module (`:35-37`); several checks only assert file presence or absence (`TestCleanupRemovesSupersededFiles` `:58`, and similar).
- Discovery: enumerate every host package of the module (`./...`) for both qualified source sets (`GOOS=darwin GOARCH=arm64` and `GOOS=linux GOARCH=amd64`), with exact, justified exclusions for the runtime overlay, checked-in fixture source trees and separate qualification corpus module. Independently inventory Go source and nested `go.mod` files that package listing skips; retain explicit classifications for each nested module, including additions inside already excluded fixture trees. Required exclusions must match source/modules or fail stale-exclusion; only `.toolchain` and `.bin` are optional generated roots whose clean-checkout absence succeeds. Own the test-only module root without excluding future production root source. Reject included package-listing errors after filtering the exact overlay boundary. The concrete inventory and matcher controls are retained in `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-19/inventory-source-scout.md`. A package with no owner fails; prefix lookalikes, hidden/underscore source roots and unexpected nested modules cannot silently escape coverage.
- Host-effect rules are targeted, not a standard-library allowlist: name the pure modules and what they may not reach (`world`: no `os`, `net`, `os/exec`, `time.Now`, goroutine start; `runner/internal/campaign/controller.go` and the exploration frontiers: no host effects; the capability evaluator from task 11; the lifecycle owner from task 16; `record`). Mixed packages need file-level rules (`runner/internal/campaign` holds the pure controller beside effectful journals). The rule must catch an effect imported through the standard library or a dependency, on either platform's file set.
- Initialization boundary: inspect every module/dependency variable initializer and repeated init declaration in each pure root’s selected import closure, including blank, transitive and third-party imports. Stock process startup precedes those roots and is not claimed pure: require actual go-list Standard identity and a matching full immediate-Go-source directory pin for imported standard package startup. Missing or changed pins fail closed. This startup-only boundary never exempts callable functions, dependency initialization or lazy Local timezone initialization; retain their host-effect detection. See `task-19/initialization-boundary.md` for exact pinned sources and native-backed controls.
- Confirmed record purity defect: the three RFC3339Nano validation calls in `record/validation.go` use `time.Parse`, whose numeric-offset paths can initialize `time.Local`, reading TZ and timezone files and sampling the host clock. Use explicit UTC parsing through `time.ParseInLocation(..., time.UTC)` while retaining the accepted grammar, original strings and error precedence. Establish old-source architecture RED and preservation tests for all three fields, accepted instants, full parse errors, canonical bytes, hashes, signatures and decode round trips. The effect checker must reject `time.Parse` and distinguish explicit UTC from effectful location arguments; neither the time package nor all ParseInLocation calls are exempt. The pinned source chain and checker design are retained in `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-19/source-scout.md`.
- The same real effect is reachable from the pure capability evaluator through `SelectPacksForPlatform -> ValidatePack -> ValidatePackStructure -> validatePackGovernance`: `internal/compatibilitypack/schema.go` parses ReviewedAt with `time.Parse(time.RFC3339, ...)` before rejecting non-Z input. Admit only the companion explicit-UTC parse correction and schema preservation regressions. Preserve its existing UTC/Z admission, grammar, error classification/precedence and canonical pack bytes; require old-source transitive effect RED and corrected GREEN. This is no compatibility-policy or purity exemption. See `task-19/pack-governance-timezone-scope.md`.
- Reconcile the confirmed World callback defect without weakening the pure owner: `Recorder.FinishError(error)` currently invokes arbitrary Error/Is/Unwrap methods; formatting mutable exported sentinels is another callback path. Add a detached `Recorder.FinishTerminal(Terminal)` operation and closed projection for original World sentinels, concrete Capacity/ReplayDivergence errors and private model-generated classified wrappers. Keep owned detail/kind/cause data and immutable internal sentinel identities/messages; preserve ordinary Error/Unwrap/Is/As behavior, canonical records and all documented typed World inputs. Move general Error-then-errors.Is classification into the already effectful process reporting boundary, at its existing validation/cleanup position. Direct recorder custom-error/external-wrapper callback behavior and sentinel rebinding are explicitly reconciled: such inputs must be projected outside the pure core and cannot execute callbacks there. This intentional behavior/API migration must be inventoried before edits, never disguised as preservation or a purity exemption. Require actual old callback/effect RED, corrected no-callback/state-preservation GREEN, complete known-error/record-byte preservation, and reporting sequence/wrapper/join/precedence tests. See `task-19/world-terminal-design-decision.md`; do not expand other World modeling semantics or Runner production behavior.
- Public-signature rule: no exported function, method, field or interface of a public package may mention a type from an `internal/` package the consumer cannot import (the R5 defect). Extend the existing export walker (`packageExports` `:332`, `TestPublicPackagesDoNotExportTypeAliases` `:80`).
- Public API ownership stops at an importable foreign public named type/alias: preserve publisher-owned APIs such as Go 1.27.1's usable `json.RawMessage -> jsontext.Value -> jsontext.Options` sealing graph. Continue checking direct inaccessible named identities, all Gomad-owned exported/underlying/alias/embedding graphs, and foreign generic type arguments carrying private identities. This is not an internal-package or alias exemption. Exercise real external-consumer and checker-positive RawMessage construction/options, and retain negative controls for explicit foreign-internal exposure, Gomad aliases and private types in foreign generic arguments. See `task-19/foreign-public-boundary.md`.
- Confirmed signature repairs: Runner inspection fields expose private campaign journal/artifact plans (including named outcome leaves); target CompatibilityPackEvidence's defined RHS retains private governance/module/rule/adapter/source/linkname field identities; pinimpact.Spec.Packs returns private ValidatedPack values and has a production authoring-root override. Replace report fields with detached public value graphs and explicit projections, preserving field order/tags, all data, pointer presence, nil/empty slices, JSON/canonical bytes and policy ownership. Replace Packs with public PacksDirectory intent and migrate pack-refresh while preserving default loading, the explicit authoring root and validation-before-load/error precedence. Inventory these intentional Go changes before production edits; preserve all other public APIs. The exact source chains, consumers and selected corrections are in `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-19/public-signature-source-scout.md`. Require old-source checker RED, report/projection preservation and extended external-consumer compile-positive. New defined public types with builtin/public-only underlying graphs (choice.DiagnosticRecord and target's flat identity/enums) remain accepted; aliases and nested retained internal identities do not. Ordinary private fields remain encapsulated except public exposure through embeddings/promoted methods.
- Negative fixtures prove each rule rejects: an ownerless new root, a forbidden owner edge, a forbidden host effect in a pure file, and an inaccessible type in a public signature. Run the checker against the fixture and assert the specific failure; a fixture that merely exists proves nothing.
- Keep current ownership and edge checks (`packageOwner` `:398`, `ownerMayImport` `:440`, `moduleMayImport` `:468`, `TestExactModuleEdges` `:263`) including the owners added by tasks 7, 10 and 11. Valid platform-specific files (`*_unix.go`, `*_other.go`, `*_darwin.go`) stay accepted.
- Vet the complete validated host-package inventory through `TestHostPackageVet` for darwin/arm64, linux/amd64 and the actual host source set. The test runs on the actual host and applies target settings only to child package-list/vet commands. Every included package-listing error fails, and every discovered host import path reaches vet; no fixed root list or drop-on-error filtering. The original broad checkout `go vet ./...` commands fail on GOROOT-overlay inputs before edits, so those exact RED logs remain baseline evidence rather than a reason to remove host-package coverage. See `task-19/quick-gate-reconciliation.md`.

### Investigation targets
**Required:**
- `tools/gomad3/architecture_test.go` (whole file)
- `.flow/tasks/fn-102-gomad-architecture-consolidate.5.md`
- `tools/gomad3/ARCHITECTURE.md` sections "System boundary" and "World" (documented purity)
- `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/go-interface-changes.md` (public surface after R5/R13)
**Optional:**
- `tools/gomad3/internal/gomadtool/validation/` (existing Go-owned content checks)

### Quick commands
```bash
cd tools/gomad3
GOWORK=off go test -count=1 -tags test_dep .
GOWORK=off go test -count=1 -tags test_dep . -run '^TestHostPackageVet$'
make validate
cd ../.. && flowctl show fn-105-gomad-follow-ups-deferred-scope.4
```

### Constraints
- Follow MILESTONES verification instruction 5: commit each verified task separately, including implementation, tests, documentation and Flow records. Keep source-owned unavailable Darwin gates incomplete and acceptance open; transferred Linux gates remain open under fn-128; preserve unrelated changes and push only when authorized.
- No new third-party dependency. `tools/gomad3/go.mod` requires only `golang.org/x/mod`, so testify is unavailable inside `tools/gomad3`: follow the existing `t.Fatalf` style with whole-value comparisons there. In the root module (`tools/gomad3sim`, `tools/gomad3integration`) use `require` with `Equal`/`EqualValues`.
- Preserve existing comments with their owning code, CLI grammar/defaults, canonical bytes for fixed supplied identities, and error precedence/classification.
- Actual host is `linux/arm64`, not the inherited `darwin/arm64` assumption. Architecture/type/static-vet checks inspect both qualified source sets without native target execution; report that scope precisely. Required native Darwin qualification remains incomplete here; Linux qualification remains incomplete under fn-128.1/.4/.7. Neither can be inferred from stock-host or cross-vet results.
- fn-105 D12/D14 replay-divergence dispositions stay unchanged. Attribute a failure to those owners with retained evidence instead of relaxing an expectation.
- Run tests with `-tags test_dep`. Baseline the Quick commands before editing so a pre-existing failure is not attributed to this task.
- Evidence and decision records go under `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/`.
## Acceptance


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

Current native-execution acceptance is Darwin-only here. The corresponding Linux clauses and any older missing-Linux completion rule are transferred to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). All other acceptance below remains in force.

- [ ] Package discovery covers every host package on both qualified source sets with explicit, justified exclusions; an ownerless package or a stale exclusion fails.
- [ ] Targeted host-effect rules cover the named pure modules, including file-level rules for mixed packages, and catch effects reached through the standard library or a dependency.
- [ ] A public-signature rule rejects exported declarations that mention inaccessible internal types.
- [ ] Negative fixtures demonstrate rejection of an ownerless new root, a forbidden import edge, a forbidden host effect and an inaccessible public type, each asserting the specific failure.
- [ ] Existing ownership checks still pass; valid platform files and the explicit overlay/fixture exclusions remain accepted; no check relies on filename presence alone.
- [ ] fn-105.4 is closed by reference to this task (one owner).

## Source progress - bounded World lint correction (2026-10-05)

The task-owned World correction removes S1025 from `invalidSnapshot` and SA4006 from process Session Finish. It passes the builtin string directly to the existing classified-error constructor and keeps only the immutable recording header length at the existing bounds and slice positions. No public API, error text, canonical recording bytes, cleanup ordering, test, comment, policy or generator input changed.

Artifacts are retained under `../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-19/world-lint-progress-2026-10-05/`. Nine existing preservation controls pass before and after. The final ordinary World/process/mailbox run passes 45 top-level tests. Baseline/final ordinary root architecture suites, generator validation and three focused error-provenance/formatting controls pass on stock Go 1.27.1 linux/arm64. The conductor independently reran the ordinary three-package suite, nine controls and validation, verified both source hashes and all ten raw-log hashes, and reproduced the exact diagnostic comparison.

Pinned, unfiltered scoped lint remains exit 1. Findings fall from 28 to 26. Removing exactly the two resolved diagnostic blocks and updating their aggregate counts leaves a byte-identical final report with all 24 ST1005 and two forbidigo findings unchanged. The lint baseline supplies RED; behavior controls already passed before editing.

The fresh same-family Codex source-progress reviewer found no introduced Critical, Important or Minor issues and approved the source-progress commit. See `source-review.md`. This approval supplies no formal SHIP or native qualification.

This checkpoint retains task-18 dependencies and complete original R8/R18/R19, preservation, affected-consumer, current native Darwin/full/default/functional gates and formal review. Historical `sidecar_publish_failed` produced no verdict. Linux qualification remains deferred under fn-128 and does not block this source task.

stage: impl-review - skipped(policy: configured product lint remains red; fresh source-progress review is separate from formal SHIP)
stage: plan-sync - skipped(policy: planSync disabled and no task completed)

## Done summary
Blocked:
# Task 19 acceptance remains open

The source candidate is committed at `4a646710d15148f1a3cf75bdeba7bdfd2fd2edf7`.
Its round-four development gates and bounded independent corrective source review
are retained in `handover.md`, `evidence.json`, and
`round4-independent-source-review.md`. This is source progress, not formal SHIP
or supported-platform qualification.

Formal Codex fan-out on 2026-10-04 at 07:48:08Z failed before reviewer dispatch
with `sidecar_publish_failed`. Reservation
`28918d38c5ed438ea2933f6402f7a28d` was refunded. The Flow spec records a transport
failure with `round_consumed=false`, null verdict, zero task-19 rounds, and no
pending round or live reservation. No reviewer output or receipt exists.
The selected model was `gpt-6.1-sol` at high; no model executed that review.

`review-publication-debug.md` retains 40 successful original-helper scratch
replays. `review-handler-publication-debug.md` retains two successful isolated
CLI-handler metadata runs, with every mocked external boundary disclosed and
execution stopped before dispatch. These checks do not recover the exception
discarded by the original caller. The cause remains unreproduced. The installed
tool is unchanged; no actual review state, backend, model, sandbox or round
counter was reset or bypassed during diagnosis.

On 2026-10-04 the conductor freshly verified all 978 entries of
`round4-final-source.sha256`, all 109 entries of
`round4-task-owned-source.sha256`, and all 138 entries of
`round4-command-logs.sha256`. Every check exited 0. This establishes unchanged
source and retained local evidence, not a rerun of those commands or native
execution. The earlier omitted bulk logs remain local exactly as inventoried.
Flow validation also passed for all 22 tasks.

Keep task 19 and fn-105.4 acceptance open. Task 20 remains unclaimed, and task
21's current-tree comparison and final qualification are not admitted yet.
Resume formal review only after the failure's cause or relevant inputs change;
capture its original exception at the metadata boundary if an authorized real
invocation fails again. Do not substitute scratch metadata success for review.
Native darwin/arm64 and linux/amd64 gates, predecessor acceptance, and the
milestone's original requirements remain incomplete on this linux/arm64 host.

stage: impl-review - failed(sidecar_publish_failed: no draw dispatched; round refunded; no verdict)
stage: plan-sync - skipped(policy: no task reached accepted done)

Blocked:
# Task 19 acceptance remains open after guidance source admission

The task-19 source checkpoint is
`4a646710d15148f1a3cf75bdeba7bdfd2fd2edf7`. Development gates and the
independent source review/fix chain are retained under task 19. Formal fan-out
failed before dispatch with `sidecar_publish_failed`; no reviewer ran, no
receipt or verdict exists, and the reservation was refunded. The tool is
unchanged and the cause remains unreproduced. See `review-blocked.md` for the
original failure, diagnostics and ledger identities. No retry or reset occurred.

Native darwin/arm64 and linux/amd64 qualification, predecessor acceptance and
fn-105.4 closure remain incomplete. Connected GitHub inspection found historical
native runs only; none qualifies this unpublished source checkpoint.

On 2026-10-04 the conductor corrected its earlier successor restriction and
claimed task 20 for documentation-only source work under MILESTONES immediate
delivery item 4, after the integrated predecessor's independent source review
and fresh source-hash verification. The task dependency is retained; this is
not formal SHIP, task-19 completion, or an acceptance waiver. See
`../task-20/source-admission.md`. Earlier statements that task 20 is unclaimed
are historical. Task 21's current-tree comparison is not admitted yet.

stage: impl-review - failed(sidecar_publish_failed: no draw dispatched; no verdict; unchanged failure not retried)
stage: plan-sync - skipped(policy: no task reached accepted done)
## Evidence
- Commits:
- Tests:
- PRs:

## Current acceptance blocker (2026-10-05)

Task 19 has a bounded verified World lint source correction. Original task-18/predecessor acceptance, complete R8/R18/R19 preservation and affected-consumer requirements, current Darwin/full/default/functional gates and formal review remain open. Actual scoped lint remains exit 1 with 26 unchanged ST1005/forbidigo findings. The historical formal dispatch sidecar_publish_failed supplied no verdict. A fresh source-progress review does not replace formal SHIP. Missing transferred Linux evidence is not a blocker; fn-128 owns that qualification.
