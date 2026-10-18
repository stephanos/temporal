package temporal.migrate

import com.typesafe.config.{ConfigFactory, ConfigParseOptions, ConfigSyntax}
import scala.jdk.CollectionConverters.*

// fn-167.4: the existing files the gap migration reads, parsed apart from the transformation: a
// feature's `<file>.lint.json` (`{"accepted": [{kind, owner, subjects, because}]}`) and the lifter's
// `<file>.waivers.json` (`{"machines": [...], "waivers": [{machine, subject, because}]}`). No new
// format is defined; these are the shapes the lint and the lifter already write.
object Fn167Task4GapInputs:
  private val json = ConfigParseOptions.defaults.setSyntax(ConfigSyntax.JSON)

  def acceptances(text: String): Seq[Fn167Task4Gaps.Acceptance] =
    ConfigFactory
      .parseString(text, json)
      .getConfigList("accepted")
      .asScala
      .toSeq
      .map(c =>
        Fn167Task4Gaps.Acceptance(
          c.getString("kind"),
          c.getString("owner"),
          c.getStringList("subjects").asScala.toSeq,
          c.getString("because")
        )
      )

  def waivers(text: String): Seq[Fn167Task4Gaps.Waiver] =
    ConfigFactory
      .parseString(text, json)
      .getConfigList("waivers")
      .asScala
      .toSeq
      .map(c =>
        Fn167Task4Gaps
          .Waiver(c.getString("machine"), c.getString("subject"), c.getString("because"))
      )
