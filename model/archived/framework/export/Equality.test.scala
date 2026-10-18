package framework.`export`

import com.fasterxml.jackson.databind.JsonNode

private[`export`] object TestEquality:
  given CanEqual[JsonNode, JsonNode] = CanEqual.derived
