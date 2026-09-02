import Temporal.Tool.PromoteTests

def main : IO UInt32 := do
  try
    Temporal.Tool.PromoteTests.runIORegressions
    pure 0
  catch failure =>
    IO.eprintln s!"promotion regression: {failure}"
    pure 1
