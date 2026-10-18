// The bounds more than one Model folder runs its Queries under, declared once. A Query's receipt
// names its Limits by name, so a bound keeps the name its Queries were checked with wherever it is
// declared. A bound one folder alone uses stays in that folder's feature file, as does a bound
// whose name another folder gives a different budget: the close policy's `four`, `five` and
// `twelve` search further than the activity's and the task queue's of the same names.
package temporal

import framework.*

object Bounds:
  val three = Limits(steps = 3, actions = 3, search = 4096)
  val four = Limits(steps = 4, actions = 4, search = 32768)
