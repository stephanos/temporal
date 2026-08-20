import Umpire3.TraceReplayRunner

def main (arguments : List String) : IO UInt32 := do
  match arguments with
  | traceDigest :: target :: property :: world :: variant :: semanticHash :: actions =>
      let request : Umpire3.TraceReplay.Request := {
        traceDigest := traceDigest
        target := target
        property := property
        world := world
        variant := variant
        semanticHash := semanticHash
        actions := actions
      }
      if accepted : Umpire3.TraceReplay.checkRequest request then
        let _ := Umpire3.TraceReplay.checkedRequest request accepted
        IO.println request.receipt.compress
        return 0
      else
        IO.eprintln "canonical Lean trace replay rejected the request"
        return 1
  | _ =>
      IO.eprintln "trace digest, target, property, world, variant, semantic hash, and actions are required"
      return 2
