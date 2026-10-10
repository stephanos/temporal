package umpire.irgen

import com.fasterxml.jackson.databind.JsonNode

private[irgen] object TestEquality:
  given CanEqual[JsonNode, JsonNode] = CanEqual.derived
