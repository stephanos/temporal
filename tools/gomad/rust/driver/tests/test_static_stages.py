import copy
import json
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import driver
import static_stages as static


def events(rows):
    return ('\n'.join(json.dumps(row) for row in rows) + '\n').encode()


def listing(names=('TestA',), package='model'):
    return events([dict(Action='output', Package=package, Output=name+'\n') for name in names])


def execution(names=('TestA',), package='model', repeats=1):
    return events([dict(Action=action, Package=package, Test=name) for _ in range(repeats)
                   for name in names for action in ('run', 'pass')] + [dict(Action='pass', Package=package)])


class StaticStageContract(unittest.TestCase):
    def test_owned_labels_and_feature_matrix_match_original(self):
        self.assertEqual(len(static.LABELS), 24)
        self.assertEqual(static.LABELS[-2:], ('helper-corpus-regen-1', 'helper-corpus-regen-2'))
        plan = dict(repo=str(driver.REPO), lint_base='a'*40,
                    tools={key: '/pinned/'+key for key in static.TOOLS})
        recipes = {label: static.recipe(plan, label) for label in static.LABELS}
        clippy = [row for row in recipes['rust-lint'] if 'clippy' in row['argv']]
        self.assertEqual(len(clippy), 13)
        self.assertTrue(all(row['argv'][-2:] == ['-D', 'warnings'] for row in clippy))
        self.assertEqual(len(static.MODES), 6)
        self.assertIn('/pinned/true', recipes['compile-only'][0]['argv'])
        self.assertTrue(all('wasmhost/Cargo.toml' not in arg for rows in recipes.values() for row in rows for arg in row['argv']))
        self.assertEqual(recipes['validate'][9]['repeats'], 2)
        with self.assertRaises(driver.Refusal):
            static.recipe(plan, 'ordinary')

    def test_assembler_binds_only_owned_stages_and_refuses_changed_tools(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            tools = {}
            for name in static.TOOLS:
                filename = {'cargo_clippy': 'cargo-clippy', 'clippy_driver': 'clippy-driver'}.get(name, name)
                path = root / filename
                path.write_text('pinned tool fixture ' + name)
                tools[name] = dict(path=str(path), sha256=driver.sha(path))
            caches = {}
            for name in static.CACHES:
                path = root / name
                path.mkdir()
                caches[name] = str(path)
            inputs = dict(schema='gomad.driver-static-inputs/v1', repo=str(driver.REPO), driver_config=str(root/'frozen.json'),
                          tools=tools, caches=caches, lint_base='a'*40)
            base = dict(stages={label: {'adapter': 'pending'} for label in driver.mapping()}, inputs=[])
            result = static.assemble(inputs, base, root/'plan.json', root/'bindings.json')
            self.assertEqual(sum(row['adapter'] == 'checked-command' for row in result['stages'].values()), 24)
            self.assertEqual(sum(row['adapter'] == 'pending' for row in result['stages'].values()), 72)
            self.assertTrue(all('UNEXECUTED' in row['reason'] for row in result['stages'].values() if row['adapter'] == 'pending'))
            Path(tools['go']['path']).write_text('changed')
            with self.assertRaisesRegex(driver.Refusal, 'changed tool'):
                static.assemble(inputs, base, root/'plan2.json', root/'bindings2.json')

    def test_exact_known_ignored_inventory(self):
        rows = static.ignored_inventory(driver.REPO)
        self.assertEqual(len(rows), 14)
        root34 = [row for row in rows if row['name'].startswith(('final_network_', 'joined_storage_'))]
        self.assertEqual(len(root34), 2)
        self.assertTrue(all(row['final_id'] == 'transferred-live/TestClusterBrokerDurableRestartParity' for row in root34))
        self.assertEqual(sum(row['final_id'] is None for row in rows), 3)
        with tempfile.TemporaryDirectory() as directory:
            repo = Path(directory)
            (repo / 'tools/gomad/rust/gomad/src').mkdir(parents=True)
            with self.assertRaises(driver.Refusal):
                static.ignored_inventory(repo)

    def test_go_dynamic_roots_counts_subtest_skip_and_empty_refuse(self):
        inventory = static.parse_go_inventory(listing(('TestA', 'TestB')))
        result = static.verify_go(execution(('TestA', 'TestB'), repeats=2), inventory, 2)
        self.assertEqual(len(result), 2)
        for raw in (execution(('TestA',)), execution(('TestA', 'TestB'), repeats=3),
                    execution(('TestA', 'TestB')) + events([dict(Action='skip', Package='model', Test='TestA/child')])):
            with self.assertRaises(driver.Refusal):
                static.verify_go(raw, inventory, 2)
        with self.assertRaises(driver.Refusal):
            static.parse_go_inventory(events([dict(Action='output', Package='model', Output='no tests\n')]))

    def test_rust_dynamic_inventory_exact_ignored_and_summary(self):
        expected = static.test_names(b'a: test\nb: test\n2 tests, 0 benchmarks\n')
        raw = b'test a ... ok\ntest b ... ignored, original reason\ntest result: ok. 1 passed; 0 failed; 1 ignored; 0 measured; 0 filtered out\n'
        self.assertEqual(static.verify_rust(raw, expected, {'b'}), ['a'])
        for ignored in (set(), {'a'}):
            with self.assertRaises(driver.Refusal):
                static.verify_rust(raw, expected, ignored)
        with self.assertRaises(driver.Refusal):
            static.verify_rust(raw.replace(b'1 passed', b'2 passed'), expected, {'b'})
        with self.assertRaises(driver.Refusal):
            static.verify_rust(raw + b'test a ... ok\n', expected, {'b'})

    def test_reader_reparses_actual_raw_go_not_claimed_roots(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            config_path = root / 'config.json'
            config_path.write_text('{}')
            plan_path = root / 'plan.json'
            plan_path.write_text('{}')
            config = dict(root=str(root), source_sha256='1'*64, identity=dict(host='2'*64, helper='3'*64))
            action = dict(kind='go-tests', argv=['/pinned/go', 'test', './model'], cwd='/source')
            plan = dict(driver_config=str(config_path), stages={'portable': [action]})
            evidence = root / 'stages/portable/static'
            evidence.mkdir(parents=True)
            commands = []
            def command(name, argv, raw):
                driver.write(evidence / (name+'.command.json'), dict(argv=argv, cwd='/source', exit=0, observation='completed', pid=123, start_ns=1, end_ns=2))
                driver.write(evidence / (name+'.process.json'), dict(pid=123, active=False))
                (evidence / (name+'.stdout')).write_bytes(raw)
                commands.append(dict(name=name, argv=argv, cwd='/source', command_sha256=driver.sha(evidence/(name+'.command.json')),
                                     process_sha256=driver.sha(evidence/(name+'.process.json')), stdout_sha256=driver.sha(evidence/(name+'.stdout'))))
            command('command-0000', ['/pinned/go', 'test', '-json', '-list', '.', './model'], listing())
            command('command-0001', ['/pinned/go', 'test', '-json', './model'], execution())
            result = dict(kind='go-tests', listing='command-0000', run='command-0001', repetitions=1,
                          roots=[dict(package='model', name='TestA', repetitions=1)])
            report = dict(schema='gomad.driver-static-execution/v1', stage='portable', source_sha256=config['source_sha256'],
                          identity=config['identity'], plan_sha256=driver.sha(plan_path), config_sha256=driver.sha(config_path),
                          actions=[dict(recipe=action, results=[result], commands=[row['name'] for row in commands])],
                          commands=commands, deferred_ignored=[])
            def check():
                path = evidence/'result.json'
                path.write_text(json.dumps(report))
                with patch.object(static, 'plan_config', return_value=(plan, config)):
                    return static.authenticate_result(plan_path, 'portable')
            self.assertEqual(check()['testcases'][0]['id'], 'go/model/TestA')
            report['actions'][0]['results'][0]['roots'][0]['name'] = 'TestInvented'
            with self.assertRaisesRegex(driver.Refusal, 'roots differ'):
                check()
            report['actions'][0]['results'][0]['roots'][0]['name'] = 'TestA'
            report['commands'].pop()
            with self.assertRaises(driver.Refusal):
                check()


if __name__ == '__main__':
    unittest.main()
