package framework.reify

import dotty.tools.dotc.core.Contexts.Context
import dotty.tools.dotc.core.Symbols.*
import dotty.tools.dotc.core.TastyLoader
import dotty.tools.dotc.core.tasty.DottyUnpickler

final private[reify] class CompilerApiError(detail: String)
    extends Exception(s"reify requires Scala 3.9.0's pinned jar-sugar compiler API: $detail")

private[reify] object JarSugar:
  def provider(sym: Symbol)(using Context): DottyUnpickler =
    val version = dotty.tools.dotc.config.Properties.versionNumberString
    if version != "3.9.0" then throw CompilerApiError(s"found compiler $version")
    val unit = sym.compilationUnitInfo
    if unit == null || unit.getClass.getName != "dotty.tools.dotc.core.TastyLoader$$anon$2" then
      throw CompilerApiError("binary CompilationUnitInfo is not TastyLoader$$anon$2")
    try
      val outer = unit.getClass.getDeclaredField("$outer")
      if !outer.trySetAccessible() then
        throw CompilerApiError("TastyLoader$$anon$2.$outer is inaccessible")
      val loader = outer.get(unit)
      if !loader.getClass.eq(classOf[TastyLoader]) then
        throw CompilerApiError("CompilationUnitInfo.$outer is not TastyLoader")
      val accessor = classOf[TastyLoader].getMethod("dotty$tools$dotc$core$TastyLoader$$unpickler")
      accessor.invoke(loader) match
        case provider: DottyUnpickler => provider
        case _ => throw CompilerApiError("TastyLoader's unpickler is not DottyUnpickler")
    catch
      case e: ReflectiveOperationException => throw CompilerApiError(e.toString)
      case e: SecurityException            => throw CompilerApiError(e.toString)
