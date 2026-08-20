import Umpire3Veil.Generated.NexusCancellationSoundConcrete

def main : IO Unit := do
  let cancelToken ← IO.CancelToken.new
  let result ← NexusCancellationSoundConcrete.modelCheckerResult none 0 cancelToken
  IO.println (Lean.toJson result).compress
