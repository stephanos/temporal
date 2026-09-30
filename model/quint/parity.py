#!/usr/bin/env python3
"""Turn reference tables into independent, exhaustive Quint test expectations."""

import json
from pathlib import Path
import sys


def constructor(key):
    return key[0].upper() + key[1:]


def state(key, protocol):
    parts = key.split("-")
    if not protocol:
        if len(parts) != 1:
            raise ValueError(f"unexpected product state: {key}")
        return "{ phase: " + constructor(key) + " }"
    phase, attempts, close, schedule, start = parts
    return (f"{{ phase: {constructor(phase)}, attempts: {int(attempts)}, "
            f"scheduleToClose: {constructor(close)}, scheduleToStart: {constructor(schedule)}, "
            f"startToClose: {constructor(start)} }}")


def action(key):
    parts = key.split("-")
    name, inputs = parts[0], parts[1:]
    if name in ("schedule", "start"):
        close, schedule, start = inputs
        return (f"{constructor(name)}({{ scheduleToClose: {constructor(close)}, "
                f"scheduleToStart: {constructor(schedule)}, startToClose: {constructor(start)} }})")
    if name == "handlerReply":
        if inputs[0] == "handlerError":
            return f"HandlerReply(HandlerError({inputs[1]}))"
        return f"HandlerReply({constructor(inputs[0])})"
    if name == "complete":
        return f"Complete(Resolved{constructor(inputs[0])})"
    if name == "attemptResult":
        if inputs[0] == "failed":
            return f"AttemptResultAction(AttemptFailed({inputs[1]}))"
        return f"AttemptResultAction(Attempt{constructor(inputs[0])})"
    if name == "control":
        return f"ControlAction({constructor(inputs[0])})"
    if inputs:
        raise ValueError(f"unrecognized action inputs: {key}")
    return "TimeoutFired" if name == "timeout" else constructor(name)


def fact(key, worker):
    if worker:
        raise ValueError(f"worker unexpectedly emits a fact: {key}")
    parts = key.split("-")
    if len(parts) == 1:
        return "AttemptCount" if key == "attemptCount" else constructor(key)
    name, deadline = parts
    if name not in ("nexusOperationTimedOut", "statusTimedOut"):
        raise ValueError(f"unrecognized fact inputs: {key}")
    return f"{constructor(name)}(By{constructor(deadline)})"


def qset(values):
    return "Set(" + ", ".join(values) + ")"


def module(table, source):
    name = table["machine"]
    protocol = name.endswith("Protocol")
    worker = name == "workerPolling"
    if worker:
        imports = '  import Worker.* from "./worker"'
    else:
        file = "nexus_caller_vocabulary" if name.startswith("nexus") else "activity_vocabulary"
        vocab = "nexusCallerVocabulary" if name.startswith("nexus") else "activityVocabulary"
        table_file = {"nexusProduct": "nexus_product_table", "nexusProtocol": "nexus_protocol_table",
                      "activityProduct": "activity_product_table", "activityProtocol": "activity_protocol_table"}[name]
        imports = f'  import {vocab}.* from "./{file}"\n  import {name}Table.* from "./{table_file}"'
    state_of = lambda key: state(key, protocol)
    by_source = {s: [] for s in table["states"]}
    for row in table["transitions"]:
        results = []
        for result in row["results"]:
            facts = ", ".join(fact(f, worker) for f in (result["facts"] or []))
            results.append(f'{{ outcome: {constructor(result["outcome"])}, '
                           f'state: {state_of(result["state"])}, facts: [{facts}] }}')
        by_source[row["source"]].append(f'      {action(row["action"])} -> [{", ".join(results)}]')
    step_type = "WorkerStep" if worker else "ProtocolStep" if protocol else "ProductStep"
    checks = []
    for key, pairs in by_source.items():
        rows = ",\n".join(pairs)
        checks.append(f'''  run successors_{key.replace("-", "_")}Test = {{
    val expected: Action -> List[{step_type}] = Map(
{rows}
    )
    assert(actionClasses.forall(a => steps({state_of(key)}, a) ==
      (if (expected.keys().contains(a)) expected.get(a) else [])))
  }}''')
    successors_checks = "\n\n".join(checks)
    return f'''module {source}_{name} {{
  import umpire.* from "./umpire"
{imports}

  run catalogsTest = all {{
    assert(states == {qset(state_of(s) for s in table["states"])}),
    assert(starts == {qset(state_of(s) for s in table["starts"])}),
    assert(ends == {qset(state_of(s) for s in table["ends"])}),
    assert(actionClasses == {qset(action(a) for a in table["actions"])}),
  }}

{successors_checks}

  run reachableStatesTest = assert(reachableFrom(starts, states.size(),
    s => successors(s, actionClasses, steps)) == {qset(state_of(s) for s in table["reachable"])})
}}
'''


def main():
    destination = Path(sys.argv[1])
    tables = json.load(sys.stdin)
    for table in tables:
        if table["machine"] == "polling":
            table["machine"] = "workerPolling"
        name = table["machine"]
        if name not in ("nexusProduct", "nexusProtocol", "activityProduct",
                        "activityProtocol", "workerPolling"):
            raise ValueError(f"unsupported table: {name}")
        (destination / f"go_{name}.qnt").write_text(module(table, "go"))
    reference = Path(__file__).resolve().parent.parent / "go/parity/testdata/lean"
    for name, filename in [("nexusProduct", "table-nexusProduct.json"),
                           ("nexusProtocol", "table-nexusProtocol.json"),
                           ("activityProduct", "activity-table-activityProduct.json"),
                           ("workerPolling", "table-workerPolling.json")]:
        table = json.loads((reference / filename).read_text())
        table["machine"] = name
        (destination / f"lean_{name}.qnt").write_text(module(table, "lean"))


if __name__ == "__main__":
    main()
