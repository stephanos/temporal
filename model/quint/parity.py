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
        imports = '  import Worker(taskQueue = "parity") as machine from "./worker"\n  import machine.*'
    else:
        file = "nexus_caller" if name.startswith("nexus") else "standalone_activity"
        vocab = "nexusCallerVocabulary" if name.startswith("nexus") else "activityVocabulary"
        imports = f'  import {vocab}.* from "./{file}"\n  import {name}Table.* from "./{file}"'
    state_of = lambda key: state(key, protocol)
    pairs = []
    for row in table["transitions"]:
        results = []
        for result in row["results"]:
            facts = ", ".join(fact(f, worker) for f in (result["facts"] or []))
            results.append(f'{{ outcome: {constructor(result["outcome"])}, '
                           f'state: {state_of(result["state"])}, facts: [{facts}] }}')
        pairs.append(f'    ({state_of(row["source"])}, {action(row["action"])}) -> '
                     f'[{", ".join(results)}]')
    state_type = "WorkerState" if worker else "ProtocolState" if protocol else "ProductState"
    step_type = "WorkerStep" if worker else "ProtocolStep" if protocol else "ProductStep"
    return f'''module {source}_{name} {{
  import umpire.* from "./umpire"
{imports}

  pure val expected: Map[({state_type}, Action), List[{step_type}]] = Map(
{",\n".join(pairs)}
  )

  run catalogsTest = all {{
    assert(states == {qset(state_of(s) for s in table["states"])}),
    assert(starts == {qset(state_of(s) for s in table["starts"])}),
    assert(ends == {qset(state_of(s) for s in table["ends"])}),
    assert(actionClasses == {qset(action(a) for a in table["actions"])}),
  }}

  run allSuccessorsTest = assert(tuples(states, actionClasses).forall(((s, a)) =>
    steps(s, a) == (if (expected.keys().contains((s, a))) expected.get((s, a)) else [])))

  run reachableStatesTest = assert(reachableFrom(starts, states.size(),
    s => successors(s, actionClasses, steps)) == {qset(state_of(s) for s in table["reachable"])})
}}
'''


def main():
    destination = Path(sys.argv[1])
    tables = json.load(sys.stdin)
    for table in tables:
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
        (destination / f"lean_{name}.qnt").write_text(module(table, "lean"))


if __name__ == "__main__":
    main()
