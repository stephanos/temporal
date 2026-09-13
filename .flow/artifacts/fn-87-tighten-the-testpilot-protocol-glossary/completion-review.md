# fn-87 completion review

Spec: fn-87-tighten-the-testpilot-protocol-glossary — Tighten the Testpilot protocol: glossary names,
one expression language, structure by concept
Tasks: 17 done / 17 total; every task review SHIP, plan review SHIP on round two.
Reviewed tree: the `claude/umpire-order-spec-0ylk31` checkout at 7093e13 plus this session's fn-87
follow-up commit.

**Verdict: SHIP**, with the four follow-ups below closed in this session and two observations recorded
rather than fixed.

This review was conducted in-session by reading the spec's fifteen requirements against the tree and
re-running every gate that does not need a live cluster. `flowctl` is not installed in this cloud
session (no flow-next plugin install and no marketplace source recorded), so there is no
`completion-review.json` receipt beside this file and no recorded review backend; the verdict is this
document.

## Requirements coverage

| R-ID | Status | Evidence |
| --- | --- | --- |
| R1 | met | `run.proto` `RunDisposition`; `correlated.proto:28,88` `rules`/`rule_id`; `instruction.proto:50,63,66,67` `ResponseRead`, `ReadCardinality`, `ReadTarget`; `program.proto:48` `OpaqueHandleType`, `instruction.proto:107,114` `handle_slot_id`; retired compounds registered in `tools/umpire/internal/retiredvocabulary/check.go:607-660` with `make umpire-check-retired-vocabulary` clean |
| R2 | met | nine files, one per concept (`case`, `contract`, `correlated`, `event`, `expression`, `instruction`, `program`, `run`, `value`); `common/testing/testpilot/protocol_comments_test.go:25` requires a leading comment on every message and `:45` pins the diagnostic |
| R3 | met | `expression.proto:11,76` one `Expression` over one `Reference`; `:64,65` `COMPARISON_OPERATOR_EQUAL`/`NOT_EQUAL`; no `guard_equals_text` anywhere in `proto/` |
| R4 | met | `run.proto:44` `oneof payload`; `expression.proto:125,132` the payload reference a Contract path reads through |
| R5 | met, with a recorded deviation | `contract.proto:87-95` `Deadline` with `violation_state_id` and the `bound` oneof. Deviation: presence stays single-arm oneofs rather than proto3 `optional`, because the pinned `protoc-gen-go-helpers` rejects `optional`; recorded in the spec's Planning decisions |
| R6 | met | `value.proto:14` `unsigned_integer_value` with `:96` `SCALAR_KIND_UINT64` and no `natural`; no wire `EntrypointKind` in any protocol file, with the Go classification in `common/testing/testpilot/contract/profile.go:23-34`; the wire-encoding scalar kinds are kept with the reason on `ScalarKind` |
| R7 | met | `common/testing/testpilot/README.md:76-230` — "Extending the protocol", a section per extension point, and `FAULT_KIND_WORKER_STOP` traced through every place as the worked example |
| R8 | met | `common/testing/testpilot/internal/protocolmigration`: 59 declared mapping steps and two declared additions (counted from `Declared`/`Added`), `TestBaselineFixturesMapToRegeneratedFixtures` green over all twenty baseline fixtures; the offline gates below are green |
| R9 | met, as amended | `instruction.proto:17` `After after = 3;`. The amendment (dependencies within one entrypoint only) is recorded in the spec's Planning decisions and justified by preparation already rejecting cross-entrypoint dependencies |
| R10 | met | `internal/execution/prepare.go:344,466-471` resolve the derived binding graph and reject a reference outside it; `case.proto` writes no environment list, reservation or outcome field |
| R11 | met | `common/testing/testpilot/protocol_test.go:30` `runOnlyMessages` over the descriptor set keeps Run-only messages outside the Case closure |
| R12 | met | the ceilings live on the Profile's `Limits` (`program.proto:108,110`), which `case.proto` does not reference; the SEM-16 amendment is drafted in `UMPIRE4_SPEC.md` pending GOV-02 |
| R13 | met | `case.proto:22-31` `CaseProvenance` with typed `definitions`, `sources`, `known_gaps`, `correlated_rules`, `local_names` and `model_value_fingerprints` rows |
| R14 | met | `local_names` and `model_value_fingerprints` carry the mapping; `tests/testcore/testpilot/testdata/typed-nexus-case.json` is 43,215 bytes against the baseline's 315,914 — 266 KB saved against the 244 KB the requirement asks for |
| R15 | met | declaration-order ProtoJSON, string field paths and named enums in the regenerated fixtures; `internal/ir/evaluate.go:124` states the absent-operand rule, and the Producer keeps a presence check beside a negated comparison per the recorded decision |

