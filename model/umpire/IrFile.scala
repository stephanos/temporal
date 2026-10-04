package umpire

import umpire.realize.Realization

/**
 * A checked-in IR file, `model/ir/<name>.json`, and the declarations that are its roots, named by
 * value beside the Models it holds:
 *
 * {{{
 * val nexusControlFile =
 *   irFile("nexus-control")(forgedCompletion, NexusRealization.forgedCompletion)
 * }}}
 *
 * The lifter (model/lifter) reads every such val and writes each file, in one run, from its roots and
 * everything they reach, as it lifts the roots named on its command line. A root that names nothing
 * does not compile. A declaration may be a root of several files and is lifted into each; one no
 * file names stays out of model/ir, so a design can be kept out of the checked files. The file's
 * `source` lists its roots' fully qualified names.
 */
final class IrFile private[umpire] (val name: String, val roots: Seq[IrRoot])

/**
 * What an IR file names as a root: a machine, a composition, a Query, a list of Queries, a progress
 * claim, a realization, or a capability declaration with the laws it brings.
 */
type IrRoot = Machine[?, ?, ?] | Composition[?] | Query | Seq[Query] | Progress[?] | Realization |
  Capabilities[?]

/** Declares the IR file `model/ir/<name>.json` and its roots. */
def irFile(name: String)(roots: IrRoot*): IrFile = IrFile(name, roots)
