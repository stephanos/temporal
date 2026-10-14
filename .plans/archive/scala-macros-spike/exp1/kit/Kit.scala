//> using scala 3.9.0
package kit
enum Phase:
  case a, b
object Kit:
  def jarHelper(p: Phase): Int = p match
    case Phase.a => 1
    case Phase.b => 2
