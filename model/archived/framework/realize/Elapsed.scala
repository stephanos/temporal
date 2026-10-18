package framework.realize

import com.google.protobuf.duration.Duration

// Exact construction; the lifter and admission validate the owning field's bounds and presence.
object Elapsed:
  def milliseconds(value: Long): Duration =
    Duration(seconds = value / 1000, nanos = ((value % 1000) * 1000000).toInt)
