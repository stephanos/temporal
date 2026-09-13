---
satisfies: [R2, R11]
---
# fn-87-tighten-the-testpilot-protocol-glossary.4 One protocol file per concept, documented messages, dense field numbers, closure check

## Description
Restructure the protocol into the spec's Target structure (R2): move messages into `case`, `value`, `expression`, `program`, `instruction`, `contract`, `correlated`, `event` and `run` files, give every message and every non-obvious field a leading comment, and renumber fields densely from 1. Add the two descriptor checks: every message carries a leading comment (R2) and the Case import closure excludes Run-only messages (R11). This is the last mechanical task; its proof is that the equivalence test passes with no new mapping step (file moves and renumbering do not change ProtoJSON). It closes the Early proof point.

**Size:** M
**Files:** `proto/internal/temporal/server/api/testpilot/v1/*.proto` (new `correlated.proto`, `event.proto`), `api/testpilot/v1/*` (regenerated; removed/added files), `model/lakefile.lean:14-52` (schema list), `Makefile:96-103` (`TESTPILOT_PROTOCOL_PROTOS`) and `umpire-check-testpilot-protocol` recipe, `model/Testpilot/Protocol.lean` (load `run.proto`'s closure beside `case.proto`'s), `common/testing/testpilot/protocol_test.go` (closure test, file list), `common/testing/testpilot/correlated_facade_test.go:53-58` (its descriptor closure is rooted at `run.proto` and names `CorrelatedEvidence`, which moves to `correlated.proto`; root it at `correlated.proto` or `case.proto`), new `common/testing/testpilot/protocol_comments_test.go`, `tools/umpire/internal/retiredvocabulary/check.go` (only if a scanned path moves)
**Touches:** [proto/internal/temporal/server/api/testpilot/**, api/testpilot/**, model/lakefile.lean, model/Testpilot/Protocol.lean, Makefile, common/testing/testpilot/protocol_test.go, common/testing/testpilot/protocol_comments_test.go, common/testing/testpilot/correlated_facade_test.go, tools/umpire/internal/retiredvocabulary/check.go]

### Approach
- Placement follows the spec table exactly. Notable moves: `FormatVersion` → `case.proto`; `Slot`, `Observation` → `program.proto`; the evidence-lift messages (`CorrelatedEvidenceProjection`, `CorrelatedEvidenceRule`, `CorrelatedEvidenceBinding`) → `correlated.proto` beside the correlated contract and `CorrelatedEvidence`; `RunEventKind`, `RunEventFilter`, `RunEventField` → `event.proto`, imported by `contract.proto` and `run.proto`; `FaultKind` and `FaultInjected`: the fault kind enum stays with `InjectFault` in `instruction.proto`, `FaultInjected` (a recorded fact) moves to `run.proto` (its comment "so the accessor stays in the file that declares the enum" goes). `contract.proto` must no longer import `run.proto`.
- Dense numbering: renumber every message's fields 1..n in declaration order (`CorrelatedRule` starts at 7 today, `CorrelatedEvidence` skips 5, `Program.environment` is 8). No `reserved` blocks: the wire has no compatibility promise and no binary Case is persisted. Reorder declarations so each message declares identity first, then what it does, then when it runs (spec Presentation), since .15 prints in declaration order.
- Comments: a leading `//` comment on every message and enum, and on every field whose meaning is not its name (every Limits field, `RequestAssignment`, `ActivationReservation`, `Slot`, `InvokeRpc`, ...). Preserve existing comments and api-linter directives verbatim when moving them.
- Comment check (R2): generated Go descriptors have no source info (`api/testpilot/v1/*.pb.go`), so the check reads a descriptor set built with `--include_source_info`. Follow the env-var pattern of `umpire-check-testpilot-authoring` (`Makefile:628-638`): the `umpire-check-testpilot-protocol` recipe runs protoc into a temp file and invokes `go test ./common/testing/testpilot -run '^TestProtocolMessagesCarryLeadingComments$'` with `TESTPILOT_PROTOCOL_DESCRIPTOR_SET` set. The test fails naming each message or enum without a leading comment, each message whose field numbers are not dense from 1, and each message whose declaration order differs from field-number order (so .15's declaration-order printing cannot drift). When the variable is unset it skips with a message naming the make target that runs it. Negative unit tests over synthetic `FileDescriptorProto`s cover each failure.
- Closure check (R11): in `protocol_test.go`, walk `File_temporal_server_api_testpilot_v1_case_proto`'s transitive `Imports()` (reuse the walk at `conformance_test.go:175-199`) and require that no file in it declares `Run`, `Verdict`, `RunDiagnostic`, `RunEvent`, `CleanupOutcome` or `FaultInjected`; fail naming the importing file. Payload messages added by .7 are added to this list there.
- Lean: `Testpilot.Protocol` loads `case.proto`'s closure; `Testpilot.Authoring` also builds Runs and Verdicts (`Authoring.lean:500-539`), so Lean must also load `run.proto`'s closure. Files both closures share (`value`, `event`, ...) must not be declared twice: probe whether a second `#load_proto_file` skips already-loaded files; if it does not, load the union another way the library supports (for example one load per root with the shared closure first) and record the choice. Update `model/lakefile.lean`'s `input_file` list and `Makefile`'s `TESTPILOT_PROTOCOL_PROTOS` to the nine files.
- `protocol_test.go:TestProtocolUsesCohesivePublicVocabulary` lists file paths; update to nine.
- Mapping: no step. If a fixture differs, the restructure changed meaning; fix the proto, do not add a step.

### Investigation targets
**Required** (read before coding):
- `proto/internal/temporal/server/api/testpilot/v1/*.proto` (after .2/.3)
- `model/lakefile.lean:14-52` and `.flow/memory` entry "Track schema inputs before reusing generated Lean modules"
- `model/Testpilot/Protocol.lean`
- `Makefile:96-103,620-638`
- `common/testing/testpilot/protocol_test.go`, `conformance_test.go:175-199`

**Optional:**
- `proto/internal/buf.yaml`, `proto/api-linter.yaml` — lint config the new files must pass under `make proto`

### Key context
- `make proto` runs `lint-protos lint-api`; new files need the same `option go_package` and api-linter file-layout directives the existing files carry.
- After this task, record in the done summary that the Early proof point holds (mechanical R1/R2 proven by the equivalence test with no semantic step) before .5 starts.

## Acceptance
- [ ] nine proto files match the spec's Target structure; `contract.proto` does not import `run.proto`; field numbers are dense from 1 in every message
- [ ] every message and enum has a leading comment; `TestProtocolMessagesCarryLeadingComments` runs from `make umpire-check-testpilot-protocol` and fails naming an uncommented message or enum, a non-dense message, or a message whose declaration order differs from number order (negative unit tests)
- [ ] the Case import-closure test fails naming the file that pulls a Run-only message (negative case proven by a synthetic closure or a deliberately wrong import in a unit test)
- [ ] Lean loads the Case and Run closures; lakefile and Makefile schema lists name the nine files; `make umpire-check-testpilot-protocol` and `make umpire-check-testpilot-authoring` green
- [ ] fixtures regenerate byte-identical except for any key order change (none expected); equivalence test passes with no new step; `make umpire-check-regression` exit 0 with nine live identities; `make lint-model` 163; `make lint-code` no new issues


## Done summary
Restructured the Testpilot protocol into the nine files of the fn-87 Target structure: `case`, `value`, `expression`, `program`, `instruction`, `contract`, `correlated` (new), `event` (new) and `run`. Every message and enum now has a leading comment, and field numbers run densely from 1 in declaration order. Two new checks enforce this: the comment/numbering check (R2) and the Case import-closure check (R11). **Early proof point: holds.** The equivalence oracle passes with no new mapping step, and every fixture regenerates byte-identical.

- Placement follows the spec table:
  - `FormatVersion` is in `case.proto`; `Slot` and `Observation` are in `program.proto`.
  - The evidence lift, the correlated contract and `CorrelatedEvidence` are in `correlated.proto`.
  - `RunEventKind`, `RunEventFilter` and `RunEventField` are in `event.proto`.
  - `FaultInjected` is in `run.proto`, and its "accessor stays in the file" comment is gone.
  - `InstructionOutcomeDefinition` and `OutcomeFieldDefinition` are in `instruction.proto`.
  - `contract.proto` no longer imports `run.proto`. `expression.proto` also imports `event.proto`, for `RunEventFieldRef`.
- Declaration order is identity, then behavior, then when it runs. Reordered messages:
  - `Case` (`case_id` first), `Program` (`environment` after `program_id`).
  - `InstructionNode`: id, instruction, outcome, reservations, dependencies, guard, limits.
  - `ContractTransition`: filter and predicate last. `ContractRule`: captures before transitions, deadline last. `Contract`: correlated before limits.
  - `Deadline`: `violation_state_id`, `rule_events`, `elapsed_milliseconds`.
  - `CorrelatedContract`: projection id and fingerprint first. `CorrelatedRule`: trigger, response, captures, correlation, then clock, bound, ending.
  - `CorrelatedEvidenceRule`: guard last. `CorrelatedEvidence` goes from 1-4,6 to 1-5.
  - The `Instruction` oneof numbers are unchanged, because `contract.Opcode` mirrors them.
- R2 check: `TestProtocolMessagesCarryLeadingComments` in `common/testing/testpilot/protocol_comments_test.go`.
  - It reads a `--include_source_info` descriptor set that `make umpire-check-testpilot-protocol` builds, and skips with a message naming that target when `TESTPILOT_PROTOCOL_DESCRIPTOR_SET` is unset.
  - It names each message or enum without a leading comment (api-linter directives do not count), each non-dense message, and each message declared out of number order.
  - `TestProtocolDocumentationProblemsNameEachViolation` covers each failure as a negative case. A mutated live descriptor also failed with the expected names.
- R11 check: `TestCaseImportClosureExcludesRunOnlyMessages` in `protocol_test.go`. It walks `case.proto`'s imports, and the negative case imports `run.proto` from a synthetic file. It was red before the restructure, naming `contract.proto`. `protocolFiles` is shared with the vocabulary test.
- **Decision: generator fix, outside the declared Touches.** `protogen` trims enum value prefixes only per file. A singular enum field that names an enum from another file of the same package therefore generated uncompilable Go, which is why `FaultInjected` had to sit beside `FaultKind`. No layout gives `RunEvent.kind` its enum while also keeping `RunEvent` out of the Case closure. `cmd/tools/protogen/enum_references.go` now rewrites those references after generation.
  - Tests: `enum_references_test.go`, red then green.
  - No other generated file in the repository changed.
  - This is recorded in the fn-87 spec's Planning decisions.
- **Decision: Lean loading.** A probe showed that a second `#load_proto_file` re-declares the shared closure (`google.protobuf.Any has already been declared`), and the library's `read_proto_files` is not public. `Testpilot/Protocol.lean` therefore uses a `run_cmd` that runs one `protoc` over `case.proto` and `run.proto` and feeds the library's `Versions.compile_proto`. `model/lakefile.lean` and `TESTPILOT_PROTOCOL_PROTOS` list the nine schemas, and the Lake stamp rebuilt `Testpilot.Protocol`.
- Other edits outside Touches, each needed to keep a gate green:
  - `common/testing/testpilot/temporal/catalog.go`: the Driver catalog now roots at both `case.proto` and `run.proto`, because `FormatVersion` left the run closure. The first regression run failed on `synthetic-case.json`.
  - `tests/testcore/testpilot/protobuf_lean_authoring_test.go`: pins `Program.environment` at 2, the declared renumbering.
  - `tools/umpire/cmd/umpire-gen-lean-api/case_schema_test.go`: its authority-field scan now covers the two new files.
  - `correlated_facade_test.go` (in Touches) roots at `case.proto` and depends on `correlated.proto`.
  - `retiredvocabulary/check.go` needed no change.
- Follow-ups:
  - Reviewer P3, not applied after SHIP: sort rewrite offsets explicitly and skip `SelectorExpr.Sel` in `rewriteEnumReferences`. It is safe today because `ast.Inspect` visits in source order and the map is package-local.
  - Several comments state runtime facts that later tasks change (`EnvironmentRef` scope, `version` "Always 1"); .5 to .12 should update them with their shapes.
- Gates:
  - Baseline was green.
  - Passing after the change: the oracle; the testpilot-protocol, testpilot-authoring, case-runtime-conformance and retired-vocabulary checks; and `make proto`.
  - `make umpire-check-regression` ran three times. Run 1 failed on the catalog, which was then fixed. Runs 2 and 3 exited 0 with 9 passing live identities, and run 3 was on the committed tree. No known flake fired.
  - `lint-code` shows 161 issues after `go clean -cache` and `lint-model` shows 163; both match the baselines.
- The commit also sweeps another session's uncommitted `.flow` edits for fn-87 .5 and .17, and it carries the spec's Planning-decision edit.

stage: impl-review - ran (claude backend, SHIP on first round, one P3)
## Evidence
- Commits: cc6868258ec76d60db14c699fce475974499ba3e
- Tests: baseline: green (oracle, testpilot-protocol/authoring/conformance, retired-vocabulary run pre-edit; regression via honored receipt b36208a9), go test -count=1 -tags test_dep ./common/testing/testpilot/internal/protocolmigration/, make umpire-check-testpilot-protocol umpire-check-testpilot-authoring umpire-check-case-runtime-conformance, make umpire-check-retired-vocabulary, go test -count=1 ./cmd/tools/protogen/, make proto (lint-protos, lint-api, regeneration byte-identical on rerun), CC=/usr/bin/cc TMPDIR=$(cd "${TMPDIR:-/tmp}" && pwd -P) make umpire-check-regression (exit 0, 9 passing live identities; run 1 failed on the Driver catalog missing FormatVersion, fixed; runs 2 and 3 green, no flakes), go clean -cache && make lint-code GOLANGCI_LINT_FIX=false (161 issues, baseline 161), make lint-model (163 errors, baseline 163)
- PRs: