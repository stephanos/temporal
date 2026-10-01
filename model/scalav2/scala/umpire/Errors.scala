package umpire

import scala.util.boundary
import scala.util.boundary.{break, Label}

/** A model-checking failure, reported against the declaration it belongs to. */
final case class ModelError(declaration: String, message: String):
  override def toString: String = s"$declaration: $message"

/** What every check returns: the value, or the first failure with its declaration named. */
type Checked[A] = Either[ModelError, A]

/**
 * The capability to stop a `checked` block with a failure. `Label` is contravariant, so the label
 * of any `checked[A]` block grants it.
 */
type Fails = Label[Left[ModelError, Nothing]]

/**
 * Direct-style checking: `fail` and `.get` inside the block stop it with the failure, so a check
 * reads as straight-line code rather than a chain of `flatMap`s.
 */
inline def checked[A](inline body: Fails ?=> A): Checked[A] =
  boundary[Checked[A]](label ?=> Right(body(using label)))

def fail(declaration: String, message: String)(using Fails): Nothing =
  break(Left(ModelError(declaration, message)))

extension [A](c: Checked[A])
  /** The value, or stop the enclosing `checked` block with the failure. */
  def get(using Fails): A = c match
    case Right(a) => a
    case Left(e)  => break(Left(e))
