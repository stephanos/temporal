import Umpire3.TemporalLassoRunner

def main (arguments : List String) : IO UInt32 := do
  match arguments with
  | lassoDigest :: target :: property :: world :: variant :: semanticHash ::
      loopStartText :: stateCountText :: values =>
      let some loopStart := loopStartText.toNat?
        | IO.eprintln "temporal lasso loop start must be a natural number"; return 2
      let some stateCount := stateCountText.toNat?
        | IO.eprintln "temporal lasso state count must be a natural number"; return 2
      let states := values.take stateCount
      let actions := values.drop stateCount
      if states.length != stateCount || actions.length != stateCount then
        IO.eprintln "temporal lasso requires one action per state"
        return 2
      let request : Umpire3.TemporalLassoReplay.Request := {
        lassoDigest
        target
        property
        world
        variant
        semanticHash
        states
        actions
        loopStart
      }
      if accepted : Umpire3.TemporalLassoReplay.checkRequest request then
        let _ := Umpire3.TemporalLassoReplay.checkedRequest request accepted
        IO.println request.receipt.compress
        return 0
      else
        IO.eprintln "canonical Lean temporal lasso replay rejected the request"
        return 1
  | _ =>
      IO.eprintln "temporal lasso digest, identity, loop, states, and actions are required"
      return 2
