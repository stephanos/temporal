"""Controls for the generic guest builder's command and exit contract."""

import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest


SCRIPT = Path(__file__).resolve().parents[1] / "guest.sh"


class GuestBuild(unittest.TestCase):
    def test_build_fresh_stale_and_failure(self):
        with tempfile.TemporaryDirectory(prefix="guest controls ") as directory:
            root = Path(directory)
            artifact = root / "custom_library.wasm"
            artifact.write_bytes(b"\0asm\1\0\0\0")
            mise = root / "mise"
            mise.write_text(
                "#!/usr/bin/env python3\n"
                "import json, os, subprocess, sys\n"
                "from pathlib import Path\n"
                "args = sys.argv[1:]\n"
                "assert args[:2] == ['exec', '--'], args\n"
                "if args[2:] == ['rustc', '--print', 'sysroot']:\n"
                "    print('/pinned toolchain/sysroot')\n"
                "elif args[2:] == ['rustc', '-V']:\n"
                "    print('rustc 1.99.1 (58cf8f9c3 2026-10-13)')\n"
                "elif args[2:5] == ['cargo', 'metadata', '--format-version']:\n"
                "    assert '--locked' in args, args\n"
                "    print(json.dumps({'packages': [], 'resolve': {'nodes': []}}))\n"
                "else:\n"
                "    assert args[2:5] == ['cargo', 'build', '-p'], args\n"
                "    assert args[5] == 'example-guest', args\n"
                "    for flag in ['--locked', '--lib']:\n"
                "        assert flag in args, args\n"
                "    assert args[args.index('--profile') + 1] == 'guest', args\n"
                "    assert args[args.index('--target') + 1] == 'wasm32-wasip1', args\n"
                "    assert '-Zbuild-std=std,panic_unwind' in args, args\n"
                "    assert os.environ['RUSTC_BOOTSTRAP'] == '1'\n"
                "    assert Path(os.environ['RUSTC_WRAPPER']).is_file()\n"
                "    if sys.platform == 'darwin':\n"
                "        probe = Path(os.environ['ARTIFACT']).with_name('compiler')\n"
                "        probe.write_text('#!' + sys.executable + '\\nimport os\\nassert os.environ.get(\"DYLD_FALLBACK_LIBRARY_PATH\") == \"/guest-test-marker\"\\n')\n"
                "        probe.chmod(0o755)\n"
                "        subprocess.run([os.environ['RUSTC_WRAPPER'], str(probe), '-V'], env=dict(os.environ, DYLD_FALLBACK_LIBRARY_PATH='/guest-test-marker'), check=True)\n"
                "    assert json.loads(Path(os.environ['UMPIRE_GUEST_LOCK']).read_text()) == {}\n"
                "    assert os.environ['UMPIRE_GUEST_COMPILER'].startswith('rustc 1.99.1')\n"
                "    flags = os.environ['CARGO_ENCODED_RUSTFLAGS'].split(chr(31))\n"
                "    for flag in ['-Cpanic=unwind', '-Ctarget-feature=+exception-handling', '-Cllvm-args=-wasm-use-legacy-eh=false']:\n"
                "        assert flag in flags, flags\n"
                "    assert any(f.startswith('--remap-path-prefix=') and '/sysroot=' in f for f in flags), flags\n"
                "    assert any(f.startswith('--remap-path-prefix=') and '/registry=' in f for f in flags), flags\n"
                "    assert any(f.startswith('--remap-path-prefix=') and f.endswith('=/workspace') for f in flags), flags\n"
                "    assert '--remap-path-prefix=' + os.environ['CARGO_TARGET_DIR'] + '=/target' in flags, flags\n"
                "    sys.exit(37) if os.environ.get('FAIL_BUILD') else None\n"
                "    print(json.dumps({'reason': 'compiler-artifact', 'target': {'crate_types': ['cdylib']}, 'filenames': [os.environ['ARTIFACT']]}))\n"
            )
            mise.chmod(0o755)
            env = dict(os.environ, PATH=f"{root}:{os.environ['PATH']}",
                       ARTIFACT=str(artifact), CARGO_HOME=str(root / "cargo"),
                       CARGO_TARGET_DIR=str(root / "target"))
            output = root / "output.wasm"

            def run(*args, **extra):
                return subprocess.run(
                    ["bash", str(SCRIPT), "example-guest", str(output), *args],
                    env=dict(env, **extra), capture_output=True, text=True, check=False,
                )

            built = run()
            self.assertEqual(built.returncode, 0, built.stderr)
            self.assertEqual(output.read_bytes(), artifact.read_bytes())
            fresh = run("--check")
            self.assertEqual(fresh.returncode, 0, fresh.stderr)
            for stale, offset in [(b"\0asX\1\0\0\0", 3), (b"\0asm", 4), (b"\0asm\1\0\0\0X", 8)]:
                with self.subTest(stale=stale):
                    output.write_bytes(stale)
                    result = run("--check")
                    self.assertEqual(result.returncode, 2, result.stderr)
                    self.assertIn(f"existing={len(stale)} bytes", result.stderr)
                    self.assertIn("built=8 bytes", result.stderr)
                    self.assertIn(f"first differing offset={offset}", result.stderr)
                    self.assertEqual(output.read_bytes(), stale)
            output.unlink()
            missing = run("--check")
            self.assertEqual(missing.returncode, 2, missing.stderr)
            self.assertIn("missing", missing.stderr)
            self.assertFalse(output.exists())
            output.write_bytes(b"keep")
            failed = run(FAIL_BUILD="1")
            self.assertEqual(failed.returncode, 1, failed.stderr)
            self.assertEqual(output.read_bytes(), b"keep")
            if sys.platform == "darwin" and Path("/usr/bin/python3").exists():
                (root / "python3").symlink_to("/usr/bin/python3")
                system_python = run()
                self.assertEqual(system_python.returncode, 0, system_python.stderr)


if __name__ == "__main__":
    unittest.main()
