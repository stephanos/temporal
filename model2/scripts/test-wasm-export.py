"""Regression controls for the WASI libc/free export-name collision."""

import importlib.util
import pathlib
import subprocess
import sys
import tempfile
import unittest

sys.dont_write_bytecode = True

spec = importlib.util.spec_from_file_location(
    "wasm_export", pathlib.Path(__file__).with_name("wasm-export.py")
)
wasm_export = importlib.util.module_from_spec(spec)
spec.loader.exec_module(wasm_export)


def section(kind, payload):
    return bytes([kind]) + wasm_export.encode_u32(len(payload)) + payload


def export(name, kind=0, index=3):
    encoded = name.encode()
    return wasm_export.encode_u32(len(encoded)) + encoded + bytes([kind, index])


class ExportNormalization(unittest.TestCase):
    def test_changes_only_one_export_name_and_preserves_code(self):
        prefix = b"\0asm\x01\0\0\0" + section(0, b"\x03tagpayload")
        code = section(10, b"\x01\x02\0\x0b")
        actual = prefix + section(7, b"\x02" + export("alloc") + export("umpire_free")) + code
        expected = prefix + section(7, b"\x02" + export("alloc") + export("free")) + code
        self.assertEqual(wasm_export.normalize(actual), expected)

    def test_refuses_ambiguous_missing_wrong_kind_and_malformed_exports(self):
        for entries in [
            [],
            [export("umpire_free"), export("umpire_free")],
            [export("umpire_free"), export("free")],
            [export("umpire_free", kind=2)],
        ]:
            with self.subTest(entries=entries), self.assertRaises(ValueError):
                wasm_export.normalize(b"\0asm\x01\0\0\0" + section(7, bytes([len(entries)]) + b"".join(entries)))
        for data in [b"bad", b"\0asm\x01\0\0\0\x07\xff", b"\0asm\x01\0\0\0" + section(7, b"\x01\x7fshort"),
                     b"\0asm\x01\0\0\0" + section(7, b"\x01" + export("umpire_free")) * 2]:
            with self.subTest(data=data), self.assertRaises(ValueError):
                wasm_export.normalize(data)

    def test_linker_keeps_other_artifacts_and_fails_closed_for_guest(self):
        with tempfile.TemporaryDirectory() as scratch:
            other = pathlib.Path(scratch) / "other.wasm"
            guest = pathlib.Path(scratch) / "umpire_wasm.wasm"
            for output, expected in [(other, 0), (guest, 1)]:
                output.write_bytes(b"not an oracle module")
                result = subprocess.run(
                    [sys.executable, str(pathlib.Path(__file__).with_name("wasm-export.py")),
                     "-o", str(output)], capture_output=True, check=False
                )
                self.assertEqual(result.returncode, expected, result.stderr)
                self.assertEqual(output.read_bytes(), b"not an oracle module")


if __name__ == "__main__":
    unittest.main()
