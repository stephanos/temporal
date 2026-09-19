import Umpire.Command

/-!
# What every Temporal Model declaration shares

One declaration, read by the Model commands: Temporal hangs its Definition IDs off the `temporal`
root and treats `Temporal.Feature` as scaffolding rather than semantic family. Everything else a
declaration carries -- including its Known Gaps -- it declares for itself.
-/

model_conventions root "temporal" under Temporal.Feature
