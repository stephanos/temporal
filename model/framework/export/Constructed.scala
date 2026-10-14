package framework.`export`

import framework.realize.Realization
import scala.util.control.NonFatal

private[`export`] object Constructed:
  def read(root: String, loader: ClassLoader): Realization =
    try
      val split = root.lastIndexOf('.')
      require(split > 0, s"$root has no declaring object")
      val owner = root.take(split).replace("$.", "$")
      val cls = java.lang.Class.forName(owner, true, loader)
      val module = cls.getField("MODULE$").get(null) // scalafix:ok DisableSyntax.null
      val getter = cls.getDeclaredMethod(root.drop(split + 1))
      require(getter.trySetAccessible(), s"$root has no accessible declaring getter")
      getter.invoke(module) match
        case value: Realization => value
        case _ => throw IllegalArgumentException("the declaration is not a constructed realization")
    catch
      case NonFatal(error)     => throw refused(root, error)
      case error: LinkageError => throw refused(root, error)

  private def refused(root: String, error: Throwable): IllegalArgumentException =
    val cause = Iterator.iterate(error)(_.getCause).takeWhile(_ != null).toVector.last
    IllegalArgumentException(
      s"while constructing root $root, declaration $root failed: ${cause.getMessage}",
      error
    )
