---
satisfies: [R3]
---
# fn-125-represent-dynamic-configuration-in-the.2 Declare settings over finite domains in model/umpire, lift them and build one table per valuation

## Description
Implements R3 (spec Part A). Temporal-agnostic framework: a Model setting is a named constant over a `Finite` domain, read by step functions through a given `Valuation`. No Temporal Model reads one yet, so every existing IR file, table and Case stays byte-identical.

**Cross-spec entry gate:** fn-114 closed (paths after fn-114.9 and its Case freeze). Not concurrent with fn-124.8 (the `tools/umpire/model` package split); whichever lands second rebases onto the other's paths. Tasks 2 and 5 both edit `ir.proto`; the second takes the next free field numbers.

**Size:** L
**Files:** `model/umpire/` (new `Setting.scala` beside `Action.scala`'s `input[T]`, `Machine.scala`, plain-Scala tests); `model/irgen/**`; `proto/internal/temporal/server/api/umpire/v1/ir.proto` and generated `api/umpire/v1/*`; `tools/umpire/model/**` (IR reader, validation, machine build); lifter and Go fixtures; `model/SEMANTICS.md` (Settings section).
**Touches:** [model/umpire/**, model/irgen/**, model/check/**, model/ir/**, proto/internal/temporal/server/api/umpire/v1/**, api/umpire/v1/**, tools/umpire/model/**, tools/umpire/checker/**, model/SEMANTICS.md]

### Approach
- Scala: `setting[T](using Finite[T])`, mirroring `input[T]` (`model/umpire/Action.scala:90-176`), read as `s.value` under a `given Valuation`. Step functions, guards, `ends` and starts may read it; reading with no valuation in scope does not compile. Plain step-function tests supply a `given Valuation`. Settle spellings within `.plans/DSL_OPERATORS.md`.
- IR: a catalog message `Setting {id, name, position, domain}` on the IR file and an expression `setting(id)`. Default-empty fields keep existing canonical bytes and fingerprints; extend `TestSchemaRenameKeepsTheWireBytes` via `schemaSupplement` (procedure as in fn-112.11 / fn-118.2).
- Lifter: records the declaration and translates each read to `setting(id)`; refuses at its line a non-finite domain, a second declaration of one setting, and a read it cannot translate.
- Go: derive the settings each machine reads, transitively through calls, members and refinements, and build one table per valuation (product over the settings read). Composition members, a refining machine and its product see one valuation; a refinement is checked per valuation of the union of the settings both sides read. A machine that reads none builds exactly today's table.
- A fixture machine that reads one setting proves the path end to end (lift, IR, per-valuation tables with different transitions).

### Investigation targets
**Required:**
- `model/umpire/Action.scala:80-180` (`input[T]`, `Finite`); `model/umpire/Machine.scala`
- `model/irgen/` expression translation and its refusal reporting
- `proto/internal/temporal/server/api/umpire/v1/ir.proto` (expressions, IR file catalogs)
- `tools/umpire/model/machine.go`, `schema_test.go`
**Optional:**
- `.plans/DSL_OPERATORS.md`; `.plans/DYNAMIC_CONFIG.md` section 5 (TLA+ `CONSTANT`, Quint `const`, P `param`)

### Quick commands
```bash
make protoc && make umpire-gen-model && git diff --stat model/ir model/cases
make umpire-check-model
go test -count=1 -tags test_dep ./tools/umpire/model/... ./tools/umpire/checker/...
```

### Execution constraints
- Frozen where unbound: every existing IR file, table, key, total, answer, Definition ID and Case is byte-identical (baseline goldens while they exist, fn-124 R7).
- No Query binding yet (task 3); no Temporal Model reads a setting.

## Acceptance
- [ ] `setting[T]` over a `Finite` domain is declarable in `model/umpire` and readable in step functions, guards, `ends` and starts under a given `Valuation`; a read with no valuation in scope does not compile.
- [ ] The IR generator records `Setting {id, name, position, domain}` and each read as `setting(id)`; existing IR bytes and fingerprints are unchanged (schema test extended).
- [ ] Go derives the settings each machine reads, transitively, and builds a table per valuation; a fixture machine shows different tables under different values.
- [ ] A non-finite domain, a second declaration and an untranslatable read are refused at their line.
- [ ] Existing tables and Cases are byte-identical; `make umpire-check-model`, tooling tests and `make lint-code-fast` pass.


## Done summary
Blocked:
Blocked: deferred by the owner on 2026-10-05 together with the whole of fn-125 (dynamic configuration in the Models). Task 1 (the HSM/CHASM switch fixes) is done and merged; revive the spec to continue.
## Evidence
- Commits:
- Tests:
- PRs:
