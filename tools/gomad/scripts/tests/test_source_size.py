"""Counter invariants cannot be observed through the public Gomad corpus."""
import importlib.util
from pathlib import Path
import unittest

spec = importlib.util.spec_from_file_location('source_size', Path(__file__).parents[1] / 'source-size.py')
size = importlib.util.module_from_spec(spec)
spec.loader.exec_module(size)


class SourceSize(unittest.TestCase):
    def test_comments_literals_and_nested_rust_comments(self):
        text = '\n// comment\nlet a = "// literal"; // tail\n/* outer\n/* inner */\nend */ let b = 2;\nlet r = r#"/* literal\n// still literal"#;\n'
        self.assertEqual(size.code_lines(text, 'Rust'), 4)
        self.assertEqual(size.code_lines('/* c */\nint x; /* tail */\n', 'C'), 1)
        self.assertEqual(size.code_lines('let x: &\'a str; // lifetime\n', 'Rust'), 1)
        self.assertEqual(size.code_lines('let r = r"\\"; /* actual comment\nend */\n', 'Rust'), 1)

    def test_go_raw_and_python_triple_literals(self):
        self.assertEqual(size.code_lines('var x = `\n// literal\n`\n// comment\n', 'Go'), 3)
        self.assertEqual(size.code_lines('x = """\n# literal\n"""\n# comment\n', 'Python'), 3)
        self.assertEqual(size.code_lines('echo "# literal" # tail\n# comment\n', 'Shell'), 1)
        self.assertEqual(size.code_lines("select '-- literal'; -- tail\n-- comment\n", 'SQL'), 1)
        self.assertEqual(size.code_lines('key = "# literal" # tail\n# comment\n', 'TOML'), 1)

    def test_generated_tests_templates_and_preexisting_helper(self):
        self.assertEqual(size.category('x_test.go', b'package x'), 'tests')
        self.assertEqual(size.category('tests/x.rs', b'// @generated\n'), 'generated')
        self.assertEqual(size.language('runtime.go.txt', b''), 'Go')
        self.assertEqual(size.language('wire.rs.tmpl', b''), 'Rust')
        self.assertEqual(size.subsystem('tools/gomad/internal/backends/wasm/wasmhost/src/lib.rs'), 'preexisting-rust-helper')
        self.assertEqual(size.subsystem('tools/gomad/rust/wasmhost/src/lib.rs'), 'linked-rust-engine')
        self.assertEqual(size.subsystem('tools/gomad/internal/model/world/world.go'), 'go-model-oracle-toolchain-and-support')


if __name__ == '__main__':
    unittest.main()
