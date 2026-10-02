import Temporal.Tool.Inventory

/-! Effect-thin executable boundary for the checked semantic inventory. -/

namespace Temporal.Tool.InventoryMain

open Temporal.Tool.Inventory

end Temporal.Tool.InventoryMain

def main (_args : List String) : IO UInt32 :=
  Temporal.Tool.Inventory.run Temporal.Tool.Inventory.currentInventory
    IO.print IO.eprint
