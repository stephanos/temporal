#!/usr/bin/env python3
"""Normalize guest compiler inputs without changing Cargo's cache identities."""

import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import sys


def options(arguments):
    result = {}
    names = {"--crate-name", "--crate-type", "--edition", "--out-dir", "--target",
             "--emit", "--cfg", "--extern", "-C", "-Z"}
    index = 1
    while index < len(arguments):
        argument = arguments[index]
        if argument in names:
            index += 1
            if index == len(arguments):
                raise ValueError(f"missing value for {argument}")
            result.setdefault(argument, []).append(arguments[index])
        elif argument.startswith(("-C", "-Z")):
            result.setdefault(argument[:2], []).append(argument[2:])
        elif argument.partition("=")[0] in names:
            name, _, value = argument.partition("=")
            result.setdefault(name, []).append(value)
        index += 1
    return result


def identity_file(artifact):
    stem = artifact.stem
    if stem.startswith("lib"):
        stem = stem[3:]
    return artifact.with_name(stem + ".umpire-id")


def prepare(arguments, environment):
    parsed = options(arguments)
    types = sorted(kind for value in parsed.get("--crate-type", []) for kind in value.split(","))
    target = parsed.get("--target", [None])[-1]
    if "--emit" not in parsed or (target != "wasm32-wasip1" and "proc-macro" not in types):
        return arguments, None, None
    manifest = environment["CARGO_MANIFEST_DIR"]
    mappings = [(environment["UMPIRE_GUEST_WORKSPACE"], "/workspace"),
                (environment["UMPIRE_GUEST_SYSROOT"], "/rust/sysroot"),
                (str(Path(environment["UMPIRE_GUEST_CARGO_HOME"]) / "registry"), "/cargo/registry"),
                (environment["UMPIRE_GUEST_TARGET"], "/target")]

    def normalized(value):
        value = os.path.abspath(value)
        for old, new in reversed(mappings):
            if value == old or value.startswith(old + os.sep):
                return new + value[len(old):]
        raise ValueError(f"guest source is outside the remapped roots: {value}")

    name = parsed["--crate-name"][-1]
    source = next(argument for argument in arguments if argument.endswith(".rs"))
    codegen = [value for value in parsed.get("-C", [])
               if value.partition("=")[0] not in {"metadata", "extra-filename", "incremental"}]
    cfg = parsed.get("--cfg", [])
    dependencies = []
    host_variants = {}
    if target == "wasm32-wasip1":
        for external in parsed.get("--extern", []):
            alias, separator, path = external.partition("=")
            if not separator:
                raise ValueError(f"guest dependency has no artifact path: {external}")
            record = json.loads(identity_file(Path(path)).read_text())
            if (not isinstance(record, dict) or record.get("schema") != 1
                    or not isinstance(record.get("identity"), str)
                    or re.fullmatch(r"[0-9a-f]{64}", record["identity"]) is None):
                raise ValueError(f"invalid guest dependency identity: {path}")
            dependencies.append((alias, record["identity"]))
            variants = record.get("host_variants", {})
            if not isinstance(variants, dict) or any(
                not isinstance(key, str) or re.fullmatch(r"[0-9a-f]{64}", key) is None
                or not isinstance(value, str) or re.fullmatch(r"[0-9a-f]{64}", value) is None
                for key, value in variants.items()
            ):
                raise ValueError(f"invalid guest native macro identities: {path}")
            for key, value in variants.items():
                if key in host_variants and host_variants[key] != value:
                    raise ValueError(f"conflicting native macro variants in guest dependency: {alias}")
                host_variants[key] = value
    else:
        cfg = [value for value in cfg if value.startswith('feature=')]
        codegen = [value for value in codegen if value.partition("=")[0] in
                   {"opt-level", "debug-assertions", "overflow-checks", "panic", "codegen-units", "lto"}]
    key = dict(schema=1, compiler=environment["UMPIRE_GUEST_COMPILER"],
               lock=hashlib.sha256(Path(environment["UMPIRE_GUEST_LOCKFILE"]).read_bytes()).hexdigest(),
               package=[environment["CARGO_PKG_NAME"], environment["CARGO_PKG_VERSION"], normalized(manifest)],
               name=name, source=normalized(os.path.join(manifest, source)), types=types,
               target="wasm32-wasip1" if target == "wasm32-wasip1" else "proc-macro-host",
               edition=parsed.get("--edition", ["2015"])[-1], cfg=sorted(set(cfg)), codegen=codegen,
               unstable=parsed.get("-Z", []) if target == "wasm32-wasip1" else [],
               dependencies=sorted(dependencies))
    identity = hashlib.sha256(json.dumps(key, sort_keys=True, separators=(",", ":")).encode()).hexdigest()
    extra = next((value.partition("=")[2] for value in parsed.get("-C", [])
                  if value.startswith("extra-filename=")), "")
    path = Path(parsed["--out-dir"][-1]) / (name + extra + ".umpire-id")
    if target != "wasm32-wasip1":
        metadata = [value for value in parsed.get("-C", []) if value.startswith("metadata=")]
        host_variants[identity] = hashlib.sha256(json.dumps(metadata).encode()).hexdigest()
        return arguments, path, dict(schema=1, identity=identity, host_variants=host_variants)
    command = []
    index = 0
    while index < len(arguments):
        argument = arguments[index]
        if argument == "-C" and arguments[index + 1].startswith("metadata="):
            index += 2
            continue
        if argument.startswith("-Cmetadata="):
            index += 1
            continue
        command.append(argument)
        index += 1
    command.extend(["-C", "metadata=" + identity])
    if out := environment.get("OUT_DIR"):
        command.append("--remap-path-prefix=" + out + "=/generated/" + identity)
    return command, path, dict(schema=1, identity=identity, host_variants=host_variants)


def main():
    try:
        command, path, record = prepare(sys.argv[1:], os.environ)
        if previous := os.environ.get("UMPIRE_GUEST_PREVIOUS_WRAPPER"):
            command = [previous, *command]
        if path is not None:
            # Cargo can start dependents as soon as rustc emits metadata, before it exits.
            path.parent.mkdir(parents=True, exist_ok=True)
            temporary = path.with_name(path.name + "." + str(os.getpid()))
            temporary.write_text(json.dumps(record) + "\n")
            temporary.replace(path)
        try:
            result = subprocess.run(command, check=False, close_fds=False)
        except OSError:
            if path is not None:
                path.unlink(missing_ok=True)
            raise
        if result.returncode:
            if path is not None:
                path.unlink(missing_ok=True)
            return result.returncode if result.returncode > 0 else 128 - result.returncode
        return 0
    except (OSError, ValueError, KeyError, IndexError, StopIteration) as error:
        print(f"guest compiler input failed: {error}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())
