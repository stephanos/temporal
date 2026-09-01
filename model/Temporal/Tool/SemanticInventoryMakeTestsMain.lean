import Temporal.Tool.SemanticInventoryMakeTests

def main : IO UInt32 := do
  try
    Temporal.Tool.SemanticInventoryMakeTests.runIO
    pure 0
  catch failure =>
    IO.eprintln s!"semantic-inventory Make regression: {failure}"
    pure 1
