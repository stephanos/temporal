# Re-pinned positions (fn-174)

The plain and edited frozen originals (`original-<root>.json`, `original-edited-<root>.json`)
were captured when the declaration reader left every position in the fixtures' `Signature.scala`
empty (`"position": { }`), while it positioned the same declarations in the jar and single-unit
variants. The reader now positions them. Only those empty positions are re-pinned, each to the
file name and line the current reader records, under the capture's own directory; every other
field and position is unchanged. The re-pinned lines equal the ones the frozen jar variant
records for the same `Signature.scala` declarations.
