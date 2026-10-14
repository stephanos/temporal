# Review: `cmp/go/` (Go sample)

## Scores (1-5, 5 best)

1. **Spec fidelity: 4.** Both Models are complete, in the Lean section order with comments preserved, the revised Model 2 is applied (`productOf` maps `PauseRequested` to `ProductStarted` at `standaloneactivity/standalone_activity.go:591`; visible retry rows at `standalone_activity.go:231-234`), and all pins are present. Spec names are exact only as `Name:` strings; Go identifiers drift by necessity (`NexusProduct`, `ProductScheduled`, `ResolutionSucceeded`, `PhasePolling`, `AttemptCompleted`), which weakens side-by-side reading of enum arms.
2. **Language plausibility: 5.** Every construct checks out for Go 1.24: generic type alias `Composed[S]` (`umpire/umpire.go:457`), `tool` directive in `go.mod:13`, `reflect.TypeFor` (`umpire.go:531`), interface types satisfying `comparable`, all type-parameter inference in `Bind1`/`Bind3`/`Refines`/`Steps` resolvable from arguments, map literals keyed by interface. No invented syntax found.
3. **Authoring readability: 3.** Machines, Sets and Scenarios read as configuration (struct literals with named fields); Properties are 7-9 lines of braces for a 4-line Lean claim, and each payload-carrying enum costs an interface, N structs, N marker methods, a `//sumtype:decl` line and a hand-listed `umpire.Sum(...)` (`nexuscaller/nexus_caller.go:58-80`). Enum prefixes (`ProductNexusOperationCompleted`) are noise a reader must strip.
4. **Check story accuracy: 5.** The README table (`README.md:68-82`) correctly puts undeclared action, step/action signature mismatch, scenario input arity and protobuf message existence at compile time, exhaustiveness at lint time (explicitly not `go vet`), and every semantic check (domain membership, evidence, refinement, query answers, pins) at `go test`. No overclaims found.
5. **Framework realism: 4.** `umpire.go` gives typed `Step`, `Machine`, `Refines`, `Property`, `Scenario`, `Limits`, `Query`, `Set`, `Compose`, `Restrict`, `At`, `Sync` with a precise statement of the mapped-state refinement rule (`umpire.go:237-239`) and a `Table`/`Answer` shape; enumeration is a plausible reflection walk (`Fields`, `umpire.go:78-81`). Search, table build and refinement bodies are `panic("sketched")`, and the `Claim`/`Path` erasure is a real, admitted gap.
6. **README honesty and library table: 5.** Costs are stated plainly (double length, everything semantic at test time, names written two or three times). All five spot-checks match: rapid `2026-09-04` not archived, go-check-sumtype `2026-09-20`, exhaustive `2026-09-13`, hashstructure archived `2023-01-03`, cespare/xxhash `2024-07-03`.

## Verbatim snippets

**`handlerReplyStep` (Nexus product)** at `nexuscaller/nexus_caller.go:210-234`
```go
func handlerReplyStep(state ProductState, reply Reply) []ProductStep {
	if state.Phase != ProductScheduled {
		return nil
	}
	switch reply := reply.(type) {
	case SyncSuccess:
		return productStep(ProductSucceeded, ProductNexusOperationCompleted)
	case Async:
		return productStep(ProductStarted, ProductNexusOperationStarted)
	case OperationFailed:
		return productStep(ProductFailed, ProductNexusOperationFailed)
	case OperationCanceled:
		return productStep(ProductCanceled, ProductNexusOperationCanceled)
	case HandlerError:
		if reply.Retryable {
			return nil
		}
		return productStep(ProductFailed, ProductNexusOperationFailed)
	default:
		panic(fmt.Sprintf("unhandled Reply %T", reply))
	}
}
```

**`syncSucceeds`** at `nexus_caller.go:623-630`
```go
var syncSucceeds = &protocolProperty{
	Name:    "syncSucceeds",
	Machine: NexusProtocol,
	When:    handlerReply.With(SyncSuccess{}),
	Holds: func(step ProtocolStep) bool {
		return step.State.Phase == Succeeded && step.Records(NexusOperationCompleted{})
	},
}
```

**`syncReplied` and `syncCompletion`** at `nexus_caller.go:724-732` and `:829`
```go
var syncReplied = &protocolScenario{
	Name:   "syncReplied",
	Model:  NexusProtocol,
	Starts: unscheduled,
	Actions: []umpire.Class{
		schedule.With(common.Unset, common.Unset, common.Unset),
		handlerReply.With(SyncSuccess{}),
	},
}

syncCompletion = &umpire.Query{Name: "syncCompletion", Find: syncSucceeds, In: syncReplied, Limits: two}
```

