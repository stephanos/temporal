"""A package's lock identity follows only its own non-dev dependency closure."""

import hashlib
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


def workspace(checkout="/linux/workspace", unrelated=False, dev=False, checksums=None, edges=None):
    packages = {"guest": None, "core": None, "serde": REGISTRY, "windows": REGISTRY}
    # windows is reachable only as a platform-specific build-dependency.
    deps = {"guest": [("core", None, None)],
            "core": [("serde", None, None), ("windows", "build", "cfg(windows)")],
            "serde": [], "windows": []}
    if unrelated:
        packages.update(tool=None, regex=REGISTRY)
        deps.update(tool=[("core", None, None), ("regex", None, None)], regex=[])
    if dev:
        deps["core"].append(("tool", "dev", None))
        packages.setdefault("tool", None)
        deps.setdefault("tool", [])
    for name, extra in (edges or {}).items():
        deps[name] = deps[name] + extra
    checksums = dict({"serde": "c" * 64, "windows": "c" * 64, "regex": "c" * 64}, **(checksums or {}))

    def identity(name):
        if packages[name] is None:
            return f"path+file://{checkout}/{name}#{name}@1.0.0"
        return f"{packages[name]}#{name}@1.0.0"

    metadata = dict(
        packages=[dict(id=identity(name), name=name, version="1.0.0", source=source)
                  for name, source in packages.items()],
        resolve=dict(nodes=[dict(id=identity(name), deps=[
            dict(name=dependency, pkg=identity(dependency), dep_kinds=[dict(kind=kind, target=target)])
            for dependency, kind, target in deps[name]]) for name in packages]))
    lock = dict(version=4, package=[
        dict(name=name, version="1.0.0",
             **({} if source is None else dict(source=source, checksum=checksums[name])),
             dependencies=[dependency for dependency, _, _ in deps[name]])
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

    def test_workspace_unified_optional_edges_of_shared_crates_change_closures(self):
        # cargo metadata unifies features across the workspace: an unrelated crate enabling an
        # optional dependency of a shared crate adds that edge. This over-hashes; it never under-hashes.
        baseline = guest_lock.closures(*workspace())
        unified = guest_lock.closures(*workspace(unrelated=True, edges={"serde": [("regex", None, None)]}))
        self.assertNotEqual(unified["guest 1.0.0"], baseline["guest 1.0.0"])
        self.assertNotEqual(unified["serde 1.0.0"], baseline["serde 1.0.0"])

    def test_checkout_location_does_not_change_closures(self):
        self.assertEqual(guest_lock.closures(*workspace("/linux/workspace")),
                         guest_lock.closures(*workspace("/Users/someone/different checkout")))

    def test_closure_changes_reach_every_dependent(self):
        baseline = guest_lock.closures(*workspace())
        changed = guest_lock.closures(*workspace(checksums={"serde": "d" * 64}))
        for package in ["guest 1.0.0", "core 1.0.0", "serde 1.0.0"]:
            with self.subTest(package=package):
                self.assertNotEqual(changed[package], baseline[package])
        self.assertEqual(changed["windows 1.0.0"], baseline["windows 1.0.0"])

    def test_platform_specific_build_dependencies_are_part_of_the_closure(self):
        baseline = guest_lock.closures(*workspace())
        changed = guest_lock.closures(*workspace(checksums={"windows": "d" * 64}))
        for package in ["guest 1.0.0", "core 1.0.0"]:
            with self.subTest(package=package):
                self.assertNotEqual(changed[package], baseline[package])
        self.assertEqual(changed["serde 1.0.0"], baseline["serde 1.0.0"])

    def test_edge_structure_is_part_of_the_closure(self):
        baseline = guest_lock.closures(*workspace())
        rewired = guest_lock.closures(*workspace(edges={"guest": [("serde", None, None)]}))
        self.assertNotEqual(rewired["guest 1.0.0"], baseline["guest 1.0.0"])
        self.assertEqual(rewired["core 1.0.0"], baseline["core 1.0.0"])

    def test_same_name_and_version_from_two_sources_is_ambiguous(self):
        for source in ["other", None]:
            with self.subTest(source=source):
                metadata, lock = workspace()
                other = f"{source or 'path+file:///linux/workspace/serde'}#serde@1.0.0"
                metadata["packages"].append(dict(id=other, name="serde", version="1.0.0", source=source))
                metadata["resolve"]["nodes"].append(dict(id=other, deps=[]))
                metadata["resolve"]["nodes"][0]["deps"].append(
                    dict(name="serde", pkg=other, dep_kinds=[dict(kind=None, target=None)]))
                if source is not None:
                    lock["package"].append(dict(name="serde", version="1.0.0", source=source, checksum="e" * 64))
                closures = guest_lock.closures(metadata, lock)
                self.assertIsNone(closures["serde 1.0.0"])
                self.assertIsNotNone(closures["guest 1.0.0"])

    def test_registry_package_without_lockfile_entry_refuses(self):
        metadata, lock = workspace()
        lock["package"] = [entry for entry in lock["package"] if entry["name"] != "windows"]
        with self.assertRaisesRegex(ValueError, "no lockfile entry for windows"):
            guest_lock.closures(metadata, lock)
        metadata, lock = workspace()
        git = "git+https://example.invalid/serde?rev=1#0123456789abcdef"
        metadata["packages"][2]["source"] = git
        lock["package"][2] = dict(name="serde", version="1.0.0", source=git, dependencies=[])
        self.assertIsNotNone(guest_lock.closures(metadata, lock)["guest 1.0.0"])

    def test_command_writes_a_content_addressed_closure_map(self):
        metadata, lock = workspace()
        with tempfile.TemporaryDirectory(prefix="guest lock ") as directory:
            lockfile = Path(directory) / "Cargo.lock"
            lockfile.write_text("version = 4\n\n" + "\n".join(
                "[[package]]\n" + "".join(f"{key} = {json.dumps(value)}\n" for key, value in entry.items())
                for entry in lock["package"]))
            target = Path(directory) / "target"
            result = subprocess.run([sys.executable, str(SCRIPT), str(lockfile), str(target)],
                                    input=json.dumps(metadata), capture_output=True, text=True, check=False)
            self.assertEqual(result.returncode, 0, result.stderr)
            output = Path(result.stdout.strip())
            self.assertEqual(output.parent, target)
            self.assertEqual(json.loads(output.read_text()), guest_lock.closures(metadata, lock))
            self.assertEqual(output.name, "umpire-lock-"
                             + hashlib.sha256(output.read_bytes()).hexdigest()[:16] + ".json")
            failed = subprocess.run([sys.executable, str(SCRIPT), str(lockfile), str(target)],
                                    input="not json", capture_output=True, text=True, check=False)
            self.assertEqual(failed.returncode, 1)
            self.assertEqual(failed.stdout, "")


if __name__ == "__main__":
    unittest.main()
