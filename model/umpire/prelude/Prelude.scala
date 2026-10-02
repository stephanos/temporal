/* The runtime half of the kernel prelude. The kernel's step functions are written against `Steps`,
 * `Step` and the constructors below; here they are the framework's own list and step, so the
 * machines, the pins and the Case bytes run on the same code proofs/umpire/Prelude.scala hands Stainless
 * with Stainless's list instead. */
package umpire.prelude
type Steps[A] = List[A]
type Facts[F] = List[F]
type Step[S, O, F] = umpire.Step[S, O, F]

def step[S, O, F](outcome: O, state: S, facts: Facts[F]): Step[S, O, F] =
  umpire.Step(outcome, state, facts)
def none[A]: Steps[A] = Nil
def one[A](a: A): Steps[A] = List(a)
def facts0[F]: Facts[F] = Nil
def facts1[F](a: F): Facts[F] = List(a)
