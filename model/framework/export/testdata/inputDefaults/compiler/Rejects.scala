package fixture.inputdefaults.compiler

import framework.dsl.*
import fixture.inputdefaults.*

// Record classes the compiler refuses where they are written: a required field omitted, a field
// given twice, and a field the record does not declare.
val omitted = caller.needs()
val partial = caller.needs(preferred = Mode.first)
val duplicate = caller.two(preferred = Mode.first, preferred = Mode.second)
val foreign = caller.two(required = Mode.second)
