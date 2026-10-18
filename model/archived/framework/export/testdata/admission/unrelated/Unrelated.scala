package fixture.admission.unrelated

// A source the declaration-order lint refuses: `first` reads `second` before it is declared.
object Broken:
  val first: Int = second + 1
  val second: Int = 1
