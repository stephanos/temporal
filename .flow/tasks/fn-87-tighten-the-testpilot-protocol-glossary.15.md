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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
