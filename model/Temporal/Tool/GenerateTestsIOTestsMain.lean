import Temporal.Tool.GenerateTestsTests

def main : IO UInt32 := do
  try
    Temporal.Tool.GenerateTestsTests.runIO
    pure 0
  catch failure =>
    IO.eprintln s!"umpire-gen-tests IO regression: {failure}"
    pure 1
