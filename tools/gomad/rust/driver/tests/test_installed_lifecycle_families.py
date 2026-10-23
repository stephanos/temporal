"""Fixed installed lifecycle family inputs; no gomad or Go command is executed."""
from pathlib import Path
import sys
import tempfile
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import installed_lifecycle_families as families
from driver import Refusal, write


class InstalledLifecycleFamilies(unittest.TestCase):
    def test_private_sources_are_exactly_what_the_unchanged_checker_admits(self):
        # typed-detached and kill-24 use the typed operator fixture, kill-12 the kill fixture.
        for family, checked in (('typed-detached', 'prompt'), ('kill-24', 'kill-24'), ('kill-12', 'kill-12')):
            with self.subTest(family=family), tempfile.TemporaryDirectory() as temporary:
                source = Path(temporary) / 'source'
                source.mkdir()
                for name, raw in families.source_files(family).items():
                    (source / name).write_bytes(raw)
                families.LIFECYCLE.source_inputs(dict(source=str(source)), checked)
                (source / 'extra.go').write_bytes(b'package main\n')
                with self.assertRaises(Refusal):
                    families.LIFECYCLE.source_inputs(dict(source=str(source)), checked)

    def test_families_bind_the_three_original_typed_lifecycle_roots(self):
        self.assertEqual(families.FAMILIES, {
            'typed-detached': 'TestInstalledOperatorTypedScenarioDetachedReplay',
            'kill-12': 'TestInstalledOperatorKillResume',
            'kill-24': 'TestInstalledOperatorTypedScenarioKillResume'})

    def test_result_refuses_unknown_family_or_changed_evidence(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            write(root / 'recipe.json', dict(family='kill-12', main_build='/build.json', repo=str(families.MODULE.parents[1])))
            for result in (dict(schema='other', family='kill-12', root='TestInstalledOperatorKillResume'),
                           dict(schema=families.SCHEMA, family='kill-24', root='TestInstalledOperatorTypedScenarioKillResume'),
                           dict(schema=families.SCHEMA, family='kill-12', root='TestInstalledOperatorTypedScenarioKillResume')):
                (root / 'result.json').unlink(missing_ok=True)
                write(root / 'result.json', result)
                with self.subTest(result=result), self.assertRaises(Refusal):
                    families.read_result(root / 'result.json', None, 'kill-12')


if __name__ == '__main__':
    unittest.main()
