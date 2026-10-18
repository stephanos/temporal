# Re-pinned positions (fn-174)

The frozen Models that read the fixtures' `Signature.scala` were captured when the declaration
reader left every position in that file empty (`"position": { }`). The reader now positions
them. Only those empty positions are re-pinned, each to the file name and line the current
reader records, under the capture's own directory; every other field and position is unchanged.

`Arities.scala` and `Domains.scala` declare their multi-input actions with native records
(`inputs[Two]`, `inputs[Nested]`) whose fields are the old inputs in order, and the rules bind
`(s, in) => effects.two(s, in.s, in.state)`, so the frozen `ordinary-Arities.json` and
`ordinary-Domains.json` keep their inputs, rule functions and effect functions.
The actions and the record declarations are laid out so that every positioned line of the frozen
Models holds.

`Contributors.scala` declares its fault action with one native fault kind,
`action(this).responseLoss`, which native rule acquisition requires; the lifter reads the kind as
native metadata the IR action omits, so the frozen `ordinary-Forward.json` and
`ordinary-Reverse.json` are unchanged.
