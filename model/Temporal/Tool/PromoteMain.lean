import Temporal.Tool.Promote

def main (args : List String) : IO UInt32 := do
  let result := Temporal.Tool.Promote.runCli args
  IO.print result.stdout
  IO.eprint result.stderr
  if result.status == 0 then
    pure 0
  else
    pure 1
