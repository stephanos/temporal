package umpire.irgen

import TestEquality.given
import java.nio.file.Files
import scala.collection.mutable
import scala.jdk.CollectionConverters.*
import com.fasterxml.jackson.databind.{JsonNode, ObjectMapper}
import com.fasterxml.jackson.databind.node.ObjectNode
import umpire.check.Tools

private[irgen] object FixturePositions:
  private type Coordinate = (String, Int)
  private val mapper = new ObjectMapper()

  private def positions(node: JsonNode): Map[String, JsonNode] =
    val found = mutable.Map.empty[String, JsonNode]
    def visit(value: JsonNode, pointer: String): Unit =
      if value.isObject then
        value.fields().asScala.foreach { row =>
          val child = pointer + "/" + row.getKey.replace("~", "~0").replace("/", "~1")
          if row.getKey == "position" then found(child) = row.getValue
          else visit(row.getValue, child)
        }
      else if value.isArray then
        value.elements().asScala.zipWithIndex.foreach((row, index) => visit(row, s"$pointer/$index"))
    visit(node, "")
    found.toMap

  def check(name: String, actual: JsonNode, current: Boolean): Unit =
    val original = mapper.readTree(Files.readString(
      Tools.here.directory.resolve(s"model/irgen/testdata/lifts/expected/$name.json")
    ))
    val expected = positions(original)
    val held = positions(actual)
    require(held.keySet == expected.keySet, s"$name source position nodes changed")
    val known = changes.getOrElse(name, Map.empty)
    val seen = mutable.Set.empty[Coordinate]
    val seenSplit = mutable.Set.empty[(String, String)]
    for (pointer, before) <- expected do
      val coordinate = before.path("file").asText() -> before.path("line").asInt()
      val target = if current then
        split.get(name -> pointer) match
          case Some((from, to)) =>
            require(coordinate == from, s"$name $pointer original coordinate changed")
            seenSplit += name -> pointer
            Some(to)
          case None =>
            known.get(coordinate).map { to =>
              seen += coordinate
              to
            }
      else None
      val wanted = target match
        case Some((file, line)) =>
          val changed = before.deepCopy[ObjectNode]()
          changed.put("file", file): Unit
          changed.put("line", line): Unit
          changed
        case None => before
      require(held(pointer) == wanted, s"$name $pointer source position: expected $wanted, actual ${held(pointer)}")
    if current then
      require(seen.toSet == known.keySet, s"$name has unused coordinate mappings")
      require(seenSplit.toSet == split.keySet.filter(_._1 == name), s"$name has unused split-coordinate mappings")
  private def relocate(from: String, to: String, pairs: (Int, Int)*): Map[Coordinate, Coordinate] =
    pairs.map((before, after) => (from -> before) -> (to -> after)).toMap
  private def lines(file: String, pairs: (Int, Int)*): Map[Coordinate, Coordinate] =
    relocate(file, file, pairs*)

  private val changes: Map[String, Map[Coordinate, Coordinate]] = Map(
    "admission" -> (
      lines("model/irgen/testdata/lifts/Admission.scala",
        117 -> 116, 127 -> 126, 130 -> 129, 134 -> 133, 135 -> 134, 137 -> 136, 138 -> 137, 142 -> 141, 143 -> 142, 145 -> 144,
        146 -> 145, 147 -> 146, 148 -> 147, 149 -> 148, 153 -> 152, 154 -> 153, 155 -> 154, 156 -> 155, 157 -> 156, 160 -> 159,
        162 -> 161, 163 -> 162, 164 -> 163, 166 -> 165, 167 -> 166, 168 -> 167, 172 -> 171, 173 -> 172, 174 -> 173, 175 -> 174,
        176 -> 175, 177 -> 176, 178 -> 177, 179 -> 178, 183 -> 182, 185 -> 184, 186 -> 185, 187 -> 186, 188 -> 187, 192 -> 191,
        193 -> 192, 194 -> 193, 195 -> 194, 196 -> 195, 202 -> 201, 203 -> 202, 205 -> 204, 207 -> 206, 208 -> 207, 209 -> 208,
        210 -> 209, 211 -> 210, 216 -> 215, 217 -> 216, 218 -> 217, 220 -> 219, 221 -> 220, 223 -> 222, 224 -> 223, 225 -> 224,
        226 -> 225, 227 -> 226, 232 -> 231, 233 -> 232, 234 -> 233, 235 -> 234, 236 -> 235, 237 -> 236, 238 -> 237, 240 -> 239,
        241 -> 240, 242 -> 241, 243 -> 242, 244 -> 243, 245 -> 244, 246 -> 245, 249 -> 248, 250 -> 249, 255 -> 254, 256 -> 255,
        257 -> 256, 258 -> 257, 260 -> 259, 261 -> 260, 263 -> 262, 266 -> 265, 269 -> 268, 270 -> 269, 272 -> 271, 273 -> 272,
        274 -> 273, 276 -> 275, 277 -> 276, 279 -> 278, 283 -> 282, 286 -> 285, 287 -> 286, 290 -> 289, 298 -> 297, 299 -> 298,
        300 -> 299, 301 -> 300, 304 -> 303, 307 -> 306, 308 -> 307, 311 -> 310, 319 -> 318, 320 -> 319, 321 -> 320, 322 -> 321,
        327 -> 326, 328 -> 327, 330 -> 329, 332 -> 331, 333 -> 332, 340 -> 339, 341 -> 340, 342 -> 341, 343 -> 342, 347 -> 346,
        351 -> 350, 355 -> 354, 357 -> 356, 358 -> 357, 359 -> 358, 360 -> 359, 361 -> 360, 362 -> 361, 364 -> 363
      )
    ),
    "realizations" -> (
      lines("model/irgen/testdata/lifts/Realizations.scala",
        564 -> 574, 568 -> 578, 571 -> 581, 575 -> 585, 586 -> 596, 587 -> 597, 588 -> 598, 590 -> 600, 592 -> 602, 593 -> 603,
        596 -> 606, 598 -> 608, 599 -> 609, 603 -> 613, 604 -> 614, 606 -> 616, 607 -> 617, 609 -> 619, 611 -> 621, 613 -> 623,
        614 -> 624, 615 -> 625, 616 -> 626, 618 -> 628, 620 -> 630, 621 -> 631, 622 -> 632, 625 -> 635, 628 -> 638, 629 -> 639,
        632 -> 642, 633 -> 643, 637 -> 647, 647 -> 657, 648 -> 658, 651 -> 661, 652 -> 662, 656 -> 666, 666 -> 676, 674 -> 684,
        695 -> 705, 701 -> 711, 709 -> 719, 714 -> 724, 727 -> 737, 731 -> 741, 733 -> 743, 735 -> 745, 744 -> 754, 748 -> 758,
        752 -> 762, 756 -> 766, 760 -> 770, 764 -> 774, 774 -> 784, 779 -> 789, 780 -> 790, 785 -> 795, 787 -> 797, 789 -> 799,
        790 -> 800, 792 -> 802, 799 -> 809, 804 -> 814, 806 -> 816, 811 -> 821, 816 -> 826, 818 -> 828, 826 -> 836, 830 -> 840,
        832 -> 842, 834 -> 844, 837 -> 847, 848 -> 858, 850 -> 860, 852 -> 862, 858 -> 868, 866 -> 876, 868 -> 878, 869 -> 879,
        870 -> 880, 872 -> 882, 875 -> 885, 877 -> 887, 880 -> 890, 888 -> 898, 899 -> 909, 904 -> 914, 909 -> 919, 918 -> 928,
        923 -> 933, 936 -> 946, 940 -> 950, 942 -> 952, 944 -> 954, 953 -> 963, 957 -> 967, 966 -> 976, 968 -> 978, 974 -> 984,
        978 -> 988, 981 -> 991, 991 -> 1001, 996 -> 1006, 1004 -> 1014, 1006 -> 1016, 1008 -> 1018
      )
    ),
    "rules" -> (
      lines("model/irgen/testdata/lifts/Rules.scala",
        157 -> 156, 160 -> 159, 162 -> 161, 165 -> 164, 166 -> 165, 170 -> 169, 173 -> 172, 176 -> 175, 177 -> 176, 178 -> 177,
        179 -> 178, 180 -> 179, 181 -> 180, 184 -> 183
      )
    ),
    "rejections" -> (
      lines("model/framework/Syntax.scala",
        55 -> 57, 60 -> 62
      )
    ),
    "taskqueue" -> (
      lines("model/temporal/foundations/taskqueue/TaskQueue.scala",
        96 -> 100, 97 -> 101, 98 -> 102, 100 -> 104, 101 -> 105, 102 -> 106, 110 -> 114, 111 -> 115
      ) ++
      lines("model/temporal/foundations/taskqueue/product/Product.scala",
        17 -> 16, 74 -> 30, 78 -> 32
      ) ++
      relocate("model/temporal/foundations/taskqueue/product/Product.scala", "model/temporal/foundations/taskqueue/product/Queueing.scala",
        28 -> 18, 29 -> 19, 31 -> 21, 32 -> 22, 33 -> 23, 34 -> 24, 37 -> 27, 41 -> 31, 42 -> 32, 43 -> 33,
        44 -> 34, 45 -> 35, 46 -> 36, 47 -> 37, 51 -> 41, 52 -> 42, 53 -> 43, 54 -> 44, 55 -> 45, 59 -> 49,
        60 -> 50, 61 -> 51, 62 -> 52, 63 -> 53, 64 -> 54, 79 -> 65, 80 -> 66, 82 -> 68, 83 -> 69, 85 -> 71,
        86 -> 72
      ) ++
      relocate("model/temporal/foundations/taskqueue/system/System.scala", "model/temporal/foundations/taskqueue/protocol/ForgetfulQueue.scala",
        284 -> 13
      ) ++
      relocate("model/temporal/foundations/taskqueue/system/System.scala", "model/temporal/foundations/taskqueue/protocol/Protocol.scala",
        23 -> 20, 28 -> 26, 31 -> 30, 32 -> 31, 33 -> 32, 34 -> 33, 35 -> 34, 38 -> 36, 44 -> 41, 45 -> 42,
        48 -> 44, 49 -> 45, 54 -> 50, 55 -> 51, 57 -> 53, 58 -> 54, 59 -> 55, 60 -> 56, 62 -> 57, 65 -> 60,
        68 -> 63, 70 -> 64, 147 -> 66
      ) ++
      relocate("model/temporal/foundations/taskqueue/system/System.scala", "model/temporal/foundations/taskqueue/protocol/Queueing.scala",
        73 -> 19, 74 -> 20, 76 -> 22, 77 -> 23, 78 -> 24, 79 -> 25, 82 -> 28, 87 -> 33, 88 -> 34, 90 -> 36,
        91 -> 37, 93 -> 39, 94 -> 40, 96 -> 42, 97 -> 43, 98 -> 44, 99 -> 45, 101 -> 50, 106 -> 55, 111 -> 66,
        112 -> 67, 113 -> 68, 114 -> 69, 119 -> 74, 121 -> 76, 123 -> 78, 132 -> 89, 133 -> 91, 135 -> 93, 148 -> 106,
        149 -> 107, 151 -> 109, 152 -> 110, 154 -> 112, 155 -> 113, 157 -> 115, 158 -> 116, 160 -> 118, 161 -> 119, 163 -> 121,
        164 -> 122, 166 -> 124, 167 -> 125, 169 -> 127, 170 -> 128
      )
    ),
    "retryCapabilities" -> (
      lines("model/irgen/testdata/lifts/RetryCapabilities.scala",
        6 -> 7, 10 -> 11, 16 -> 17, 22 -> 23, 29 -> 30, 32 -> 33, 35 -> 36, 36 -> 37, 37 -> 38, 41 -> 42,
        44 -> 45, 45 -> 46, 48 -> 49, 51 -> 52, 52 -> 53, 53 -> 54, 54 -> 55, 55 -> 56, 56 -> 57, 58 -> 59,
        59 -> 60, 60 -> 61, 61 -> 62, 62 -> 63, 63 -> 64, 64 -> 65, 66 -> 67, 67 -> 68, 68 -> 69, 69 -> 70,
        70 -> 71, 71 -> 72, 72 -> 73, 75 -> 76, 76 -> 77, 77 -> 78, 78 -> 79, 79 -> 80, 80 -> 81, 81 -> 82,
        82 -> 83, 83 -> 84, 85 -> 86, 86 -> 87, 87 -> 88, 88 -> 89, 89 -> 90, 90 -> 91, 92 -> 93, 93 -> 94,
        94 -> 95, 95 -> 96, 96 -> 97, 97 -> 98, 98 -> 99, 99 -> 100, 100 -> 101, 101 -> 102, 102 -> 103, 107 -> 108,
        109 -> 110, 110 -> 111, 113 -> 114, 114 -> 115, 115 -> 116, 118 -> 119, 120 -> 121, 125 -> 126, 127 -> 128, 132 -> 133,
        136 -> 137, 138 -> 139, 139 -> 140, 142 -> 143, 143 -> 144, 144 -> 145, 145 -> 146, 146 -> 147, 149 -> 150, 151 -> 152,
        158 -> 159, 160 -> 161
      ) ++
      relocate("model/temporal/capabilities/Closable.scala", "model/temporal/capabilities/product/Closable.scala",
        23 -> 23, 29 -> 29, 30 -> 30, 31 -> 31, 39 -> 38, 47 -> 48, 48 -> 49, 49 -> 50
      ) ++
      relocate("model/temporal/capabilities/Retryable.scala", "model/temporal/capabilities/system/Retryable.scala",
        28 -> 28, 32 -> 32, 44 -> 44, 45 -> 45, 46 -> 46, 49 -> 49, 60 -> 60, 61 -> 61, 62 -> 62, 65 -> 65,
        77 -> 77, 78 -> 78, 79 -> 79, 82 -> 82, 92 -> 92, 93 -> 93, 94 -> 94, 98 -> 98, 102 -> 102, 103 -> 103,
        104 -> 104, 105 -> 105
      )
    ),
    "retrySelection" -> (
      lines("model/irgen/testdata/lifts/RetryCapabilities.scala",
        16 -> 17, 22 -> 23, 29 -> 30, 32 -> 33, 35 -> 36, 36 -> 37, 37 -> 38, 44 -> 45, 45 -> 46, 75 -> 76,
        76 -> 77, 77 -> 78, 78 -> 79, 79 -> 80, 80 -> 81, 81 -> 82, 82 -> 83, 83 -> 84, 85 -> 86, 86 -> 87,
        87 -> 88, 88 -> 89, 89 -> 90, 90 -> 91, 92 -> 93, 93 -> 94, 94 -> 95, 95 -> 96, 96 -> 97, 97 -> 98,
        98 -> 99, 99 -> 100, 100 -> 101, 101 -> 102, 102 -> 103, 136 -> 137, 138 -> 139, 139 -> 140, 142 -> 143, 143 -> 144,
        144 -> 145, 145 -> 146, 146 -> 147, 149 -> 150, 151 -> 152, 158 -> 159, 160 -> 161, 177 -> 178, 178 -> 179, 182 -> 183,
        184 -> 185, 185 -> 186, 188 -> 189, 189 -> 190, 190 -> 191, 191 -> 192, 192 -> 193, 195 -> 196, 197 -> 198, 204 -> 205,
        206 -> 207
      ) ++
      relocate("model/temporal/capabilities/Retryable.scala", "model/temporal/capabilities/system/Retryable.scala",
        32 -> 32, 44 -> 44, 45 -> 45, 46 -> 46, 49 -> 49, 60 -> 60, 61 -> 61, 62 -> 62, 65 -> 65, 77 -> 77,
        78 -> 78, 79 -> 79, 82 -> 82, 92 -> 92, 93 -> 93, 94 -> 94, 98 -> 98, 102 -> 102, 103 -> 103, 104 -> 104,
        105 -> 105
      )
    ),
    "deadlineCapabilities" -> (
      relocate("model/temporal/capabilities/Expirable.scala", "model/temporal/capabilities/system/Expirable.scala",
        49 -> 49, 51 -> 51, 61 -> 61, 62 -> 62, 65 -> 65, 77 -> 77, 78 -> 78, 79 -> 79, 80 -> 80, 83 -> 83,
        96 -> 96, 97 -> 97, 98 -> 98, 99 -> 99, 102 -> 102, 115 -> 115, 116 -> 116, 117 -> 117, 118 -> 118
      )
    )
  )

  private val split: Map[(String, String), (Coordinate, Coordinate)] = Map(
    ("taskqueue", "/functions/30/body/position") -> (("model/temporal/foundations/taskqueue/system/System.scala", 107), ("model/temporal/foundations/taskqueue/protocol/Queueing.scala", 56)),
    ("taskqueue", "/functions/30/body/list/items/0/position") -> (("model/temporal/foundations/taskqueue/system/System.scala", 107), ("model/temporal/foundations/taskqueue/protocol/Queueing.scala", 57)),
    ("taskqueue", "/functions/30/body/list/items/0/construct/args/0/position") -> (("model/temporal/foundations/taskqueue/system/System.scala", 107), ("model/temporal/foundations/taskqueue/protocol/Queueing.scala", 58)),
    ("taskqueue", "/functions/30/body/list/items/0/construct/args/2/position") -> (("model/temporal/foundations/taskqueue/system/System.scala", 107), ("model/temporal/foundations/taskqueue/protocol/Queueing.scala", 60)),
    ("taskqueue", "/functions/30/body/list/items/0/construct/args/2/list/items/0/position") -> (("model/temporal/foundations/taskqueue/system/System.scala", 107), ("model/temporal/foundations/taskqueue/protocol/Queueing.scala", 60)),
    ("taskqueue", "/functions/30/body/list/items/0/construct/args/3/position") -> (("model/temporal/foundations/taskqueue/system/System.scala", 107), ("model/temporal/foundations/taskqueue/protocol/Queueing.scala", 57)),
    ("taskqueue", "/functions/32/body/list/items/0/construct/args/1/position") -> (("model/temporal/foundations/taskqueue/system/System.scala", 100), ("model/temporal/foundations/taskqueue/protocol/Queueing.scala", 46)),
    ("taskqueue", "/functions/32/body/list/items/0/construct/args/1/copy/updates/0/value/position") -> (("model/temporal/foundations/taskqueue/system/System.scala", 100), ("model/temporal/foundations/taskqueue/protocol/Queueing.scala", 47)),
    ("taskqueue", "/functions/32/body/list/items/0/construct/args/1/copy/updates/1/value/position") -> (("model/temporal/foundations/taskqueue/system/System.scala", 100), ("model/temporal/foundations/taskqueue/protocol/Queueing.scala", 48)),
    ("taskqueue", "/functions/32/body/list/items/0/construct/args/1/copy/updates/1/value/call/args/0/position") -> (("model/temporal/foundations/taskqueue/system/System.scala", 100), ("model/temporal/foundations/taskqueue/protocol/Queueing.scala", 48)),
    ("taskqueue", "/functions/32/body/list/items/0/construct/args/1/copy/updates/1/value/call/args/0/field/base/position") -> (("model/temporal/foundations/taskqueue/system/System.scala", 100), ("model/temporal/foundations/taskqueue/protocol/Queueing.scala", 48)),
    ("taskqueue", "/functions/32/body/list/items/0/construct/args/1/copy/base/position") -> (("model/temporal/foundations/taskqueue/system/System.scala", 100), ("model/temporal/foundations/taskqueue/protocol/Queueing.scala", 46))
  )
