/* The Stainless half of the kernel prelude: the same names as src/prelude/Prelude.scala, over
 * Stainless's own list, which is the one its solvers reason about. Stainless reads src/kernel with
 * this file in place of the runtime prelude. */
package kernel.prelude

import stainless.collection.*

type Steps[A] = List[A]
type Facts[F] = List[F]

final case class Step[S, O, F](outcome: O, state: S, facts: List[F])

def step[S, O, F](outcome: O, state: S, facts: Facts[F]): Step[S, O, F] = Step(outcome, state, facts)
def none[A]: Steps[A] = Nil[A]()
def one[A](a: A): Steps[A] = Cons(a, Nil[A]())
def facts0[F]: Facts[F] = Nil[F]()
def facts1[F](a: F): Facts[F] = Cons(a, Nil[F]())
