namespace Shared.TraceReplay

def followNamed {State Action : Type}
    (successors : State → List (Action × State)) (actionName : Action → String) :
    List State → List String → List State
  | states, [] => states
  | states, identifier :: identifiers =>
      followNamed successors actionName
        (states.flatMap fun state =>
          (successors state).filterMap fun successor =>
            if actionName successor.1 = identifier then some successor.2 else none)
        identifiers

def check {State Action : Type}
    (successors : State → List (Action × State)) (actionName : Action → String)
    (initials : List State) (property : State → Bool) (identifiers : List String) : Bool :=
  (followNamed successors actionName initials identifiers).any fun state => !property state

end Shared.TraceReplay
