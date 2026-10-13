package temporal.check

import java.nio.file.{Files, Path}
import framework.check.*
import framework.check.equivalence.*

private def historicalPins: Path =
  Tools.here.directory.resolve("model/temporal/check/table-pins/pins.json")

@main def replayTables(ir: String, dumper: String, out: String, receipt: String): Unit =
  val result = InterpreterTables.observe(() =>
    InterpreterTables.replay(Path.of(ir), Path.of(dumper), Path.of(out), historicalPins)
  )
  val file = Path.of(receipt).toAbsolutePath
  Files.createDirectories(file.getParent): Unit
  Files.writeString(file, Equivalence.canonical(result) + "\n")
  if !result.get("status").contains("complete") then
    System.err.println(Equivalence.canonical(result))
    sys.exit(1)

@main def tableMutants(native: String, out: String): Unit =
  val result = InterpreterMutants.probe(
    Path.of(native),
    Path.of(out),
    historicalPins,
    "activity-standalone-race.json",
    "activityProduct",
    "heldDispatch"
  )
  println(Equivalence.canonical(result))
