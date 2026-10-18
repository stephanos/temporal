package framework.checker

import framework.dsl.*
import scala.deriving.Mirror

private[framework] object DomainRuntime:
  def stateCompare(vars: Vars, left: State, right: State): Int =
    val (x, y) = (vars.owned(left), vars.owned(right))
    vars.schema.vars.indices.iterator
      .map(i => vars.schema.vars(i).data.compare(x.values(i), y.values(i)))
      .find(_ != 0)
      .getOrElse(0)
  def entries[M](channel: Channel[M]): Vector[Delivery[M]] =
    for
      message <- channel.messages.values.toVector;
      redeliveries <- (0 to channel.duplicates).toVector
    yield Delivery(message, redeliveries)

  def deliveryCompare[M](channel: Channel[M], left: Delivery[M], right: Delivery[M]): Int =
    val message = channel.messages.compare(left.message, right.message)
    if message != 0 then message else Integer.compare(left.redeliveries, right.redeliveries)

  def send[M](inbox: Inbox[M], message: M): Inbox[M] =
    val entry = Delivery(message, 0)
    if inbox.channel.order == Order.fifo then inbox.copy(deliveries = inbox.deliveries :+ entry)
    else
      val (before, after) =
        inbox.deliveries.span(e => deliveryCompare(inbox.channel, e, entry) <= 0)
      inbox.copy(deliveries = before ++ (entry :: after))
  def contents[M](channel: Channel[M]): Finite[Inbox[M]] =
    given Data.Structural[M] = channel.messages.structural
    val ordered =
      if channel.order == Order.fifo then channel.entries
      else channel.entries.sortWith(channel.compareDelivery(_, _) < 0)
    val sequences = (0 to channel.capacity).toList.flatMap { n =>
      (1 to n).foldLeft(List(List.empty[Delivery[M]])) { (prefixes, _) =>
        for prefix <- prefixes; e <- ordered.toList yield prefix :+ e
      }
    }
    val held = if channel.order == Order.fifo then sequences
    else
      sequences.filter(s => s.zip(s.drop(1)).forall((a, b) => channel.compareDelivery(a, b) <= 0))
    Finite.of(held.map(Inbox(channel, _))*)

  def inboxCompare[M](message: Data.Structural[M], left: Inbox[M], right: Inbox[M]): Int =
    val owner =
      java.lang.Long.compare(left.channel.comparisonIdentity, right.channel.comparisonIdentity)
    if owner != 0 then owner
    else
      val a = left.deliveries.iterator
      val b = right.deliveries.iterator
      var result = 0
      while result == 0 && a.hasNext && b.hasNext do
        val x = a.next()
        val y = b.next()
        result = message.compare(x.message, y.message)
        if result == 0 then result = Integer.compare(x.redeliveries, y.redeliveries)
      if result != 0 then result else java.lang.Boolean.compare(a.hasNext, b.hasNext)
  def range(high: Int): IndexedSeq[Int] = 0 to high
  def optional[A](domain: Finite[A]): IndexedSeq[Option[A]] = None +: domain.values.map(Some(_))

  def catalog[T](mirror: Mirror.Of[T], fields: List[Finite[?]]): IndexedSeq[T] =
    val values = mirror match
      case _: Mirror.Sum           => fields.flatMap(_.values)
      case product: Mirror.Product =>
        fields
          .foldLeft(List(List.empty[Any]))((prefixes, field) =>
            for prefix <- prefixes; value <- field.values.toList yield prefix :+ value
          )
          .map(values => product.fromProduct(Tuple.fromArray(values.toArray)))
    values.asInstanceOf[List[T]].toIndexedSeq // scalafix:ok DisableSyntax.asInstanceOf

  def compare[T](program: Data.Native[T], left: T, right: T): Int =
    val fields = program.fields
    program.kind match
      case Data.Kind.ordered => program.ordering.get.compare(left, right)
      case Data.Kind.string  =>
        val x = left.asInstanceOf[String] // scalafix:ok DisableSyntax.asInstanceOf
        val y = right.asInstanceOf[String] // scalafix:ok DisableSyntax.asInstanceOf
        val bytes = java.util.Arrays.compareUnsigned(
          x.getBytes(java.nio.charset.StandardCharsets.UTF_8),
          y.getBytes(java.nio.charset.StandardCharsets.UTF_8)
        )
        if bytes != 0 then bytes else x.compareTo(y)
      case Data.Kind.option =>
        (
          left.asInstanceOf[Option[Any]],
          right.asInstanceOf[Option[Any]]
        ) match // scalafix:ok DisableSyntax.asInstanceOf
          case (None, None)       => 0
          case (None, Some(_))    => -1
          case (Some(_), None)    => 1
          case (Some(x), Some(y)) => fields.head.compare(x, y)
          case _ => throw IllegalArgumentException("an optional comparison requires Options")
      case Data.Kind.list | Data.Kind.set | Data.Kind.map =>
        val x = left.asInstanceOf[Iterable[Any]].toList // scalafix:ok DisableSyntax.asInstanceOf
        val y = right.asInstanceOf[Iterable[Any]].toList // scalafix:ok DisableSyntax.asInstanceOf
        val order = Ordering.fromLessThan[Any]((a, b) => fields.head.compare(a, b) < 0)
        program.kind match
          case Data.Kind.list => sequence(x, y, fields.head.compare)
          case Data.Kind.set  =>
            sequence(x.sorted(using order), y.sorted(using order), fields.head.compare)
          case _ =>
            val entries = Ordering.fromLessThan[Any]((a, b) =>
              fields.head.compare(
                a.asInstanceOf[Product].productElement(0),
                b.asInstanceOf[Product].productElement(0)
              ) < 0
            ) // scalafix:ok DisableSyntax.asInstanceOf
            sequence(
              x.sorted(using entries),
              y.sorted(using entries),
              (a, b) =>
                val p = a.asInstanceOf[Product] // scalafix:ok DisableSyntax.asInstanceOf
                val q = b.asInstanceOf[Product] // scalafix:ok DisableSyntax.asInstanceOf
                val key = fields.head.compare(p.productElement(0), q.productElement(0))
                if key != 0 then key
                else fields(1).compare(p.productElement(1), q.productElement(1))
            )
      case Data.Kind.mirrored =>
        program.mirror.get match
          case sum: Mirror.SumOf[T] =>
            val index = sum.ordinal(left)
            val tag = Integer.compare(index, sum.ordinal(right))
            if tag != 0 then tag else fields(index).compare(left, right)
          case _: Mirror.ProductOf[T] =>
            val x = left.asInstanceOf[Product] // scalafix:ok DisableSyntax.asInstanceOf
            val y = right.asInstanceOf[Product] // scalafix:ok DisableSyntax.asInstanceOf
            fields.iterator.zipWithIndex
              .map((field, i) => field.compare(x.productElement(i), y.productElement(i)))
              .find(_ != 0)
              .getOrElse(0)

  private def sequence(left: List[Any], right: List[Any], compare: (Any, Any) => Int): Int =
    val x = left.iterator
    val y = right.iterator
    var result = 0
    while result == 0 && x.hasNext && y.hasNext do result = compare(x.next(), y.next())
    if result != 0 then result else java.lang.Boolean.compare(x.hasNext, y.hasNext)
