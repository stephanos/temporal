package umpire

/**
 * Where the Definition IDs of the declarations of one owner, an object or a file, hang off. An action,
 * monitor, assumption, hole, channel or realization takes its ID from the owner of its `val` and the
 * `val`'s name. Declarations moved out of `former` keep the IDs they had there through one pin in
 * their new owner, the compiler's name for the old one (`temporal.standaloneactivity.Model$package$`
 * for the file `Model.scala`, `temporal.nexuscaller.Control$` for `object Control`), rather than one
 * ID per declaration:
 *
 * {{{
 * object Protocol:
 *   given DefinitionScope = DefinitionScope("temporal.standaloneactivity.Model$package$")
 * }}}
 *
 * The lifter reads it (model/lifter): an owner pins at most once, a pin inside an owner that pins is
 * refused, and so is a pin of the owner itself or two declarations that would share an ID. Owners
 * nested in a pinned one keep their own IDs.
 */
final case class DefinitionScope(former: String)
