package samples

// A helper in its own file: an edit to its body alone changes no signature.
object KettleHelpers:
  // the current temperature in the kettle, in Celsius
  def temperature(state: Kettle.Snapshot): Int = state.reading + Kettle.floor + 1
