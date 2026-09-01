import Temporal.Tool.SemanticInventory

/-! Canonical semantic-inventory validation and rendering regressions. -/

namespace Temporal.Tool.SemanticInventoryTests

open Umpire
open Temporal.Tool.SemanticInventory

private def rendered (inventory : Inventory) : Option String :=
  (validateAndRender inventory).toOption

private def occurrences (document needle : String) : Nat :=
  (document.splitOn needle).length - 1

example : (rendered currentInventory).isSome = true := by
  native_decide

example :
    rendered { currentInventory with
      outcomeFamilies := currentInventory.outcomeFamilies.reverse
      projectionSentinels := currentInventory.projectionSentinels.reverse
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
          document.contains "## Projection sentinels\n" &&
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
          currentInventory.projectionSentinels.all (fun sentinel =>
            occurrences document ("| `" ++ sentinel.id ++ "` |") == 1) &&
          currentInventory.knownGaps.all (fun row =>
            occurrences document ("| `" ++ row.id ++ "` | `" ++ row.owner ++ "` |") == 1)) = true := by
  native_decide

end Temporal.Tool.SemanticInventoryTests
