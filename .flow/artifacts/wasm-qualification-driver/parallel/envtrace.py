#!/usr/bin/env python3
"""Static trace of the working directory and GOMAD_* exports in effect at each
stage call, for the serial reference (executed top to bottom, loop bodies
expanded) and for lib.sh (each step function with its env_* helpers inlined).
Values are compared as source text; see KNOWN_EQUIVALENT for the one spelling
that differs."""
import re
import sys
from pathlib import Path

ASSIGN = re.compile(r'([A-Za-z_][A-Za-z0-9_]*)=("(?:[^"\\]|\\.)*"|\'[^\']*\'|\S*)')
CALL = re.compile(r'^(stage|named_test|named_native_test|rust_test) +("?)([^\s"]+)\2')


def logical_lines(text):
    return [line.strip() for line in text.replace("\\\n", " ").splitlines()]


def apply(line, env, cwd, local):
    """Update env/cwd for one simple command line; returns cwd."""
    for part in re.split(r";\s+|\s+&&\s+", line):
        part = part.strip()
        if part.startswith("export "):
            for name, value in ASSIGN.findall(part[len("export "):]):
                env[name] = value.strip('"')
            for bare in re.findall(r"(?:^|\s)([A-Za-z_]\w*)(?=\s|$)", part[len("export "):]):
                if bare not in env and bare in local:
                    env[bare] = local[bare]
        elif part.startswith("unset "):
            for name in part.split()[1:]:
                env.pop(name, None)
        elif part.startswith("cd "):
            cwd = part.split(None, 1)[1].strip('"')
        else:
            match = re.match(r'^([A-Za-z_]\w*)=(".*"|\S+)$', part)
            if match:
                local[match.group(1)] = match.group(2).strip('"')
    return cwd


def gomad(env):
    return {k: v for k, v in env.items() if k.startswith("GOMAD_")}


def serial_trace(path):
    lines = logical_lines(Path(path).read_text())
    start = lines.index('cd "$QUAL_REPO"')
    env, local, cwd, trace = {}, {}, "", {}
    # Preamble exports before the first stage (same in both drivers by construction).
    for line in lines[:start]:
        if line.startswith(("export GOMAD_", "export GOMAD_WASM_MATCHED")):
            apply(line, env, cwd, local)
    loop = None
    for line in lines[start:]:
        if line.startswith(("for ", "while ")) or line.endswith("; do"):
            if "$(seq 0 31)" in line or "0 1 2 3" in line or "'fixture controlled" in line or "success canary" in line:
                loop = line
        cwd = apply(line, env, cwd, local) if not line.startswith(("#", "if ", "for ", "done")) else cwd
        match = CALL.match(line)
        if match and match.group(3) != "$label":
            trace.setdefault(match.group(3), (cwd, gomad(env)))
    return trace


def lib_trace(path):
    text = Path(path).read_text()
    functions = {name: body for name, body in re.findall(r"^([a-z_0-9]+)\(\) \{\n(.*?)^\}\n", text, re.M | re.S)}
    base = {"GOMAD_STOCK_GO": "$qualification_go", "GOMAD_WASM_STOCK_GO": "$qualification_go",
            "GOMAD_WASM_MATCHED_COLD_MEMORY_GUARD": "1"}

    def walk(name, env, cwd, local, trace):
        for line in logical_lines(functions[name]):
            for word in re.findall(r"(?:^|;\s+)(env_[a-z_]+|workload_inputs|load_primary_state)\b", line):
                cwd = walk(word, env, cwd, local, trace)
            cwd = apply(line, env, cwd, local) if not line.startswith(("#", "if ", "for ", "done", "[[")) else cwd
            match = CALL.match(line)
            if match and match.group(3) != "$label":
                trace.setdefault(match.group(3), (cwd, gomad(env)))
        return cwd

    trace = {}
    for name in functions:
        if name.startswith("step_"):
            walk(name, dict(base), "", {}, trace)
    return trace


# Same value, different spelling: serial reads the profile from the loop's last
# entry; lib.sh states it. Both are `cooperative` (the multi-instance entry).
KNOWN_EQUIVALENT = {("GOMAD_WASM_QUALIFICATION_PROFILE", "$profile", "cooperative"),
                    # serial keeps the shell array element; lib.sh reads the path the
                    # workload-corpus step recorded from it (digest re-checked on load).
                    ("GOMAD_WASM_SQLITE_NATIVE_CATALOGUE_RESULT", "${sqlite_sources[0]}",
                     '$(cat "$QUAL_ROOT/primary-sqlite.path")')}


def compare(reference, lib):
    serial, parallel = serial_trace(reference), lib_trace(lib)
    problems = []
    for label, (cwd, env) in serial.items():
        if label.startswith("native-"):
            continue
        if label not in parallel:
            problems.append(f"{label}: no stage call traced in lib.sh")
            continue
        pcwd, penv = parallel[label]
        if pcwd != cwd:
            problems.append(f"{label}: cwd {pcwd} != serial {cwd}")
        for name in sorted(set(env) | set(penv)):
            a, b = env.get(name), penv.get(name)
            if a != b and (name, a, b) not in KNOWN_EQUIVALENT:
                problems.append(f"{label}: {name} serial={a!r} parallel={b!r}")
    return serial, parallel, problems


if __name__ == "__main__":
    _, _, problems = compare(sys.argv[1], sys.argv[2])
    print("\n".join(problems) or "environment and cwd identical at every stage call")
    sys.exit(1 if problems else 0)
