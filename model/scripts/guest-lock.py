#!/usr/bin/env python3
"""Digest each package's own resolved dependency closure from the workspace lockfile."""

import hashlib
import json
import os
from pathlib import Path
import sys
import tomllib


def canonical(value):
    return json.dumps(value, sort_keys=True, separators=(",", ":"))


def closures(metadata, lock):
    packages = {package["id"]: package for package in metadata["packages"]}
    entries = {(entry["name"], entry["version"], entry.get("source")): entry
               for entry in lock.get("package", [])}

    def coordinate(package_id):
        package = packages[package_id]
        # Path packages carry no source, so checkout locations never enter the digest.
        return [package["name"], package["version"], package["source"]]

    def checksum(package_id):
        name, version, source = coordinate(package_id)
        entry = entries.get((name, version, source))
        if source is not None and entry is None:
            raise ValueError(f"no lockfile entry for {name} {version} from {source}")
        return None if entry is None else entry.get("checksum")

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
        members = sorted(([*coordinate(member), checksum(member),
                           sorted((coordinate(dependency) for dependency in edges[member]), key=canonical)]
                          for member in seen), key=canonical)
        record = dict(schema=1, lock=lock.get("version"), packages=members)
        key = packages[root]["name"] + " " + packages[root]["version"]
        digest = hashlib.sha256(canonical(record).encode()).hexdigest()
        # Equal name and version from different sources cannot be told apart by the wrapper.
        result[key] = None if key in result else digest
    return result


def main():
    if len(sys.argv) != 3:
        print("usage: cargo metadata --format-version 1 | guest-lock.py <Cargo.lock> <directory>", file=sys.stderr)
        return 1
    try:
        lockfile, directory = map(Path, sys.argv[1:])
        content = canonical(closures(json.load(sys.stdin), tomllib.loads(lockfile.read_text()))) + "\n"
        # Content-addressed, so concurrent builds sharing a target never read another checkout's map.
        output = directory / ("umpire-lock-" + hashlib.sha256(content.encode()).hexdigest()[:16] + ".json")
        if not output.exists():
            directory.mkdir(parents=True, exist_ok=True)
            temporary = output.with_name(output.name + "." + str(os.getpid()))
            temporary.write_text(content)
            temporary.replace(output)
        print(output)
        return 0
    except (OSError, ValueError, KeyError, TypeError) as error:
        print(f"guest lock closure failed: {error}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())
