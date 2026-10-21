"""Exercise freshness policy through the real builder and Make gate seam."""

import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import tomllib
import unittest


REPO = Path(__file__).resolve().parents[3]
SCRIPT = REPO / "model/scripts/guest-freshness.py"


class GuestFreshness(unittest.TestCase):
    def setUp(self):
        self.scratch = tempfile.TemporaryDirectory(prefix="guest-freshness-")
        self.addCleanup(self.scratch.cleanup)
        self.root = Path(self.scratch.name)
        self.python_only = self.root / "python-only"
        self.python_only.mkdir()
        (self.python_only / "python3").symlink_to(sys.executable)
        self.pins = tomllib.loads((REPO / "mise.toml").read_text())["tools"]
        for tool, names in [("rust", ["rustc", "cargo"]), ("go", ["go"]), ("protoc", ["protoc"])]:
            for name in names:
                path = self.root / tool / "bin" / name
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text("#!/bin/sh\nexit 0\n")
                path.chmod(0o755)
        self.source = self.root / "rust/lib/rustlib/src/rust/library/Cargo.toml"
        self.source.parent.mkdir(parents=True)
        self.source.write_text("[workspace]\n")
        self.target = self.root / "rust/lib/rustlib/wasm32-wasip1/lib/libstd-control.rlib"
        self.target.parent.mkdir(parents=True)
        self.target.touch()
        self.artifact = self.root / "built.wasm"
        self.artifact.write_bytes(b"\0asm\1\0\0\0")
        self.output = self.root / "committed.wasm"
        self.output.write_bytes(self.artifact.read_bytes())
        mise = self.root / "mise"
        mise.write_text(
            "#!/usr/bin/env python3\n"
            "import json, os, sys\n"
            "from pathlib import Path\n"
            "args = sys.argv[1:]\n"
            "root = Path(os.environ['FIXTURE'])\n"
            "assert os.environ['MISE_AUTO_INSTALL'] == 'false'\n"
            "assert os.environ['MISE_EXEC_AUTO_INSTALL'] == 'false'\n"
            "if args[:3] == ['ls', '--installed', '--json']:\n"
            "    if os.environ.get('PROBE_ERROR'): sys.exit(23)\n"
            "    if os.environ.get('BAD_METADATA'): print('[{}]'); sys.exit(0)\n"
            "    tool = args[3]\n"
            "    print(json.dumps([] if os.environ.get('MISSING') == tool else [dict(version=json.loads(os.environ['PINS'])[tool], install_path=str(root / tool), installed=True, active=True)]))\n"
            "elif args == ['exec', '--', 'rustc', '--print', 'sysroot']:\n"
            "    print(root / 'rust')\n"
            "elif args[:5] == ['exec', '--', 'cargo', 'build', '-p']:\n"
            "    assert args[5] == 'example-guest' or args[5] == 'umpire-temporal-guest', args\n"
            "    (root / 'cargo-called').touch()\n"
            "    if os.environ.get('FAIL_BUILD'):\n"
            "        print('compiler failed', file=sys.stderr); sys.exit(1)\n"
            "    print(json.dumps(dict(reason='compiler-artifact', target=dict(crate_types=['cdylib']), filenames=[str(root / 'built.wasm')])))\n"
            "else:\n"
            "    raise AssertionError(args)\n"
        )
        mise.chmod(0o755)
        self.env = dict(os.environ, PATH=f"{self.root}:{os.environ['PATH']}",
                        FIXTURE=str(self.root), PINS=json.dumps(self.pins),
                        CARGO_TARGET_DIR=str(self.root / "target"))
        self.env.pop("CI", None)

    def run_check(self, **extra):
        return subprocess.run(
            ["python3", str(SCRIPT), "example-guest", str(self.output)],
            cwd=REPO, env=dict(self.env, **extra), capture_output=True, text=True,
            check=False,
        )

    def test_ready_fresh_stale_missing_and_compile_failure(self):
        fresh = self.run_check()
        self.assertEqual(fresh.returncode, 0, fresh.stderr)
        self.output.write_bytes(b"\0asX\1\0\0\0")
        stale = self.run_check()
        self.assertEqual(stale.returncode, 2, stale.stderr)
        self.assertIn("existing=8 bytes, built=8 bytes", stale.stderr)
        self.assertIn("first differing offset=3", stale.stderr)
        self.assertEqual(self.output.read_bytes(), b"\0asX\1\0\0\0")
        self.output.unlink()
        missing = self.run_check()
        self.assertEqual(missing.returncode, 2, missing.stderr)
        self.assertIn("existing=missing", missing.stderr)
        self.assertFalse(self.output.exists())
        self.output.write_bytes(b"keep")
        for ci in [{}, {"CI": ""}]:
            with self.subTest(ci=ci):
                failed = self.run_check(FAIL_BUILD="1", **ci)
                self.assertEqual(failed.returncode, 1, failed.stderr)
                self.assertIn("compiler failed", failed.stderr)
                self.assertNotIn("skipped", failed.stderr)
                self.assertEqual(self.output.read_bytes(), b"keep")

    def test_missing_prerequisites_skip_only_with_ci_unset(self):
        paths = {"cargo": self.root / "rust/bin/cargo",
                 "rust-src": self.source, "wasm-target": self.target}
        for missing in ["mise", "rust", "rustup-toolchain", "go", "protoc", *paths]:
            with self.subTest(missing=missing):
                path = paths.get(missing)
                rustup = self.root / "rust/bin/rustup"
                if missing == "rustup-toolchain":
                    os.link(self.root / "rust/bin/rustc", rustup)
                if path:
                    path.rename(str(path) + ".saved")
                try:
                    extra = {"PATH": str(self.python_only)} if missing == "mise" else {}
                    local = self.run_check(MISSING=missing, **extra)
                    self.assertEqual(local.returncode, 0, local.stderr)
                    self.assertIn("warning", local.stderr)
                    self.assertIn("freshness skipped", local.stderr)
                    ci = self.run_check(MISSING=missing, CI="", **extra)
                    self.assertEqual(ci.returncode, 1, ci.stderr)
                    self.assertNotIn("skipped", ci.stderr)
                    self.assertFalse((self.root / "cargo-called").exists())
                    self.assertEqual(self.output.read_bytes(), self.artifact.read_bytes())
                finally:
                    if missing == "rustup-toolchain":
                        rustup.unlink()
                    if path:
                        Path(str(path) + ".saved").rename(path)

    def test_probe_error_is_not_missing_toolchain(self):
        for extra in [{"PROBE_ERROR": "1"}, {"BAD_METADATA": "1"}]:
            with self.subTest(extra=extra):
                result = self.run_check(**extra)
                self.assertEqual(result.returncode, 1, result.stderr)
                self.assertNotIn("skipped", result.stderr)
                self.assertFalse((self.root / "cargo-called").exists())

    def test_stale_guest_fails_real_model_check(self):
        env = dict(self.env)
        env.pop("UMPIRE_TIME_STEPS", None)
        result = subprocess.run(
            ["make", "--no-print-directory", "model-check",
             f"PROTOC_GEN_GO={self.root / 'go/bin/go'}", f"API_BINPB={self.root / 'go/bin/go'}"],
            cwd=REPO, env=env, capture_output=True, text=True, check=False,
        )
        self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn("stale guest", result.stderr)
        self.assertIn("built=8 bytes", result.stderr)
        self.assertIn("first differing offset=8", result.stderr)

    def test_missing_embedded_file_fails_go_compilation(self):
        go = subprocess.run(["mise", "which", "go"], cwd=REPO, check=True,
                            capture_output=True, text=True).stdout.strip()
        module = self.root / "isolated-host"
        module.mkdir()
        shutil.copyfile(REPO / "model/temporal/hosts/go/guest.go", module / "guest.go")
        (module / "go.mod").write_text("module embedded-guest-control\n\ngo 1.27.0\n")
        result = subprocess.run(
            [go, "test", "-tags", "test_dep", "./..."], cwd=module,
            env=dict(os.environ, GOTOOLCHAIN="local", CGO_ENABLED="0"),
            capture_output=True, text=True, check=False,
        )
        self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn("umpire_temporal_guest.wasm: no matching files found", result.stderr)


if __name__ == "__main__":
    unittest.main()
