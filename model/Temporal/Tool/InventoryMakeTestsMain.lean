import Temporal.Tool.InventoryMakeTests

def main : IO UInt32 := do
  try
    Temporal.Tool.InventoryMakeTests.runIO
    pure 0
  catch failure =>
    IO.eprintln s!"inventory Make regression: {failure}"
    pure 1
