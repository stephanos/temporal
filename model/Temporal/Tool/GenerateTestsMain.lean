import Temporal.Tool.GenerateTests

def main (args : List String) : IO UInt32 := do
  let result := Temporal.Tool.GenerateTests.runCli args
  if let some batch := result.batch then
    try
      Temporal.Tool.GenerateTests.writeBatch batch
    catch failure =>
      IO.eprintln s!"umpire-gen-tests: {failure}"
      return 1
  IO.print result.stdout
  IO.eprint result.stderr
  if result.status == 0 then pure 0 else pure 1
