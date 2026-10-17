package framework.dsl

// The words a Machine's claims and effects are written in where they read its vars by name, which
// a Machine and a derivation of one inherit, so that inside the object they are these. Each reads
// where it is written (ClaimMacros): in a block of the retired `expect`, whose expectation it finds
// as a given, it is the requirement it was there; in the Machine's sections it asserts natively
// (NativeClaims), and outside the sections it belongs in it is refused.
//
//   - `after(target):` a block of a postcondition about the steps of one action, or class of it;
//   - `before`: the State record the step started from, in a postcondition or a monitor;
//   - `records(fact)`: in a claim, that the step recorded the fact; in an effect, a fact it records;
//   - `oneOf:` with `branch:` blocks: alternatives, of which one holds.
trait ClaimWords:
  protected transparent inline def after: Any = ${ ClaimMacros.after }
  protected transparent inline def before: Any = ${ ClaimMacros.before }
  protected transparent inline def records: Any = ${ ClaimMacros.records }
  protected transparent inline def oneOf: Any = ${ ClaimMacros.oneOf }
  protected transparent inline def branch: Any = ${ ClaimMacros.branch }