Unaddressed R-IDs: none.

## Gates re-run for this review

| Gate | Result |
| --- | --- |
| `go test -count=1 -tags test_dep ./common/testing/testpilot/...` | pass (12 packages) |
| `go test -count=1 -tags test_dep ./tools/umpire/... ./cmd/tools/protogen/...` | pass |
| `make umpire-build-model` | 595 jobs, success |
| `make umpire-check-lean-api umpire-check-goldens umpire-check-regression-views umpire-check-testpilot-protocol umpire-check-testpilot-authoring umpire-check-case-runtime-conformance umpire-check-inventory umpire-check-retired-vocabulary` | pass |
| `make umpire-check-live-tests` | see the order document's gate baselines for this session's run |

## Follow-ups the spec left open, closed here

1. **`ir` no longer calls a path read a projection.** `Operator.Project` is `ReadPath`; the compiler's
   `project`, `projectPayload` and `projectOperand` are `readPath`, `readPayloadPath` and
   `readOperandPath`; the runtime's `project` is `readPath`; `admission.projectionWork` is
   `pathReadWork`; the diagnostic reads "reference or path read requires an explicit presence guard",
   with its four pins updated. `MaxProjectionWork` and the correlated `ProjectionRules` keep the word:
   there it means `Umpire.Case.Projection`.
2. **Hand-written Go no longer calls an Opcode a capability.** `testpilot.InstructionCapability` is
   `InstructionOpcode` (retired in the vocabulary gate), `opcodeContext`'s parameter, the
   `programUsage` loops and `DeriveProfile`'s local are `opcode`, `MaxOpcode`'s comment says Opcode,
   and the two `unknown capability` diagnostics and `policy.capabilities` say opcode. The README's
   third sense ("a `Contract.correlated` capability") now says Correlated Contract.
3. **The frozen baseline tree is a `.gitignore` negation, not `git add -f`.** `.gitignore:21-22`
   negates `/common/testing/testpilot/internal/protocolmigration/testdata/`, matching the convention
   of the other testpilot negations, and the package README says so.
4. **protogen's deferred P3.** `rewriteEnumReferences` sorts by offset explicitly instead of relying
   on `ast.Inspect`'s visit order, and the collection skips a `SelectorExpr.Sel`, where the
   package-local trimming this rewrite reverses does not apply.

## Observations, not fixed

- **The Driver contract still says capability in the effect-handle sense** (`OpaqueCapability`,
  `CapabilityEffect`, `CapabilityBridge`, `InvokeCapability`, `NewCapability`, the server Session's
  `opaqueCapability`, `capabilitySlot` and `capabilityClaim`). The rename table retires those names on
  the wire and says the runtime concepts say effect handle, but fn-87 carved the hand-written Driver
  seam out and the vocabulary gate records `CapabilityBridge` and `CapabilityEffect` as live. Renaming
  it touches the server, worker and composite Sessions, every test Session, the conformance corpus and
  Umpire's lowering, so it belongs in its own change rather than in a closeout.
- **`make umpire-check-retired-vocabulary` takes about twelve minutes** on a four-core session: 353
  compiled rules are matched against every line of every scanned tree, the frozen baseline fixtures
  included (they are exempt from violations but still read and scanned). It is correct and it is the
  slowest offline gate by an order of magnitude.

## Defect found and fixed while re-running the gates

`tools/umpire/regression/ci_workflow_test.go`'s `listTestpilotDependencies` decoded
`go list -deps -test -json` from `CombinedOutput`, so any line `go` writes to stderr lands in the JSON
stream. On a cold module cache the download progress made `TestTestpilotOwnsCaseProtocolAndRuntime`
fail with `decode shared Driver dependency closure: invalid character 'g' looking for beginning of
value` — a green tree failing `make umpire-check-regression-views` for an environment reason. It now
reads stdout alone and reports stderr only in the error. This is pre-existing, not an fn-87 defect.
