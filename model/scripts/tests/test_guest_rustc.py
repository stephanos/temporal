"""Cross-host compiler inputs retain crate and dependency distinctions."""

import importlib.util
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest


SCRIPT = Path(__file__).resolve().parents[1] / "guest-rustc.py"
spec = importlib.util.spec_from_file_location("guest_rustc", SCRIPT)
guest_rustc = importlib.util.module_from_spec(spec)
spec.loader.exec_module(guest_rustc)


class GuestRustc(unittest.TestCase):
    def setUp(self):
        self.scratch = tempfile.TemporaryDirectory(prefix="guest identities ")
        self.addCleanup(self.scratch.cleanup)
        self.root = Path(self.scratch.name)
        self.lock = self.root / "umpire-lock.json"
        self.closures = {"example 1.0.0": "e" * 64, "example 2.0.0": "f" * 64,
                         "another 1.0.0": "0" * 64}
        self.lock.write_text(json.dumps(self.closures))

    def invocation(self, host, metadata="host-hash", dependency=None, **extra):
        workspace = self.root / host / "workspace"
        target = self.root / host / "target"
        out = target / "wasm32-wasip1/guest/build/example-host-hash/out"
        args = ["rustc", "--crate-name", "example", "--edition=2024", "src/lib.rs",
                "--crate-type", "lib", "--emit=dep-info,metadata,link", "-C", "opt-level=3",
                "-C", "metadata=" + metadata, "-C", "extra-filename=-" + host,
                "--out-dir", str(target / "wasm32-wasip1/guest/deps"),
                "--target", "wasm32-wasip1", "--cfg", 'feature="example"',
                "--remap-path-prefix=" + str(workspace) + "=/workspace",
                "--remap-path-prefix=" + str(target) + "=/target"]
        if dependency is not None:
            path = target / ("wasm32-wasip1/guest/deps/libdependency-" + host + ".rlib")
            path.parent.mkdir(parents=True, exist_ok=True)
            path.with_name("dependency-" + host + ".umpire-id").write_text(
                json.dumps({"schema": 1, "identity": dependency}))
            args += ["--extern", "dependency=" + str(path)]
        env = dict(os.environ, UMPIRE_GUEST_WORKSPACE=str(workspace),
                   UMPIRE_GUEST_SYSROOT=str(self.root / host / "rust"),
                   UMPIRE_GUEST_CARGO_HOME=str(self.root / host / "cargo"),
                   UMPIRE_GUEST_TARGET=str(target), UMPIRE_GUEST_LOCK=str(self.lock),
                   UMPIRE_GUEST_COMPILER="rustc 1.99.1 (58cf8f9c3 2026-10-13)",
                   CARGO_MANIFEST_DIR=str(workspace / "crate"), CARGO_PKG_NAME="example",
                   CARGO_PKG_VERSION="1.0.0", OUT_DIR=str(out), **extra)
        return args, env

    def test_host_hashes_and_paths_do_not_change_target_identity(self):
        linux, lp, li = guest_rustc.prepare(*self.invocation("linux", "linux-metadata", "a" * 64))
        macos, mp, mi = guest_rustc.prepare(*self.invocation("macos", "macos-metadata", "a" * 64))
        self.assertEqual(li, mi)
        self.assertIsNotNone(li)
        self.assertNotEqual(lp, mp)
        for args, host in [(linux, "linux"), (macos, "macos")]:
            self.assertNotIn("metadata=" + host + "-metadata", args)
            self.assertIn("metadata=" + li["identity"], args)
            mappings = [a for a in args if a.startswith("--remap-path-prefix=")]
            self.assertTrue(any(a.endswith("=/generated/" + li["identity"]) for a in mappings))

    def test_target_dependency_identity_propagates(self):
        _, _, first = guest_rustc.prepare(*self.invocation("linux", dependency="a" * 64))
        _, _, second = guest_rustc.prepare(*self.invocation("linux", dependency="b" * 64))
        self.assertNotEqual(first, second)

    def test_package_features_profile_and_compiler_remain_distinct(self):
        args, env = self.invocation("linux")
        _, _, baseline = guest_rustc.prepare(args, env)
        variants = [(args, dict(env, CARGO_PKG_VERSION="2.0.0")),
                    (args, dict(env, CARGO_PKG_NAME="another")),
                    (args, dict(env, UMPIRE_GUEST_COMPILER="rustc another")),
                    (args + ["--cfg", 'feature="other"'], env),
                    (args + ["-C", "overflow-checks=on"], env),
                    ([*args[:args.index("opt-level=3")], "opt-level=1",
                      *args[args.index("opt-level=3") + 1:]], env)]
        for variant, environment in variants:
            with self.subTest(args=variant, version=environment["CARGO_PKG_VERSION"]):
                self.assertNotEqual(guest_rustc.prepare(variant, environment)[2], baseline)
        self.lock.write_text(json.dumps(dict(self.closures, **{"example 1.0.0": "1" * 64})))
        self.assertNotEqual(guest_rustc.prepare(args, env)[2], baseline)

    def test_missing_or_ambiguous_lock_closure_refuses(self):
        args, env = self.invocation("linux")
        for closures in [{}, {"example 1.0.0": None}, {"example 1.0.0": "invalid"}]:
            with self.subTest(closures=closures):
                self.lock.write_text(json.dumps(closures))
                with self.assertRaisesRegex(ValueError, "lockfile closure"):
                    guest_rustc.prepare(args, env)

    def test_build_std_units_need_no_workspace_lock_closure(self):
        args, env = self.invocation("linux")
        self.lock.write_text("{}")
        library = self.root / "linux/rust/lib/rustlib/src/rust/library"
        library.mkdir(parents=True)
        (library / "Cargo.lock").write_text(
            'version = 4\n\n[[package]]\nname = "example"\nversion = "1.0.0"\n')
        registry = self.root / "linux/cargo/registry/src/index/example-1.0.0"
        flagged = args + ["-Z", "force-unstable-if-unmarked"]
        for manifest in [library / "std", registry]:
            with self.subTest(manifest=manifest):
                _, _, record = guest_rustc.prepare(flagged, dict(env, CARGO_MANIFEST_DIR=str(manifest)))
                self.assertIsNotNone(record)
        refused = [(args, dict(env, CARGO_MANIFEST_DIR=str(library / "std"))),
                   (args + ["-Zforce-unstable-if-unmarked"], env),
                   (flagged, dict(env, CARGO_MANIFEST_DIR=str(registry), CARGO_PKG_VERSION="2.0.0"))]
        for variant, environment in refused:
            with self.subTest(args=variant[-1], manifest=environment["CARGO_MANIFEST_DIR"]):
                with self.assertRaisesRegex(ValueError, "lockfile closure"):
                    guest_rustc.prepare(variant, environment)

    def test_missing_or_corrupt_dependency_identity_refuses(self):
        args, env = self.invocation("linux", dependency="a" * 64)
        identity = next((self.root / "linux/target").rglob("*.umpire-id"))
        for data in ["{}", '{"schema": 1, "identity": "invalid"}', "not json", None]:
            with self.subTest(data=data):
                if data is None:
                    identity.unlink()
                else:
                    identity.write_text(data)
                with self.assertRaises((ValueError, OSError)):
                    guest_rustc.prepare(args, env)

    def test_native_macro_identity_is_portable_without_changing_its_flags(self):
        identities = []
        for host in ["linux", "macos"]:
            args, env = self.invocation(host)
            target = args.index("--target")
            del args[target:target + 2]
            args[args.index("--crate-type") + 1] = "proc-macro"
            args += ["--cfg", "native_" + host]
            actual, _, identity = guest_rustc.prepare(args, env)
            self.assertEqual(actual, args)
            identities.append(identity["identity"])
        self.assertEqual(identities[0], identities[1])

    def test_conflicting_native_macro_variants_refuse(self):
        args, env = self.invocation("linux", dependency="a" * 64)
        first = next((self.root / "linux/target").rglob("*.umpire-id"))
        record = dict(schema=1, identity="a" * 64, host_variants={"b" * 64: "c" * 64})
        first.write_text(json.dumps(record))
        second = first.with_name("other-linux.umpire-id")
        second.write_text(json.dumps(dict(record, host_variants={"b" * 64: "d" * 64})))
        args += ["--extern", "other=" + str(second.with_name("libother-linux.rlib"))]
        with self.assertRaisesRegex(ValueError, "conflicting native macro"):
            guest_rustc.prepare(args, env)

    def test_probes_pass_through_without_requiring_cargo_environment(self):
        args = ["rustc", "--crate-name", "___", "--print=file-names", "--target", "wasm32-wasip1", "-"]
        self.assertEqual(guest_rustc.prepare(args, {}), (args, None, None))

    def test_compiler_failure_does_not_publish_identity(self):
        args, env = self.invocation("linux")
        compiler = self.root / "compiler"
        compiler.write_text("#!/bin/sh\nexit 23\n")
        compiler.chmod(0o755)
        args[0] = str(compiler)
        result = subprocess.run([sys.executable, str(SCRIPT), *args], env=env,
                                capture_output=True, text=True, check=False)
        self.assertEqual(result.returncode, 23, result.stderr)
        self.assertFalse(list((self.root / "linux/target").rglob("*.umpire-id")))

    def test_identity_is_available_during_pipelined_compilation(self):
        args, env = self.invocation("linux")
        _, path, record = guest_rustc.prepare(args, env)
        compiler = self.root / "compiler"
        compiler.write_text("#!" + sys.executable + "\nimport json\nfrom pathlib import Path\n"
                            + "assert json.loads(Path(" + repr(str(path)) + ").read_text()) == "
                            + repr(record) + "\n")
        compiler.chmod(0o755)
        args[0] = str(compiler)
        result = subprocess.run([sys.executable, str(SCRIPT), *args], env=env,
                                capture_output=True, text=True, check=False)
        self.assertEqual(result.returncode, 0, result.stderr)

    def test_cargo_jobserver_descriptor_reaches_compiler(self):
        reader, writer = os.pipe()
        self.addCleanup(os.close, reader)
        self.addCleanup(os.close, writer)
        compiler = self.root / "compiler"
        compiler.write_text("#!" + sys.executable + "\nimport os\nos.fstat(" + str(reader) + ")\n")
        compiler.chmod(0o755)
        result = subprocess.run([sys.executable, str(SCRIPT), str(compiler), "-V"],
                                pass_fds=(reader,), capture_output=True, text=True, check=False)
        self.assertEqual(result.returncode, 0, result.stderr)


if __name__ == "__main__":
    unittest.main()
