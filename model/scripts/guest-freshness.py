"""Apply the local/CI prerequisite policy before the generic byte comparison."""

import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tomllib


MODEL = Path(__file__).resolve().parents[1]


class MissingPrerequisite(Exception):
    pass


def check_prerequisites(env):
    if shutil.which("mise", path=env["PATH"]) is None:
        raise MissingPrerequisite("mise")
    pins = tomllib.loads((MODEL.parent / "mise.toml").read_text())["tools"]
    rust = tomllib.loads((MODEL / "rust-toolchain.toml").read_text())["toolchain"]["channel"]
    if pins["rust"] != rust:
        raise ValueError("Rust toolchain pins disagree")
    for tool, commands in [("rust", ["rustc", "cargo"]), ("go", ["go"]), ("protoc", ["protoc"])]:
        result = subprocess.run(
            ["mise", "ls", "--installed", "--json", tool], cwd=MODEL,
            env=env, check=True, capture_output=True, text=True,
        )
        entries = json.loads(result.stdout)
        if not isinstance(entries, list) or any(
            not isinstance(entry, dict) or not isinstance(entry.get("version"), str)
            or not isinstance(entry.get("install_path"), str)
            or type(entry.get("installed")) is not bool or type(entry.get("active")) is not bool
            for entry in entries
        ):
            raise ValueError("invalid installed-tool metadata")
        installed = [entry for entry in entries if entry.get("version") == pins[tool]
                     and entry.get("installed") is True and entry.get("active") is True]
        if not installed:
            raise MissingPrerequisite(f"pinned {tool} {pins[tool]}")
        location = Path(installed[0]["install_path"])
        for command in commands:
            candidates = [location / command, location / "bin" / command]
            if not any(path.is_file() and os.access(path, os.X_OK) for path in candidates):
                raise MissingPrerequisite(command)
            if command == "rustc":
                compiler = next(path for path in candidates if path.is_file() and os.access(path, os.X_OK))
                rustup = compiler.with_name("rustup")
                if rustup.is_file() and compiler.samefile(rustup):
                    toolchains = subprocess.run(
                        [str(rustup), "toolchain", "list"], cwd=MODEL, env=env,
                        check=True, capture_output=True, text=True,
                    )
                    if not any(line.split()[0].startswith(rust + "-")
                               for line in toolchains.stdout.splitlines() if line.strip()):
                        raise MissingPrerequisite(f"pinned Rust {rust}")
    result = subprocess.run(
        ["mise", "exec", "--", "rustc", "--print", "sysroot"], cwd=MODEL,
        env=env, check=True, capture_output=True, text=True,
    )
    if not result.stdout.strip():
        raise ValueError("empty Rust sysroot")
    sysroot = Path(result.stdout.strip())
    if not (sysroot / "lib/rustlib/src/rust/library/Cargo.toml").is_file():
        raise MissingPrerequisite("rust-src")
    if not any((sysroot / "lib/rustlib/wasm32-wasip1/lib").glob("libstd-*.rlib")):
        raise MissingPrerequisite("wasm32-wasip1 target")


def main():
    if len(sys.argv) != 3:
        print("usage: guest-freshness.py <crate> <output>", file=sys.stderr)
        return 1
    env = dict(os.environ, MISE_AUTO_INSTALL="false", MISE_EXEC_AUTO_INSTALL="false")
    try:
        check_prerequisites(env)
    except MissingPrerequisite as error:
        if "CI" not in env:
            print(f"warning: guest freshness skipped: missing {error}", file=sys.stderr)
            return 0
        print(f"guest freshness failed: missing {error} (CI is set)", file=sys.stderr)
        return 1
    except (OSError, ValueError, KeyError, subprocess.CalledProcessError) as error:
        print(f"guest freshness prerequisite check failed: {error}", file=sys.stderr)
        return 1
    return subprocess.call(["bash", str(MODEL / "scripts/guest.sh"), *sys.argv[1:], "--check"], env=env)


if __name__ == "__main__":
    sys.exit(main())
