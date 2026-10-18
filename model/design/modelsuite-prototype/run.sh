#!/usr/bin/env bash
set -eu
cd "$(dirname "$0")/../../.."
if [ "${1:-}" = --inventory ]; then
  exec timeout 600 env -u BASH_ENV mise exec -- scala-cli run --server=false \
    --workspace "${MODELSUITE_WORKSPACE:-.flow/tmp/fn1951/inventory-workspace}" \
    model/framework/check/Registry.scala model/design/modelsuite-prototype/Inventory.scala \
    --main-class framework.prototype.Inventory
fi
if [ "${1:-}" = --early-refusal ]; then
  exec timeout 600 env -u BASH_ENV mise exec -- scala-cli compile --server=false \
    --dependency org.scalameta::munit:1.2.0 \
    --workspace "${MODELSUITE_WORKSPACE:-.flow/tmp/fn1951/early-refusal-workspace}" \
    model/framework/checker model/framework/samples/Kettle.scala \
    model/design/modelsuite-prototype/ModelSuite.scala model/design/modelsuite-prototype/Early.refusal.scala
fi
exec timeout 600 env -u BASH_ENV mise exec -- scala-cli test --server=false \
  --dependency org.scalameta::munit:1.2.0 \
  --workspace "${MODELSUITE_WORKSPACE:-.flow/tmp/fn1951/prototype-workspace}" \
  model/framework/checker model/framework/samples/Bank.scala model/framework/samples/Kettle.scala \
  model/design/modelsuite-prototype/ModelSuite.scala model/design/modelsuite-prototype/*.test.scala \
  --require-tests --test-only "${1:-framework.prototype.*Test}"
