package umpire.irgen

// The lifter's rules that land switched off, so the Models they would refuse can be migrated first
// (Batch 1d): each is named here, documented in model/README.md, and turned on for good by adding it
// to `defaults` once every Model keeps it, after which its switch goes. `UMPIRE_LIFT_RULES`, a
// comma-separated list of names, turns more on for one run, as the lifter's tests of each rule do.
//
//   - `facets-only` (fn-161 R8): a declared machine binds its actions and declares its effects only
//     in facets, `object rules extends Facets(...)`; derived machines and compositions are exempt.
//   - `facet-vocabulary` (fn-161 R9): every facet takes its name from its feature's facet
//     vocabulary, `enum FacetName`, and an action bound at the Product and the Protocol level is
//     owned by the same-named facet at both.
//   - `layout` (fn-160 R9-R13): the flat prefix layout and the Protocol level of model/README.md,
//     "The layout of a feature".
object Switches:
  val defaults: Set[String] = Set.empty

  val known: Set[String] = Set("facets-only", "facet-vocabulary", "layout")

  lazy val on: Set[String] =
    val asked =
      sys.env.get("UMPIRE_LIFT_RULES").toSeq.flatMap(_.split(',')).map(_.trim).filter(_.nonEmpty)
    for name <- asked if !known(name) do
      sys.error(s"UMPIRE_LIFT_RULES names $name, which is none of ${known.toSeq.sorted.mkString(", ")}")
    defaults ++ asked

  def facetsOnly: Boolean = on("facets-only")
  def facetVocabulary: Boolean = on("facet-vocabulary")
  def layout: Boolean = on("layout")