**`nexusCaller` compose** at `nexus_caller.go:926-945`
```go
var HandlerWorker = umpire.Restrict("handlerWorker", worker.Polling, worker.WorkerStop, worker.Serve)

type NexusCallerState struct {
	Operation ProtocolState      `umpire:"operation"`
	Worker    worker.WorkerState `umpire:"worker"`
}

var NexusCaller = &umpire.Compose[NexusCallerState]{
	Name:    "nexusCaller",
	For:     []*umpire.Entity{operation, worker.Worker},
	Members: umpire.Members{"operation": NexusProtocol, "worker": HandlerWorker},
	Sync: umpire.Sync{
		workerStop:   {"operation": workerStop, "worker": worker.WorkerStop},
		handlerReply: {"operation": handlerReply, "worker": worker.Serve},
	},
	Starts: []NexusCallerState{{Operation: unscheduled, Worker: worker.WorkerState{Phase: worker.PhasePolling}}},
	Ends:   func(s NexusCallerState) bool { return terminalPhase(s.Operation.Phase) },
}
```

## Line counts

| File | Lines |
| --- | --- |
| `go.mod` | 16 |
| `README.md` | 240 |
| `umpire/umpire.go` | 557 |
| `common/common.go` | 21 |
| `worker/worker.go` | 104 |
| `nexuscaller/nexus_caller.go` | 979 |
| `nexuscaller/pins_test.go` | 112 |
| `standaloneactivity/standalone_activity.go` | 988 |
| `standaloneactivity/pins_test.go` | 82 |
| **Two Model files** | **1967** |

The Lean reference is 777 lines, so the README's "roughly twice" for Model 1 (979) is accurate.

## Red flags

- **Identifier names do not match the spec**, only `Name:` strings do. `ProductScheduled`/`Scheduled`, `ResolutionSucceeded`, `AttemptCompleted`, `PhasePolling`, exported `NexusProduct`. The README explains why (`README.md:28-32`) but the rubric's exact-name requirement is only met at the string level.
- **The `finite` generator is referenced but absent.** Every `//go:generate go run ../umpire/cmd/finite` line (`common.go:5`, `worker.go:14`, `nexus_caller.go:31`, `standalone_activity.go:24`) points at a command that does not exist in the sample, while the library table recommends `dmarkham/enumer` plus a template instead (`README.md:220`). Without it, `Fields[ProtocolState]()` fails at `Check` because `Phase` and `common.Timeout` have no `Values()`.
- **"Protobuf message named by an action exists: compile"** (`README.md:82`) is true for existence, but the name string itself is produced at package-init time via `proto.MessageName`; a decision maker should not read it as the schema being checked against the action's inputs.
- **Query pairing is untyped.** `Query.Find`/`Verify`/`In` are erased interfaces (`umpire.go:350-356`), so a Property of one machine paired with a Scenario of an unrelated machine compiles. README admits this (`README.md:76`, `:194-195`); the sample error at `README.md:139` is fabricated output, as the README's opening admits nothing was compiled.
- **Exhaustiveness relies on linter defaults.** Both linters must run with `default-signifies-exhaustive` off (the default) for the `default: panic` arms to remain reportable; the README says so (`README.md:119-122`), but a lint-config change silently removes the only exhaustiveness check.
- **No semantic drift found.** Every step arm of both product and protocol machines was checked against SPEC.md, including the three `failed(true)` yield arms (`standalone_activity.go:485-494`), `requestCancel` collapsing to all non-terminal phases, and `running`/`held` defined by negation; all equivalent to the spec's lists.

## Strengths

- **Compile-time action/step binding is real.** `umpire.Bind1(handlerReply, protocolHandlerReplyStep)` (`nexus_caller.go:592`) infers `A` from both arguments, so a step bound to the wrong action or given the wrong input type fails at `go vet`, exactly as `README.md:99-104` claims.
- **Typed protobuf schemas exceed Lean.** `umpire.Schema(&nexuspb.StartOperationResponse{}, &nexuspb.HandlerError{})` (`nexus_caller.go:110`) makes a renamed API message a compile error where Lean carries a string.
- **Revised Model 2 is applied completely and pinned.** The product retry rows, the `PauseRequested -> started` mapping, and a dedicated pin for all three retryable-failure yields (`standaloneactivity/pins_test.go:37-45`) go beyond the spec's minimum pin list.
- **Library table is accurate.** All five maintenance dates and archive flags spot-checked via `gh api` match the README exactly, and no-go entries are justified.
- **Comments preserved nearly verbatim** with the `// authoring:` markers, including the empty `case` section kept for order (`nexus_caller.go:905-911`).

## One-paragraph verdict

This sample shows that Go can host the Umpire model layer with zero new toolchain and with two genuinely compile-time guarantees Lean lacks (typed protobuf schemas) or matches (action/step input binding via generic inference), while every semantic check (finite domain membership, evidence completeness, refinement, query answers, pins) moves to `go test` a second later. The Models are faithful and complete, the framework surface is well-typed, and the README is unusually honest about cost. The single biggest reservation is ceremony: the two Model files are 1967 lines against a 777-line Lean reference, with sum types costing interface, structs, marker methods, a linter directive and a hand-listed variant set each, and enum constants wearing type prefixes that break the spec's naming for side-by-side reading. Readers who mostly read will find machines and sets legible as configuration, but the vocabulary sections and properties read as code.
