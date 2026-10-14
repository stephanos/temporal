package user
import kit.*
object Main:
  def sameFileHelper(p: Phase): Int = p match
    case Phase.a => 10
    case Phase.b => 20
  @main def run(): Unit =
    println(probe.Probe.probe((p: Phase) => p match
      case Phase.a => sameFileHelper(p) + Kit.jarHelper(p)
      case Phase.b => if Other.otherFileHelper(p) then 1 else 2))
