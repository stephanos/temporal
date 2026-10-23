"""Original-source inventory tests; these do not execute any receiving gate."""
import sys
from pathlib import Path
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))


class OriginalReach(unittest.TestCase):
    def test_go_fixture_text_is_not_callable_or_subtest(self):
        import reach
        source = b'''package p
// func TestComment(t *testing.T) {}
func TestActual(t *testing.T) {
 fixture := `func TestFixture(t *testing.T) { t.Run("phantom", f) }`
 quoted := "} /* func TestQuoted(t *testing.T) {} */"
 rune := '}'
 /* func TestBlock(t *testing.T) {} */
 t.Run("real", func(t *testing.T) { helper(t) })
}
func helper(t *testing.T) { t.Run(name, f) }
'''
        functions = reach.functions(source)
        self.assertEqual([item['name'] for item in functions], ['TestActual', 'helper'])
        self.assertTrue(source[functions[0]['end_byte'] - 1:functions[0]['end_byte']] == b'}')
        self.assertEqual([item['expression'] for item in functions[0]['subtests']], ['"real"'])
        self.assertEqual(functions[1]['subtests'][0]['expression'], 'name')
        self.assertEqual(functions[0]['calls'], ['helper'])

    def test_nested_subtest_offsets_and_utf8_are_original_bytes(self):
        import reach
        source = 'package p\n// café\nfunc TestActual(t *testing.T) {\n t.Run(fmt.Sprintf("outer-%d", n), func(t *testing.T) { t.Run("child", f) })\n}\n'.encode()
        item = reach.functions(source)[0]
        self.assertEqual(len(item['subtests']), 2)
        self.assertEqual(item['subtests'][0]['expression'], 'fmt.Sprintf("outer-%d", n)')
        self.assertEqual(item['subtests'][1]['parent'], item['subtests'][0]['start_byte'])
        self.assertEqual(item['subtests'][1]['start_byte'], source.index(b't.Run("child"'))

    def test_unterminated_go_source_fails_closed(self):
        import reach
        for source in (b'func TestActual(t *testing.T) {', b'package p\n/* unfinished', b'package p\n`unfinished'):
            with self.subTest(source=source), self.assertRaises(ValueError): reach.functions(source)


if __name__ == '__main__':
    unittest.main()
