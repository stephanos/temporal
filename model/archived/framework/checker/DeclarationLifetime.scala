package framework.checker

import java.lang.ref.{Reference, ReferenceQueue, WeakReference}

private[checker] object DeclarationLifetime:
  private val released = new ReferenceQueue[AnyRef]

  final private class Key(value: AnyRef) extends WeakReference[AnyRef](value, released):
    private val identity = System.identityHashCode(value)
    override def hashCode: Int = identity
    override def equals(that: Any): Boolean = that match
      case other: Key => (this eq other) || Option(get).exists(_ eq other.get)
      case _          => false

  private val names = new java.util.HashMap[Key, String]

  private def reap(): Unit =
    Iterator
      .continually(Option(released.poll()))
      .takeWhile(_.nonEmpty)
      .foreach: key =>
        names.remove(key.get): Unit

  def name(declaration: AnyRef): Option[String] = synchronized:
    reap()
    val result = Option(names.get(new Key(declaration)))
    Reference.reachabilityFence(declaration)
    result

  def spell(declaration: AnyRef, name: String): Unit = synchronized:
    reap()
    names.putIfAbsent(new Key(declaration), name): Unit
    Reference.reachabilityFence(declaration)
