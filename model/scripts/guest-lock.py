#!/usr/bin/env python3
"""Digest each package's own resolved dependency closure from the workspace lockfile."""

import hashlib
import json
import os
from pathlib import Path
import sys
import tomllib


def closures(metadata, lock):
    packages = {package["id"]: package for package in metadata["packages"]}
    checksums = {(entry["name"], entry["version"], entry.get("source")): entry.get("checksum")
                 for entry in lock.get("package", [])}

    def coordinate(package_id):
        package = packages[package_id]
        # Path packages carry no source, so checkout locations never enter the digest.
        return [package["name"], package["version"], package["source"]]

    # Dev-dependencies never reach a library build; all platforms keep the graph host-independent.
    edges = {node["id"]: sorted({dependency["pkg"] for dependency in node["deps"]
                                 if any(kind["kind"] != "dev" for kind in dependency["dep_kinds"])})
             for node in metadata["resolve"]["nodes"]}
    result = {}
    for root in edges:
        seen, pending = set(), [root]
        while pending:
            current = pending.pop()
            if current not in seen:
                seen.add(current)
                pending.extend(edges[current])
        entries = sorted([*coordinate(member), checksums.get(tuple(coordinate(member))),
                          sorted(coordinate(dependency) for dependency in edges[member])]
                         for member in seen)
        record = dict(schema=1, lock=lock.get("version"), packages=entries)
        key = packages[root]["name"] + " " + packages[root]["version"]
        digest = hashlib.sha256(json.dumps(record, sort_keys=True, separators=(",", ":")).encode()).hexdigest()
        # Equal name and version from different sources cannot be told apart by the wrapper.
        result[key] = None if key in result else digest
    return result


def main():
    if len(sys.argv) != 3:
        print("usage: cargo metadata --format-version 1 | guest-lock.py <Cargo.lock> <output>", file=sys.stderr)
        return 1
    try:
        lockfile, output = map(Path, sys.argv[1:])
        result = closures(json.load(sys.stdin), tomllib.loads(lockfile.read_text()))
        temporary = output.with_name(output.name + "." + str(os.getpid()))
        temporary.write_text(json.dumps(result, sort_keys=True) + "\n")
        temporary.replace(output)
        return 0
    except (OSError, ValueError, KeyError, TypeError) as error:
        print(f"guest lock closure failed: {error}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())
