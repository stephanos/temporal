package user
import kit.*
object Other:
  def otherFileHelper(p: Phase): Boolean = p match
    case Phase.a => true
    case _ => false
