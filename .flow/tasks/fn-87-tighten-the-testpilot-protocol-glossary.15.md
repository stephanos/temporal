---
satisfies: [R15]
---
# fn-87-tighten-the-testpilot-protocol-glossary.15 Readable fixtures: declaration-order ProtoJSON, string field paths, named enum literals

## Description
The presentation half of R15: `Testpilot.ProtoJSON` emits fields in declaration order (identity first, which .4's declaration order already arranged); a field path is a string in a documented grammar, parsed at preparation; enum literals carry the value name. The absent-operand semantics are .16, split because they change evaluation and need their own Verdict check.

**Size:** M
**Files:** `model/Testpilot/ProtoJSON.lean`, `model/Testpilot/Tests/{ProtoJSON,ProtoJSONMain}.lean`, `proto/.../v1/{value,expression,program,instruction,correlated}.proto` (`FieldPath` → string fields; `EnumValue`), `api/testpilot/v1/*`, `common/testing/testpilot/internal/ir/{path.go,path_test.go,type.go,evaluate.go,runtime_value.go,read.go,write.go}`, `common/testing/testpilot/internal/execution/{dataflow.go,projection.go,request.go}`, `common/testing/testpilot/internal/verification/*.go` (paths in predicates), `model/Testpilot/Authoring.lean` (`Path`, `Value.enumeration`), `model/Umpire/Case/Projection/{Lowering,Coordinates}.lean`, Producers and `CaseSupport.succeeded` enum literal, `tools/umpire/cmd/umpire-gen-case-runtime-conformance/{generate.go,json.go}` (key-order check), fixtures, mapping, `common/testing/testpilot/README.md` (grammar section)
**Touches:** [model/Testpilot/**, model/Umpire/Case/**, model/Temporal/**, proto/internal/temporal/server/api/testpilot/**, api/testpilot/**, common/testing/testpilot/**, tools/umpire/cmd/umpire-gen-case-runtime-conformance/**, tests/testcore/testpilot/**]

### Approach
- Declaration order: the upstream printer builds a sorted `Std.TreeMap`/`Lean.Json` object (`model/.lake/packages/protobuf/Protobuf/Json/Codec.lean:961-967`), so no print option helps. In `Testpilot.ProtoJSON`, render the library's `Lean.Json` (or the `DynamicMessage`) with a descriptor-guided emitter that writes each object's keys in field-number order (which .4 made equal to declaration order), recursing through nested messages, repeated fields, maps and `Any` (well-known types keep their JSON forms). Keep compact output and the existing option semantics; the Go generator re-indents without reordering (`json.go:28`). Canonical: equal values produce equal strings (update the module docstring).
- Key-order check: the Go generator's `requirePersistedForm` path gains a check that every object's keys follow descriptor field order for its message type, naming file and JSON path; ordinary tests stay free of Lean (ART-12).
- Path grammar (document it in `common/testing/testpilot/README.md` and on the proto field): dot-separated protobuf field names in `snake_case`; `[*]` repeated wildcard; `["text"]`, `[42]`, `[true]` map keys typed by the map key kind (text keys JSON-quoted with JSON escapes); `?` suffix for the presence selector; `<name>` suffix for a oneof arm selection (spec example `attributes<nexus_operation_completed_event_attributes>.scheduled_event_id`, `history.events[*]`). One Go parser in `internal/ir/path.go` and one Lean printer in `Testpilot.Authoring.Path` (plus a Lean parser only if a Lean test needs round trips); a printer/parser round-trip test in Go over every segment kind, and a Lean `#guard` table printing each kind.
- Wire: every `FieldPath` field becomes `string` (name it `path`), and `FieldPath`/`FieldPathSegment`/selector messages are deleted. Preparation parses and rejects a path outside the grammar with the offending text and a located path (unit tests: unterminated key, unknown selector, map key of the wrong kind, empty segment).
- Enum literals: `EnumValue { string name = 1; }`; preparation resolves the name against the expected enum type from context (outcome status, observation field, capture type) and rejects an unknown name with the offending text, an enum name where the expected type is not an enum, and an enum literal with no type context (literal compared to literal); unit tests for each. Runtime values read from protobuf messages carry names too (one representation); record it.
- Lean: `CaseSupport.succeeded`'s `Value.enumeration 1` becomes the name `INSTRUCTION_OUTCOME_STATUS_SUCCEEDED` (if .11 left any use); Authoring `Value.enumeration` takes a name.
- Mapping: structured `FieldPath` objects → grammar strings; `enumValue.number` → name via the snapshot descriptors and the expected type; key order is not a ProtoJSON value difference, so no step for it. Retire `FieldPathSegment`, `RepeatedWildcard`, `MapKeySelector`, `PresenceSelector`, `OneofSelector` (if deleted).

### Investigation targets
**Required** (read before coding):
- `model/Testpilot/ProtoJSON.lean`, `model/.lake/packages/protobuf/Protobuf/Json/Codec.lean:940-1000,1400-1425`
- `common/testing/testpilot/internal/ir/path.go` (whole file)
- `tools/umpire/cmd/umpire-gen-case-runtime-conformance/json.go`
- `model/Testpilot/Authoring.lean:96-120` — `Path` constructors
- `model/Umpire/Case/Projection/Coordinates.lean` — the Lean walker that builds paths

**Optional:**
- `.flow/memory` entry "Portable schemas must preserve source semantic cardinality" (ProtoJSON pitfalls)

### Key context
- Every fixture changes on disk in this task; the equivalence test is what shows only presentation moved. List every fixture's before/after size in the done summary.

## Acceptance
- [ ] `Testpilot.ProtoJSON` emits every object's fields in declaration order; the generator rejects a fixture whose key order differs, naming file and path
- [ ] field paths are strings in the documented grammar; a path outside it rejects at preparation with the offending text (unit tests per error); Go round-trip and Lean printing tests cover every segment kind
- [ ] enum literals carry names; an undeclared name, a name on a non-enum type and an untyped enum literal reject at preparation with the offending text (unit tests)
- [ ] equivalence test passes with declared path and enum steps; Verdict pins unchanged; retired tokens added
- [ ] `make umpire-check-regression` exit 0 with nine live identities; `make lint-model` 163; `make lint-code` no new issues


## Done summary
Fixtures now read as reviewed artifacts. Every message object is written in declaration order. Field paths are strings in a documented grammar, parsed at preparation. Enum literals carry value names. The equivalence oracle shows that only presentation moved, and the Verdict pins are unchanged.

**Protocol**
- The four `FieldPath` fields are now strings: `PathExpression.path`, `RequestAssignment.target`, `ResponseRead.path`, and `CorrelatedEvidenceRule.operation`.
  - The names `target` and `operation` were kept rather than renamed `path`: each names what its path addresses.
  - The grammar is documented on `PathExpression.path` and in `common/testing/testpilot/README.md`.
- Deleted: `FieldPath`, `FieldPathSegment` and the four selector messages.
- `EnumValue` is now `{ string name = 1; }`, with an api-linter `core::0123::resource-annotation` suppression.
- `instruction.proto` no longer imports `value.proto`.

**Go runtime**
- `ir` parses and prints paths in one grammar (`path_syntax.go`). A segment takes at most one selector: `<member>`, `[*]`, `["text"]`, `[42]`, `[true]`, or a final `?`.
- `BindPath(source, location, text, limits)` locates every rejection at the path's field and quotes the whole text. That covers grammar errors, unknown fields or members, and keys of the wrong kind.
  - A payload-arm rejection moves from `...path.path.segments[0].field` to `...path.path`.
- Presence facts are keyed by the path's canonical spelling.
- Enum literals are resolved by name against the expected enum. Three cases reject, each quoting the name:
  - an undeclared name (`unknown`);
  - a name where the expected type is not an enum (`type_mismatch`);
  - an untyped literal (`type_mismatch`).
- Runtime values read from messages carry names too, via `ir.EnumValue` and `ir.EnumName`. A number the enum does not declare is spelled in decimal.
  - Temporal's generated `String()` is camel-case, so descriptor names are used instead.
- The worker Driver compares binding targets as strings.

**Lean**
- `Testpilot.ProtoJSON.canonical` now works over any generated message. It re-emits the library's key-sorted JSON in declaration order:
  - map fields keep the library's key order;
  - `Any` writes `@type` first;
  - well-known types keep their own JSON forms.
- The correlated corpus writes each row by hand, `name` first.
- `Testpilot.Authoring.Path` has `Key`, `Selector` and `Segment`, and `make` is the one printer. `Path.oneofSelector` became `Path.oneofMember`. `Value.enumeration` takes a name.
- `Coverage.scalarValue` and lowering read enum names from the schema's hex descriptor bytes (`Coverage.enumValueName`). `CaseSupport.succeeded` had already been removed by .11.
  - That reader and the key escaping walk bytes by hand. The protobuf decoder and `String.toList` depend on `Classical.choice`, and `Umpire.Case.Projection.lower` keeps its pinned `[propext, Quot.sound]` inventory.
- Producers spell `HISTORY_EVENT_FILTER_TYPE_CLOSE_EVENT` and the fault kind names.

**Tests**
- Go:
  - `TestPathGrammarRoundTripsEverySegmentKind` round-trips every segment kind.
  - `TestPathsOutsideTheGrammarRejectWithTheirText` covers an unterminated key, unterminated selector, unknown selector (bracket and character), a map key of the wrong kind, an empty segment, a trailing separator, and an unterminated oneof member.
  - `TestPathKeysTakeTheirMapKeyKind` checks key typing.
  - `TestEnumLiteralsNameADeclaredValue` covers the three enum rejections.
- Lean:
  - A `#guard` table in `Testpilot/Tests/Authoring.lean` prints each segment kind.
  - ProtoJSON tests assert declaration order, `Any` order and enum names.
  - `TypedUnary` guards pin enum naming and its rejection.
- Generator: `requireDeclarationOrder` and `requireCorrelatedDeclarationOrder` reject reordered objects, naming the file and the JSON path. Tests cover a top-level field, a nested field, `Any`, an undeclared key, a map value, and a correlated event.

**Oracle**
- New `Resolve` step kind, which receives the baseline snapshot.
- R15 enum step: names each baseline number by the snapshot enum its context expects. The contexts are request-assignment target fields, instruction-status comparisons, and payload-path comparisons. It requires the current enum to declare the name and fails on any other context. It is declared before the R9 step, which decodes the success guard into the current `Expression`.
- R15 path step: spells baseline paths with its own printer.
- `cloneTree` and the fault-coordinate builder keep snapshot annotations.
- With either step disabled, the oracle goes red.
- Key order needs no step.

**Other**
- Retired tokens: `FieldPathSegment`, `RepeatedWildcard`, `MapKeySelector`, `PresenceSelector`, `OneofSelector`.
- The fn-87 Planning decisions record ".15".
- Out-of-Touches edits: `tools/umpire/internal/retiredvocabulary/check.go` and the `.flow` spec.

**Fixture sizes (bytes, before → after)**

| Fixture | Before | After |
|---|---|---|
| typed-nexus | 58,824 | 47,519 |
| async-nexus | 26,092 | 22,107 |
| worker-outage | 17,706 | 13,801 |
| typed-unary | 15,211 | 11,001 |
| get-system-info | 3,458 | 3,287 |
| synthetic | 2,311 | 2,311 |
| correlated.json | 357,160 | 356,224 |

- Conformance `case.json` sizes did not change (cleanup-failure 2,983; cross-run 2,601; inconclusive 2,567; satisfied 2,551; static-rejection 2,586; expression-context 2,814; violated 2,545). Their key order changed.
- Every `expected.json` is byte-identical.

**Gates**
- Baseline: green via receipt c7255198.
- `make umpire-check-regression`, at c8ab88ed:
  - run 1 was red with known flake (c): `TestTestpilotAsyncNexusCase` was INCONCLUSIVE instead of SATISFIED;
  - run 2 was green with 9 identities.
- `make umpire-check-regression` at 34298eb8: green with 9 identities.
- `lint-code` at 35234ca5: 161, the baseline. That commit only regroups one test file's imports; the generator package tests pass there.
- `lint-model`: 163, the baseline.
- Protocol, authoring, conformance and vocabulary checks passed.

stage: impl-review - ran (claude backend, SHIP on the first round; P3s applied in 34298eb8: identity path shims inlined, generator order tests for map values, undeclared keys and correlated events, cross-reference comments)
## Evidence
- Commits: c8ab88edb9f5c94e037f1ffbb3b64b2e046c6472, 34298eb81671b580c22ad9a2dd816179370bf2cb, 35234ca537bf215f0f3498d0ea1c211430b16678
- Tests: baseline: green via receipt c7255198 (regression), go test -count=1 -tags test_dep ./common/testing/testpilot/internal/protocolmigration/, CC=/usr/bin/cc TMPDIR=$(cd "${TMPDIR:-/tmp}" && pwd -P) make umpire-check-regression (c8ab88ed: run 1 red, known flake (c) TestTestpilotAsyncNexusCase SATISFIED expected got INCONCLUSIVE; run 2 green, 9 identities; 34298eb8: green, 9 identities), go clean -cache && make lint-code GOLANGCI_LINT_FIX=false (161 at 35234ca5), make lint-model (163), make umpire-check-testpilot-protocol umpire-check-testpilot-authoring umpire-check-case-runtime-conformance, make umpire-check-retired-vocabulary, go test -count=1 -tags test_dep ./tools/umpire/cmd/umpire-gen-case-runtime-conformance/ (35234ca5, import regroup only)
- PRs: