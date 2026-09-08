import Umpire.Value.Encoding

/-! Independent binary fixtures; semantic admission controls are added with the value owner. -/
open Umpire.Value.Encoding

#guard encode (.pair (.atom 2) .nil) == [2, 1, 1, 2, 0, 0]
#guard decode 3 [2, 1, 1, 2, 0, 0] == some (.pair (.atom 2) .nil, [])
#guard decode 1 [2, 1, 1, 2, 0, 0] == none
#guard decode 3 [9] == none
