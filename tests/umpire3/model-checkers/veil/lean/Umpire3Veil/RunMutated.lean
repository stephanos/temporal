import Umpire3Veil.Generated.NexusCancellationMutatedConcrete

def main : IO Unit := do
  let cancelToken ← IO.CancelToken.new
  let result ← NexusCancellationStaleCompletionGuardRemovedConcrete.modelCheckerResult none 0 cancelToken
  IO.println (Lean.toJson result).compress
