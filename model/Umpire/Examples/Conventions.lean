import Umpire.Command

/-!
# What every Umpire example shares

One declaration, read by the Model commands: Umpire's own examples hang their Definition IDs off the
`umpire` root and treat `Umpire.Examples` as scaffolding rather than semantic family, so the switch
example's ids read `umpire.switch.<kind>.<owner>.<member>`. It is Temporal's declaration
(`Temporal.Case.Conventions`) for a tree that names no platform, and the two never meet: a
declaration reads the conventions whose namespace prefix covers it.
-/

model_conventions root "umpire" under Umpire.Examples
