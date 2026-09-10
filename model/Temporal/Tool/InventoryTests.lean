import Temporal.Tool.Inventory

/-! Canonical inventory validation and rendering regressions. -/

namespace Temporal.Tool.InventoryTests

open Umpire
open Temporal.Tool.Inventory

private def rendered (inventory : Inventory) : Option String :=
  (validateAndRender inventory).toOption

private def occurrences (document needle : String) : Nat :=
  (document.splitOn needle).length - 1

example : (rendered currentInventory).isSome = true := by
  native_decide

example :
    rendered { currentInventory with
      outcomeFamilies := currentInventory.outcomeFamilies.reverse
      notRunMarkers := currentInventory.notRunMarkers.reverse
      knownGaps := currentInventory.knownGaps.reverse } =
      rendered currentInventory := by
  native_decide

example :
    (match currentInventory.outcomeFamilies.head? with
    | none => false
    | some family =>
        match validateAndRender {
          currentInventory with
          outcomeFamilies := { family with constructors := family.constructors.reverse } ::
            currentInventory.outcomeFamilies.drop 1
        } with
        | .error _ => true
        | .ok _ => false) = true := by
  native_decide

example :
    (match rendered currentInventory with
    | none => false
    | some document =>
        document.startsWith "# Umpire semantic inventory\n\n" &&
          document.contains "## Outcome families\n" &&
          document.contains "## Stage not-run markers\n" &&
          document.contains "## Known Gap flows\n" &&
          document.contains
            "| Catalog ID | Owner | Lineage | Scope | Shape | Source/reference | Field mapping | Description |" &&
          document.endsWith "\n" && !document.endsWith "\n\n" &&
          !document.contains "/Users/" && !document.contains "Generated at" &&
          currentInventory.outcomeFamilies.all (fun family =>
            occurrences document ("### `" ++ family.id ++ "`") == 1 &&
              family.constructors.all (fun constructor =>
                occurrences document ("| `" ++ constructor.name ++ "` | " ++
                  constructor.description ++ " |") == 1)) &&
          currentInventory.notRunMarkers.all (fun marker =>
            occurrences document ("| `" ++ marker.id ++ "` |") == 1) &&
          currentInventory.knownGaps.all (fun row =>
            occurrences document ("| `" ++ row.id ++ "` | `" ++ row.owner ++ "` |") == 1)) = true := by
  native_decide

example :
    let malformedFamilies := [
      currentInventory.outcomeFamilies ++ currentInventory.outcomeFamilies,
      currentInventory.outcomeFamilies.map fun family =>
        { family with constructors := family.constructors ++ family.constructors },
      currentInventory.outcomeFamilies.map fun family => { family with owner := "" }
    ]
    let malformed := malformedFamilies.map (fun families =>
      { currentInventory with outcomeFamilies := families }) ++ [
      { currentInventory with notRunMarkers := [] },
      { currentInventory with knownGaps := [] }
    ]
    malformed.map rendered = [none, none, none, none, none] := by
  native_decide

end Temporal.Tool.InventoryTests
