import Temporal.Veil.NexusCancellationFencing.MutatedBinding

def main (arguments : List String) : IO UInt32 := do
  let [semanticHash] := arguments
    | IO.eprintln "first-order semantic hash is required"
      return 2
  let some binding := Umpire3.Temporal.Veil.NexusCancellationFencing.mutatedBindingExport
    | IO.eprintln "mutated Veil binding does not match its checked first-order export"
      return 2
  let cancelToken ← IO.CancelToken.new
  let result ← NexusCancellationStaleCompletionGuardRemovedConcrete.modelCheckerResult none 0 cancelToken
  IO.println (Lean.Json.mkObj [
    ("binding", binding.compiledJson semanticHash),
    ("result", Lean.toJson result),
  ]).compress
  return 0
