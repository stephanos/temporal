"""A package's lock identity follows only its own non-dev dependency closure."""

import importlib.util
import json
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest


SCRIPT = Path(__file__).resolve().parents[1] / "guest-lock.py"
spec = importlib.util.spec_from_file_location("guest_lock", SCRIPT)
guest_lock = importlib.util.module_from_spec(spec)
spec.loader.exec_module(guest_lock)
REGISTRY = "registry+https://github.com/rust-lang/crates.io-index"


def workspace(checkout="/linux/workspace", unrelated=False, dev=False, checksum="c" * 64):
    packages = {"guest": None, "core": None, "serde": REGISTRY, "windows": REGISTRY}
    deps = {"guest": [("core", None)], "core": [("serde", None), ("windows", "build")],
            "serde": [], "windows": []}
    if unrelated:
        packages.update(tool=None, regex=REGISTRY)
        deps.update(tool=[("core", None), ("regex", None)], regex=[])
    if dev:
        deps["core"].append(("tool", "dev"))
        packages.setdefault("tool", None)
        deps.setdefault("tool", [])

    def identity(name):
        if packages[name] is None:
            return f"path+file://{checkout}/{name}#{name}@1.0.0"
        return f"{packages[name]}#{name}@1.0.0"

    metadata = dict(
        packages=[dict(id=identity(name), name=name, version="1.0.0", source=source)
                  for name, source in packages.items()],
        resolve=dict(nodes=[dict(id=identity(name), deps=[
            dict(name=dependency, pkg=identity(dependency), dep_kinds=[dict(kind=kind, target=None)])
            for dependency, kind in deps[name]]) for name in packages]))
    lock = dict(version=4, package=[
        dict(name=name, version="1.0.0", **({} if source is None else dict(source=source, checksum=checksum)),
             dependencies=[dependency for dependency, _ in deps[name]])
        for name, source in packages.items()])
    return metadata, lock


class GuestLock(unittest.TestCase):
    def test_unrelated_workspace_packages_and_dev_dependencies_do_not_change_closures(self):
        baseline = guest_lock.closures(*workspace())
        for variant in [workspace(unrelated=True), workspace(dev=True),
                        workspace(unrelated=True, dev=True)]:
            closures = guest_lock.closures(*variant)
            for package in ["guest 1.0.0", "core 1.0.0", "serde 1.0.0"]:
                with self.subTest(package=package):
                    self.assertEqual(closures[package], baseline[package])

    def test_checkout_location_does_not_change_closures(self):
        self.assertEqual(guest_lock.closures(*workspace("/linux/workspace")),
                         guest_lock.closures(*workspace("/Users/someone/different checkout")))

    def test_closure_changes_reach_every_dependent(self):
        baseline = guest_lock.closures(*workspace())
        changed = guest_lock.closures(*workspace(checksum="d" * 64))
        for package in baseline:
            with self.subTest(package=package):
                self.assertNotEqual(changed[package], baseline[package])
        metadata, lock = workspace()
        metadata["resolve"]["nodes"][0]["deps"] = []
        self.assertNotEqual(guest_lock.closures(metadata, lock)["guest 1.0.0"], baseline["guest 1.0.0"])
        self.assertEqual(guest_lock.closures(metadata, lock)["core 1.0.0"], baseline["core 1.0.0"])

    def test_same_name_and_version_from_two_sources_is_ambiguous(self):
        metadata, lock = workspace()
        metadata["packages"].append(dict(id="other#serde@1.0.0", name="serde", version="1.0.0", source="other"))
        metadata["resolve"]["nodes"].append(dict(id="other#serde@1.0.0", deps=[]))
        self.assertIsNone(guest_lock.closures(metadata, lock)["serde 1.0.0"])

    def test_command_writes_the_closure_map(self):
        metadata, lock = workspace()
        with tempfile.TemporaryDirectory(prefix="guest lock ") as directory:
            lockfile = Path(directory) / "Cargo.lock"
            lockfile.write_text("version = 4\n\n" + "\n".join(
                "[[package]]\n" + "".join(f"{key} = {json.dumps(value)}\n" for key, value in entry.items())
                for entry in lock["package"]))
            output = Path(directory) / "umpire-lock.json"
            result = subprocess.run([sys.executable, str(SCRIPT), str(lockfile), str(output)],
                                    input=json.dumps(metadata), capture_output=True, text=True, check=False)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(json.loads(output.read_text()), guest_lock.closures(metadata, lock))
            failed = subprocess.run([sys.executable, str(SCRIPT), str(lockfile), str(output)],
                                    input="not json", capture_output=True, text=True, check=False)
            self.assertEqual(failed.returncode, 1)


if __name__ == "__main__":
    unittest.main()
