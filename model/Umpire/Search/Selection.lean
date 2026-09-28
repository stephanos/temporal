import Umpire.Search.Backend.Veil

/-!
# Choosing a Query's search backend

`select` chooses the backend a Query searches on and records why. A Query runs on `veil` when its
strategy and form are ones the product search answers and its product builds: every Property clause
lowers to a monitor and its Scenario lowers to a progress automaton. Otherwise it runs on
`reference`, and the reason names what ruled `veil` out: the strategy, the form, the first clause
kind, or the Scenario construct, checked in that order.

`search` runs the selected backend and `searchWith` a named one, for tests. `AdmittedQuery.search`,
`searchWithIntent` and `searchWith` call them. This module is the only importer of
`Umpire.Search.Backend.Veil` (`search-backend-isolation`); the two are the closed set the rollback
drill reduces.
-/

namespace Umpire.Search.Selection

open Umpire
open Umpire.Search.Product

variable {LawStatement : Law → Prop}

/-- A search backend by name. -/
inductive BackendName where
  | reference
  | veil
  deriving BEq, DecidableEq, Repr

/-- The backend chosen for one Query: `veil` with the product it searches, or `reference` with
the reason `veil` was ruled out. -/
inductive Choice (query : CheckedQuery LawStatement) where
  | veil (monitored : MonitoredProduct query.target)
  | reference (reason : BackendReason)

/-- The receipt's `backendReason` for a Query version one's product cannot encode. -/
def reasonOf : ProductUnsupported → BackendReason
  | .clause unsupported => .unsupportedClause unsupported.kind.name
  | .scenario unsupported => .unsupportedScenario unsupported.construct.name

private def strategyReason : SearchStrategy → Option BackendReason
  | .seeded => some (.unsupportedStrategy .seeded)
  | .exhaustive | .breadthFirst | .shortest => none

/-- The product search answers every form version one has; the match is exhaustive so that a new
form is decided here. -/
private def formReason : Query.Form → Option BackendReason
  | .verify _ | .find _ | .findViolation _ | .pick _ => none

/-- Choose the backend for a Query over its search view. -/
def select (query : CheckedQuery LawStatement) (view : SearchView query.target) :
    Choice query :=
  match strategyReason query.policy.strategy <|> formReason query.form with
  | some reason => .reference reason
  | none =>
      match MonitoredProduct.build query view with
      | .ok monitored => .veil monitored
      | .error unsupported => .reference (reasonOf unsupported)

namespace Choice

variable {query : CheckedQuery LawStatement}

private def withReason (reason : BackendReason) (observations : PlanningObservations) :
    PlanningObservations :=
  { observations with instrumentation := { observations.instrumentation with
      backendReason := reason } }

/-- Run the chosen backend; a `reference` run records the reason it was chosen. -/
def run (choice : Choice query) (view : SearchView query.target) : BackendResult :=
  match choice with
  | .veil monitored => Backend.Veil.run query monitored
  | .reference reason =>
      match Backend.reference query view with
      | .violationFound trace observations => .violationFound trace (withReason reason observations)
      | .complete observations => .complete (withReason reason observations)
      | .stateBound visited observations => .stateBound visited (withReason reason observations)
      | .invalid error observations => .invalid error (withReason reason observations)

end Choice

/-- Whether `search` runs a Query on `veil` when `select` chooses it. The cutover set it in the
one commit that re-pinned the goldens `veil` changes (fn-88 R18); `false` sends every Query `select`
chooses `veil` for to `reference` with reason `default`. -/
def cutover : Bool := true

/-- Search a Query on the backend `select` chooses, through the shared finalization and its kernel
replay gate. -/
def search (query : CheckedQuery LawStatement) (view : SearchView query.target) :
    Except KnownGapError PlanResult :=
  let choice : Choice query := match select query view with
    | .veil monitored => if cutover then .veil monitored else .reference .default
    | .reference reason => .reference reason
  finalizeBackendResult query view (choice.run view)

/-- Search a Query on a named backend. `veil` runs on `reference`, with the reason recorded, when
`select` rules it out. -/
def searchWith (backend : BackendName) (query : CheckedQuery LawStatement)
    (view : SearchView query.target) : Except KnownGapError PlanResult :=
  let choice : Choice query := match backend with
    | .reference => .reference .default
    | .veil => select query view
  finalizeBackendResult query view (choice.run view)

end Umpire.Search.Selection
