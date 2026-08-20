namespace Umpire3

inductive FaultKind where
  | drop
  | delay
  | duplicate
  | reorder
  | holdRelease
  | rejection
  | processCrash
  | restart
  | partition
  | failover
  | clockSkew
  | persistenceError
  deriving DecidableEq, Repr

structure FaultScope where
  namespaces : List String
  endpoints : List String := []
  taskQueues : List String := []
  services : List String := []
  routes : List String := []
  participants : List String := []
  attempts : List Nat := []
  deriving DecidableEq, Repr

structure Occurrence where
  first : Nat
  count : Nat
  deriving DecidableEq, Repr

structure Interval where
  start : Nat
  stop : Nat
  deriving DecidableEq, Repr

structure FaultTerm where
  kind : FaultKind
  scope : FaultScope
  occurrence : Occurrence
  interval : Interval
  deriving DecidableEq, Repr

def FaultTerm.safe (term : FaultTerm) : Bool :=
  !term.scope.namespaces.isEmpty &&
    term.scope.namespaces.all (· != "") &&
    term.occurrence.count > 0 &&
    term.interval.start < term.interval.stop

inductive FaultLifecycle where
  | declared
  | installed
  | active
  | observed
  | released
  | cleaned
  deriving DecidableEq, Repr

inductive FaultAction where
  | install
  | activate
  | observe
  | release
  | cleanup
  deriving DecidableEq, Repr

def faultStep : FaultLifecycle → FaultAction → Option FaultLifecycle
  | .declared, .install => some .installed
  | .installed, .activate => some .active
  | .active, .observe => some .observed
  | .active, .release => some .released
  | .observed, .release => some .released
  | .released, .cleanup => some .cleaned
  | .installed, .cleanup => some .cleaned
  | _, _ => none

theorem cleaned_is_terminal (action : FaultAction) : faultStep .cleaned action = none := by
  cases action <;> rfl

theorem safe_example : ({
    kind := FaultKind.drop
    scope := { namespaces := ["isolated"] }
    occurrence := { first := 1, count := 1 }
    interval := { start := 0, stop := 1 }
  } : FaultTerm).safe := by rfl

end Umpire3
