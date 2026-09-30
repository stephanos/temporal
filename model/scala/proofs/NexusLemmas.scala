/* What Stainless proves about the Nexus kernel, for every state and every action class rather than
 * the 192 states and 22 classes the table enumerates. Each lemma is one `holds` Stainless discharges
 * with its SMT solvers; run.sh runs them, and a failed lemma prints the counterexample state.
 *
 * The lemmas restate, as theorems, what the Lean model checks by `decide +kernel` over its finite
 * table and what the Go and Scala tests check row by row: the attempt count stays in its bound, a
 * finished operation stays finished, the protocol machine refines the product machine, and the only
 * silent accepted steps are the ones a canary set must not name.
 */
package proofs

import kernel.nexus.*
import kernel.nexus.Actions.*
import kernel.nexus.Protocol.*
import kernel.prelude.Step
import stainless.collection.*
import stainless.lang.*

object NexusLemmas:
  def valid(s: ProtocolState): Boolean = validAttempts(s.attempts)

  /** Every protocol step keeps the attempt count within its bound: the saturating successor is the
    * only arithmetic, and a table over `0..attemptBound` could not tell it from wrapping. */
  def attemptsStayBounded(s: ProtocolState, a: ProtocolAction): Boolean = {
    require(valid(s))
    protocolStep(s, a).forall(r => valid(r.state))
  }.holds

  /** Once an operation is over, no step changes its phase: `terminalIsFinal`, proved on the protocol
    * machine directly rather than read through the map. */
  def terminalIsFinal(s: ProtocolState, a: ProtocolAction): Boolean = {
    require(valid(s) && terminalPhase(s.phase))
    protocolStep(s, a).forall(r => r.state.phase == s.phase)
  }.holds

  /** The same claim on the product machine. */
  def productTerminalIsFinal(s: ProductState, a: ProductAction): Boolean = {
    require(Product.productTerminal(s))
    productStep(s, a).forall(r => r.state.phase == s.phase)
  }.holds

  /** The protocol machine refines the product machine: every protocol step either leaves the product
    * reading of its state unchanged (a stutter), or is a step of the product action of its own name
    * from the mapped state, to the mapped state, with the same outcome, recording every fact the
    * product step records. This is the Lean `refines:` theorem for every state, under a stricter
    * carrier (the action of the same name, not any product action). */
  def refines(s: ProtocolState, a: ProtocolAction): Boolean = {
    require(valid(s))
    protocolStep(s, a).forall(r => productOf(r.state) == productOf(s) || carriedBy(productReading(a), productOf(s), r))
  }.holds

  /** A product step of the named action carries the protocol step: same mapped target, same outcome,
    * and every product fact among the protocol step's facts read by name. */
  def carriedBy(reading: ProductReading, from: ProductState, r: ProtocolStep): Boolean = reading match
    case ProductReading.as(pa) => productStep(from, pa).exists((p: ProductStep) => carries(p, r))
    case ProductReading.unnamed => false

  def carries(p: ProductStep, r: ProtocolStep): Boolean =
    p.state == productOf(r.state) && p.outcome == r.outcome && subset(p.facts, productFacts(r.facts))

  def subset(xs: List[ProductFact], ys: List[ProductFact]): Boolean = xs match
    case Nil()       => true
    case Cons(x, tl) => ys.contains(x) && subset(tl, ys)

  def silent(r: ProtocolStep): Boolean = r.outcome == Outcome.accepted && r.facts == Nil[ProtocolFact]()

  /** The only accepted steps that record nothing are the backoff timer and the worker stop: the
    * silent steps a canary set is rejected for naming. */
  def silentStepsAreBackoffAndWorkerStop(s: ProtocolState, a: ProtocolAction): Boolean = {
    require(valid(s))
    protocolStep(s, a).forall(r => !silent(r) || a == ProtocolAction.backoff || a == ProtocolAction.workerStop)
  }.holds
