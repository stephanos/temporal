package umpire

/**
 * Where the Definition IDs of the declarations of one owner, an object or a file, hang off. An action,
 * monitor, assumption, hole, channel or realization takes its ID from the owner of its `val` and the
 * `val`'s name. Declarations moved out of `former` keep the IDs they had there through one pin in
 * their new owner, the compiler's name for the old one (`example.orders.Model$package$`
 * for the file `Model.scala`, `example.orders.Control$` for `object Control`), rather than one
 * ID per declaration:
 *
 * {{{
 * object Protocol:
 *   given DefinitionScope = DefinitionScope("example.orders.Model$package$")
 * }}}
 *
 * The IR generator reads it (model/irgen): an owner pins at most once, a pin inside an owner that
 * pins is refused, and so is a pin of the owner itself or two declarations that would share an ID.
 * Owners nested in a pinned one keep their own IDs.
 *
 * Type names follow a pin of a former file owner: a type at the top level of a pinning file belongs
 * to its package, not the file, so it takes the IR name `pkg.<Type>` it had beside
 * `pkg.File$package$`. A pin of an object owner leaves type names alone, and two types that would
 * share an IR name are refused.
 */
final case class DefinitionScope(former: String)
