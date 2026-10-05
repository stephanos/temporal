package umpire

/**
 * A group of a Model's declarations, written as an object that extends it: `object timers extends
 * Section`, whose members are the timers of the feature. A section is transparent to Definition IDs:
 * an action, monitor, assumption, hole, channel or realization declared in one takes the ID it would
 * take as a direct member of the section's enclosing owner, so grouping declarations changes no ID.
 * At the top level of a file that owner is the file's package object, so the file's
 * `DefinitionScope` applies; in a machine's object it is that object, under its own pin.
 *
 * The IR generator reads it (model/irgen): a section sits at the top level of a Model file or
 * directly in a machine's object, never in another section, and pins nothing of its own. Two members
 * that would take one ID, such as two sections' `tick`s at the top level of one file, are refused.
 */
trait Section

/**
 * A party whose actions are the members of its object: `object caller extends Actor`, whose
 * `val start = action(this)` is an action the caller performs. Its name is its object's, with the
 * first letter lowered, as a party's name is its val's. An actor is a section, so its actions keep the
 * Definition IDs they would have as direct members of the object's enclosing owner.
 */
abstract class Actor extends Party(), Section
