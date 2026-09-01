import Temporal.Tool.SemanticInventory

/-! Effect-thin executable boundary for the checked semantic inventory. -/

namespace Temporal.Tool.SemanticInventoryMain

open Temporal.Tool.SemanticInventory

end Temporal.Tool.SemanticInventoryMain

def main (_args : List String) : IO UInt32 :=
  Temporal.Tool.SemanticInventory.run Temporal.Tool.SemanticInventory.currentInventory
    IO.print IO.eprint
