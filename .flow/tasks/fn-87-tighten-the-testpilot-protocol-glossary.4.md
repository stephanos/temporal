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
TBD

## Evidence
- Commits:
- Tests:
- PRs:

