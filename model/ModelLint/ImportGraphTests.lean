import ModelLint.ImportGraph
import Tools.LeanImportGraphTests
import Tools.LeanSourceInventoryTests

/-! Executable synthetic regressions for model import-graph policy and inventory checking. -/

open ModelLint.ImportGraph

private def moduleRecord (name : Lean.Name) (imports : Array Lean.Name := #[]) : ModuleRecord :=
  { name, imports }

private def sourceRecord
    (module : Lean.Name) (path := s!"{module}.lean") : SourceRecord :=
  { path, module }

private def requireEqual [BEq α] [Repr α] (label : String) (actual expected : α) : IO Unit :=
  unless actual == expected do
    throw <| IO.userError s!"{label}: expected {repr expected}, got {repr actual}"

private def requireViolation
    (label : String)
    (modules : Array ModuleRecord)
    (rule : Rule)
    (path : Array Lean.Name) : IO Unit := do
  let violations := check defaultPolicy modules
  requireEqual label violations.size 1
  let some violation := violations[0]?
    | throw <| IO.userError s!"{label}: missing violation"
  requireEqual s!"{label} rule" violation.rule rule
  requireEqual s!"{label} path" violation.path path

private structure ForbiddenCase where
  label : String
  source : Lean.Name
  destination : Lean.Name
  rule : Rule

private def forbiddenCases : Array ForbiddenCase := #[
  { label := "Shared to Umpire", source := `Shared.Root, destination := `Umpire.Core,
    rule := .sharedIndependence },
  { label := "Shared to Temporal", source := `Shared.Root, destination := `Temporal.Feature.Root,
    rule := .sharedIndependence },
  { label := "Shared to Veil", source := `Shared.Root,
    destination := `Umpire.Verify.Veil.Core,
    rule := .sharedIndependence },
  { label := "Umpire to Temporal", source := `Umpire.Root, destination := `Temporal.Feature.Root,
    rule := .umpireIndependence },
  { label := "Umpire to Verify", source := `Umpire.Root,
    destination := `Temporal.Verify.Root,
    rule := .umpireIndependence },
  { label := "Veil implementation to Temporal", source := `Umpire.Verify.Veil.Core,
    destination := `Temporal.Feature.Root,
    rule := .umpireIndependence },
  { label := "Feature to System", source := `Temporal.Feature.Root,
    destination := `Temporal.System.Root,
    rule := .featureIsolation },
  { label := "Feature to Verify", source := `Temporal.Feature.Root,
    destination := `Temporal.Verify.Root,
    rule := .featureIsolation },
  { label := "Feature to Veil", source := `Temporal.Feature.Root,
    destination := `Umpire.Verify.Veil.Core,
    rule := .featureIsolation },
  { label := "System to Feature", source := `Temporal.System.Root,
    destination := `Temporal.Feature.Root,
    rule := .systemIsolation },
  { label := "Umpire to Veil", source := `Umpire.Root,
    destination := `Umpire.Verify.Veil.Core,
    rule := .verificationIsolation },
  { label := "System to Verify", source := `Temporal.System.Root,
    destination := `Temporal.Verify.Root,
    rule := .verificationIsolation },
  { label := "System to Veil", source := `Temporal.System.Root,
    destination := `Umpire.Verify.Veil.Core,
    rule := .verificationIsolation },
  { label := "Temporal to Verify", source := `Temporal.Root,
    destination := `Temporal.Verify.Root,
    rule := .verificationIsolation },
  { label := "Temporal to Veil", source := `Temporal.Root,
    destination := `Umpire.Verify.Veil.Core,
    rule := .verificationIsolation },
  { label := "model tests to Verify", source := `TemporalModelTests,
    destination := `Temporal.Verify.Root,
    rule := .verificationIsolation },
  { label := "model tests to Veil", source := `TemporalModelTests,
    destination := `Umpire.Verify.Veil.Core,
    rule := .verificationIsolation },
  { label := "tool to Verify", source := `Temporal.Tool.Inspect,
    destination := `Temporal.Verify.Root,
    rule := .verificationIsolation },
  { label := "tool to Veil", source := `Temporal.Tool.Inspect,
    destination := `Umpire.Verify.Veil.Core,
    rule := .verificationIsolation }
]

