#!/usr/bin/env python3
"""Check compiled stress isolation without running a workload matrix."""
import argparse
import json
import os
from pathlib import Path
import subprocess


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--ordinary", type=Path, required=True)
    parser.add_argument("--stress", type=Path, required=True)
    parser.add_argument("--diagnostic", type=Path)
    args = parser.parse_args()
    binaries = {name: path.resolve(strict=True) for name, path in vars(args).items() if path is not None}
    marker = b"stress jitter requires host entropy"
    if marker in binaries["ordinary"].read_bytes():
        raise AssertionError("ordinary release contains the stress delay implementation")
    if marker not in binaries["stress"].read_bytes():
        raise AssertionError("stress build does not contain the delay implementation")
    observations = []
    for role, binary in binaries.items():
        environment = dict(os.environ, GOMAD_WASM_STRESS="1")
        environment.pop("GOMAD_WASM_STRESS_JITTER", None)
        command = [str(binary), "stress", "--spec", "/nonexistent-gomad-stress-spec"]
        result = subprocess.run(command, env=environment, capture_output=True, timeout=30, check=False)
        assert result.returncode == 2, (command, result.returncode, result.stderr)
        refused = b"identity-bound helper-stress-jitter build" in result.stderr
        assert refused == (role != "stress"), result.stderr
        command = [str(binary), "qualify", "--spec", "/nonexistent-gomad-qualification-spec"]
        result = subprocess.run(command, env=environment, capture_output=True, timeout=30, check=False)
        assert result.returncode == 2, (command, result.returncode, result.stderr)
        refused = b"qualification refuses" in result.stderr
        assert refused == (role != "ordinary"), result.stderr
        observations.append(dict(role=role, stress_exit=2, qualification_exit=2))
    print(json.dumps(dict(schema="gomad.stress-role-check/v1", passed=True, observations=observations), sort_keys=True))


if __name__ == "__main__":
    main()
