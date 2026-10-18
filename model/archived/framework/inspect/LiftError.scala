package framework.inspect

final case class LiftError(position: String, message: String)
    extends Exception(s"$position: $message")