private def testDirectAndTransitiveRejections : IO Unit := do
  for testCase in forbiddenCases do
    requireViolation s!"{testCase.label} direct"
      #[moduleRecord testCase.source #[testCase.destination], moduleRecord testCase.destination]
      testCase.rule #[testCase.source, testCase.destination]
    requireViolation s!"{testCase.label} transitive"
      #[
        moduleRecord testCase.source #[`ModelLint.Bridge],
        moduleRecord `ModelLint.Bridge #[testCase.destination],
        moduleRecord testCase.destination
      ]
      testCase.rule #[testCase.source, `ModelLint.Bridge, testCase.destination]

private def testAllowedOrdinaryImports : IO Unit := do
  let modules := #[
    moduleRecord `Shared.Root #[`Std.Data.HashMap],
    moduleRecord `Umpire.Root #[`Shared.Root],
    moduleRecord `Temporal.Feature.Root #[`Umpire.Root],
    moduleRecord `Temporal.System.Root #[`Umpire.Root],
    moduleRecord `Temporal.Root #[`Temporal.Feature.Root, `Temporal.System.Root]
  ]
  requireEqual "allowed ordinary imports" (check defaultPolicy modules) #[]

private def testExactRefinementException : IO Unit := do
  let allowed := #[
    moduleRecord `Temporal.System.Nexus.Refinement #[`Temporal.Feature.Nexus.Root],
    moduleRecord `Temporal.Feature.Nexus.Root
  ]
  requireEqual "exact refinement" (check defaultPolicy allowed) #[]
  requireViolation "refinement near miss"
    #[
      moduleRecord `Temporal.System.Nexus.Refinement.Extra #[`Temporal.Feature.Nexus.Root],
      moduleRecord `Temporal.Feature.Nexus.Root
    ]
    .systemIsolation
    #[`Temporal.System.Nexus.Refinement.Extra, `Temporal.Feature.Nexus.Root]

private def testExactVerifyExceptions : IO Unit := do
  let verifyDestinations := #[`Temporal.Verify.Nexus.Root, `Umpire.Verify.Veil.Core]
  for consumer in defaultPolicy.verifyConsumers do
    let modules := #[
      moduleRecord consumer verifyDestinations,
      moduleRecord verifyDestinations[0]!,
      moduleRecord verifyDestinations[1]!
    ]
    requireEqual s!"exact verification consumer {consumer}" (check defaultPolicy modules) #[]
  for nearMiss in #[
    `Temporal.Feature.Nexus.Experimental.CallerClosure.VeilTests.Extra,
    `Temporal.Tool.VerifyVeil.Extra
  ] do
    let modules := #[moduleRecord nearMiss #[`Umpire.Verify.Veil.Core],
      moduleRecord `Umpire.Verify.Veil.Core]
    let violations := check defaultPolicy modules
    requireEqual s!"verification near miss {nearMiss}" violations.size 1
  for unclassifiedNearMiss in #[`TemporalVerify.Extra, `TemporalVeilTests.Extra] do
    let sources := #[sourceRecord unclassifiedNearMiss]
    let modules := #[moduleRecord unclassifiedNearMiss]
    requireEqual s!"aggregate near miss {unclassifiedNearMiss}"
      (reconcile defaultPolicy sources modules)
      #[.unclassifiedModule unclassifiedNearMiss]

private def testModelInventoryPolicy : IO Unit := do
  requireEqual "experimental test aggregate classified"
    (reconcile defaultPolicy #[sourceRecord `TemporalExperimentalTests]
      #[moduleRecord `TemporalExperimentalTests])
    #[]
  requireEqual "unclassified source"
    (reconcile defaultPolicy #[sourceRecord `Unknown.Root] #[moduleRecord `Unknown.Root])
    #[.unclassifiedModule `Unknown.Root]
  requireEqual "unknown first-party import"
    (reconcile defaultPolicy #[sourceRecord `Temporal.Root]
      #[moduleRecord `Temporal.Root #[`Temporal.Future]])
    #[.unknownFirstPartyImport `Temporal.Root `Temporal.Future]
  requireEqual "unknown near-miss aggregate import"
    (reconcile defaultPolicy #[sourceRecord `Temporal.Root]
      #[moduleRecord `Temporal.Root #[`TemporalVerify.Extra]])
    #[.unknownFirstPartyImport `Temporal.Root `TemporalVerify.Extra]

private def testExternalLeaves : IO Unit := do
  let sources := #[sourceRecord `Shared.Root]
  let modules := #[moduleRecord `Shared.Root #[`Lean.Data.Name, `Std.Data.HashMap]]
  requireEqual "external inventory leaves" (reconcile defaultPolicy sources modules) #[]
  requireEqual "external graph leaves" (check defaultPolicy modules) #[]

private def testStableShortestPath : IO Unit := do
  let modules := #[
    moduleRecord `Temporal.Feature.Root #[`ModelLint.BridgeB, `ModelLint.BridgeA],
    moduleRecord `ModelLint.BridgeA #[`Temporal.System.Target],
    moduleRecord `ModelLint.BridgeB #[`Temporal.System.Target],
    moduleRecord `Temporal.System.Target
  ]
  requireViolation "equal shortest paths" modules .featureIsolation
    #[`Temporal.Feature.Root, `ModelLint.BridgeA, `Temporal.System.Target]

private def testMultipleFindings : IO Unit := do
  let modules := #[
    moduleRecord `Temporal.Feature.Root #[`Temporal.System.Zed, `Temporal.System.Alpha],
    moduleRecord `Temporal.System.Alpha,
    moduleRecord `Temporal.System.Zed
  ]
  let violations := check defaultPolicy modules
  requireEqual "multiple findings count" violations.size 2
  requireEqual "multiple findings order" (violations.map (·.destination))
    #[`Temporal.System.Alpha, `Temporal.System.Zed]

private def testCyclesTerminate : IO Unit := do
  let modules := #[
    moduleRecord `Temporal.Feature.Root #[`ModelLint.BridgeA],
    moduleRecord `ModelLint.BridgeA #[`ModelLint.BridgeB],
    moduleRecord `ModelLint.BridgeB #[`ModelLint.BridgeA, `Temporal.System.Target],
    moduleRecord `Temporal.System.Target
  ]
  requireViolation "cycle-safe traversal" modules .featureIsolation
    #[`Temporal.Feature.Root, `ModelLint.BridgeA, `ModelLint.BridgeB, `Temporal.System.Target]

private def controlledViolations : Array Violation :=
  check defaultPolicy #[
    moduleRecord `Shared.Root #[`ModelLint.Bridge],
    moduleRecord `ModelLint.Bridge #[`Umpire.Core],
    moduleRecord `Umpire.Core
  ]

private def runSyntheticSuite : IO UInt32 := do
  Tools.LeanImportGraphTests.run
  Tools.LeanSourceInventoryTests.run
  testAllowedOrdinaryImports
  testDirectAndTransitiveRejections
  testExactRefinementException
  testExactVerifyExceptions
  testModelInventoryPolicy
  testExternalLeaves
  testStableShortestPath
  testMultipleFindings
  testCyclesTerminate
  IO.println "-- Model import-graph synthetic tests passed."
  pure 0

private def runControlledViolation : IO UInt32 := do
  for violation in controlledViolations do
    IO.eprintln violation.render
  pure <| exitCode controlledViolations.isEmpty true

def main (args : List String) : IO UInt32 :=
  match args with
  | [] => runSyntheticSuite
  | ["--controlled-violation"] => runControlledViolation
  | _ => do
      IO.eprintln "usage: modelLintTests [--controlled-violation]"
      pure 2
