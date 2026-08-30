import Temporal.Tool.RunEvaluation

def main (_args : List String) : IO UInt32 := do
  let input ← (← IO.getStdin).readBinToEnd
  let result := Temporal.Tool.RunEvaluation.runBytes input
  (← IO.getStdout).write result.stdout
  IO.eprint result.stderr
  pure result.status
