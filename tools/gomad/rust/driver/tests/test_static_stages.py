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
                filename = {'cargo_clippy': 'cargo-clippy', 'clippy_driver': 'clippy-driver', 'cargo_fmt': 'cargo-fmt'}.get(name, name)
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
            with patch.object(sys, 'argv', ['static_stages.py', 'prepare-plan', '--inputs', str(root/'inputs.json'),
                                          '--plan', str(root/'prepared-plan.json')]):
                driver.write(root/'inputs.json', inputs)
                static.main()
            prepared = driver.read(root/'prepared-plan.json')
            self.assertFalse((root/'frozen.json').exists())
            self.assertFalse((root/'bindings.json').exists())
            self.assertEqual(set(prepared['stages']), set(static.LABELS))
            self.assertEqual(set(prepared['files']), {value['path'] for value in tools.values()})
            result = static.assemble(inputs, base, root/'plan.json', root/'bindings.json')
            self.assertEqual(prepared, driver.read(root/'plan.json'))
            self.assertEqual(sum(row['adapter'] == 'checked-command' for row in result['stages'].values()), 24)
            self.assertEqual(sum(row['adapter'] == 'pending' for row in result['stages'].values()), 72)
            self.assertTrue(all('UNEXECUTED' in row['reason'] for row in result['stages'].values() if row['adapter'] == 'pending'))
            Path(tools['go']['path']).write_text('changed')
            with self.assertRaisesRegex(driver.Refusal, 'changed tool'):
                static.assemble(inputs, base, root/'plan2.json', root/'bindings2.json')
            with self.assertRaisesRegex(driver.Refusal, 'changed tool'):
                static.prepare_plan(inputs, root/'changed-tool-plan.json')
            self.assertFalse((root/'changed-tool-plan.json').exists())
            for key in ('stages', 'argv', 'features'):
                altered = dict(inputs, **{key: []})
                with self.subTest(key=key), self.assertRaisesRegex(driver.Refusal, 'static input fields'):
                    static.prepare_plan(altered, root/'foreign-plan.json')
            self.assertFalse((root/'foreign-plan.json').exists())

    def test_original_scope_skip_is_source_bound_and_never_pass(self):
        package, name = static.GO_SCOPE_ROOT
        inventory = static.parse_go_inventory(listing((name,), package))
        witnesses = static.go_scope_exclusions(inventory)
        raw = events([dict(Action='run', Package=package, Test=name),
                      dict(Action='output', Package=package, Test=name, Output=static.GO_SCOPE_REASON),
                      dict(Action='skip', Package=package, Test=name), dict(Action='pass', Package=package)])
        self.assertEqual(static.verify_go(raw, inventory, 1, witnesses), [])
        self.assertEqual(witnesses[0]['status'], 'UNEXECUTED')
        with self.assertRaises(driver.Refusal):
            static.verify_go(raw, inventory, 1)
        with self.assertRaises(driver.Refusal):
            static.verify_go(raw.replace(static.GO_SCOPE_REASON.encode(), b'new reason'), inventory, 1, witnesses)
        with patch.object(static, 'sha', return_value='changed'), self.assertRaises(driver.Refusal):
            static.go_scope_exclusions(inventory)

    def test_conformance_only_runs_compatible_library_and_fmt_covers_workspace(self):
        plan = dict(repo=str(driver.REPO), lint_base='a'*40, tools={key: '/pinned/'+key for key in static.TOOLS})
        conformance = static.recipe(plan, 'rust-conformance-unit')
        self.assertEqual(len(conformance), 1)
        self.assertIn('--lib', conformance[0]['argv'])
        self.assertNotIn('--test', conformance[0]['argv'])

    def test_format_inventory_includes_full_workspace(self):
        plan = dict(repo=str(driver.REPO), lint_base='a'*40, tools={key: '/pinned/'+key for key in static.TOOLS})
        fmt = static.recipe(plan, 'rust-lint')[0]['argv']
        self.assertEqual(fmt, ['/pinned/cargo', 'fmt', '--manifest-path', str(driver.REPO / 'tools/gomad/rust/Cargo.toml'),
                               '--all', '--', '--check'])

    def reader_fixture(self, label, action):
        temporary = tempfile.TemporaryDirectory()
        self.addCleanup(temporary.cleanup)
        root = Path(temporary.name)
        config_path, plan_path = root/'config.json', root/'plan.json'
        config_path.write_text('{}')
        plan_path.write_text('{}')
        config = dict(root=str(root), source_sha256='1'*64, identity=dict(host='2'*64, helper='3'*64))
        plan = dict(driver_config=str(config_path), stages={label:[action]},
                    tools={key:'/pinned/'+key for key in static.TOOLS}, caches={key:'/cache/'+key for key in static.CACHES})
        evidence = root/'stages'/label/'static'
        evidence.mkdir(parents=True)
        commands = []
        def command(argv, raw, extra=None, cwd='/source'):
            name = f'command-{len(commands):04d}'
            environment = static.command_environment(plan, config, extra)
            receipt = dict(argv=argv, cwd=cwd, exit=0, observation='completed', pid=123, start_ns=1, end_ns=2,
                           environment=environment, temporary_paths={})
            driver.write(evidence/(name+'.command.json'), receipt)
            driver.write(evidence/(name+'.process.json'), dict(pid=123, active=False))
            (evidence/(name+'.stdout')).write_bytes(raw)
            commands.append(dict(name=name, argv=argv, cwd=cwd, environment=environment,
                                 command_sha256=driver.sha(evidence/(name+'.command.json')),
                                 process_sha256=driver.sha(evidence/(name+'.process.json')), stdout_sha256=driver.sha(evidence/(name+'.stdout'))))
            return name
        report = dict(schema='gomad.driver-static-execution/v1', stage=label, source_sha256=config['source_sha256'],
                      identity=config['identity'], plan_sha256=driver.sha(plan_path), config_sha256=driver.sha(config_path),
                      actions=[dict(recipe=action, results=[], commands=[])], commands=commands, deferred_ignored=[])
        def check():
            report['actions'][0]['commands'] = [row['name'] for row in commands]
            (evidence/'result.json').write_text(json.dumps(report))
            with patch.object(static, 'plan_config', return_value=(plan,config)):
                return static.authenticate_result(plan_path, label)
        return root, evidence, command, report, check

    def test_rust_repeats_are_distinct_and_doctest_argv_is_real(self):
        action = dict(kind='cargo-tests', argv=['/pinned/cargo','test','-p','gomad-wasmhost','--lib'], cwd='/source', repeats=2)
        root, evidence, command, report, check = self.reader_fixture('validate', action)
        binary = root/'native-test'
        binary.write_bytes(b'synthetic unit fixture, never executed')
        crate = driver.REPO / 'tools/gomad/rust/wasmhost'
        artifact = dict(reason='compiler-artifact', executable=str(binary), profile=dict(test=True),
                        manifest_path=str(crate / 'Cargo.toml'),
                        target=dict(name='test-target', src_path=str(crate / 'src/lib.rs')))
        command(action['argv']+['--no-run','--message-format=json'], events([artifact]))
        listed = command([str(binary),'--list'], b'a: test\n', cwd=str(crate))
        ignored = command([str(binary),'--ignored','--list'], b'0 tests\n', cwd=str(crate))
        executions = []
        for repeat in range(2):
            run = command([str(binary),'--test-threads=1'], b'test a ... ok\ntest result: ok. 1 passed; 0 failed; 0 ignored;\n', cwd=str(crate))
            executions.append(dict(binary=str(binary), binary_sha256=driver.sha(binary), target='test-target', cwd=str(crate), crate='wasmhost', source='wasmhost/src/lib.rs',
                                   listing=listed, ignored_listing=ignored, run=run, tests=['a'], ignored=[], repeat=repeat))
        doc = command(['/pinned/cargo','test','-p','gomad-wasmhost','--doc'], b'test result: ok. 0 passed; 0 failed; 0 ignored;\n')
        result = dict(kind='cargo-tests', executables=executions, passed=2, doctest=doc)
        report['actions'][0]['results'] = [result]
        self.assertEqual(len(check()['testcases']), 2)
        original_run = executions[1]['run']
        executions[1]['run'] = executions[0]['run']
        with self.assertRaisesRegex(driver.Refusal, 'reused execution'):
            check()
        executions[1]['run'] = original_run
        row = report['commands'][-1]
        row['argv'] = [str(binary),'--test-threads=1']
        path = evidence/(doc+'.command.json')
        receipt = driver.read(path)
        receipt['argv'] = row['argv']
        path.write_text(json.dumps(receipt))
        row['command_sha256'] = driver.sha(path)
        with self.assertRaisesRegex(driver.Refusal, 'recipe environment'):
            check()

    def test_replay_regenerations_need_independent_commands_and_environment(self):
        action = dict(kind='replay-regeneration', argv=['/pinned/go','test','-tags','test_dep','-count=1',
                      '-run','^TestReplayPlanConformance$','./internal/model/choice'], cwd='/source')
        root, evidence, command, report, check = self.reader_fixture('validate', action)
        results = []
        for index in range(2):
            extra = dict(GOMAD_REPLAY_PLAN_CORPUS_REGENERATE='1',
                         GOMAD_REPLAY_PLAN_CORPUS_DIR=str(evidence/('replay-generation-'+str(index))))
            listed = command([*action['argv'][:2], '-json', '-list', '^TestReplayPlanConformance$', *action['argv'][2:]],
                             listing(('TestReplayPlanConformance',)), extra)
            run = command([*action['argv'][:2], '-json', *action['argv'][2:]], execution(('TestReplayPlanConformance',)), extra)
            results.append(dict(kind='go-tests', listing=listed, run=run, repetitions=1, scope_exclusions=[],
                                roots=[dict(package='model', name='TestReplayPlanConformance', repetitions=1)]))
        report['actions'][0]['results'] = results
        with patch.object(static, 'tree_hashes', return_value={'original':'bytes'}):
            self.assertEqual(len(check()['testcases']), 2)
            original = results[1]['run']
            results[1]['run'] = results[0]['run']
            with self.assertRaisesRegex(driver.Refusal, 'reused execution'):
                check()
            results[1]['run'] = original
            row = report['commands'][-1]
            row['environment'].pop('GOMAD_REPLAY_PLAN_CORPUS_REGENERATE')
            path = evidence/(original+'.command.json')
            receipt = driver.read(path)
            receipt['environment'] = row['environment']
            path.write_text(json.dumps(receipt))
            row['command_sha256'] = driver.sha(path)
            with self.assertRaisesRegex(driver.Refusal, 'recipe environment'):
                check()

    def test_go_producer_lists_the_executed_root_selection(self):
        selected = 'TestReplayPlanConformance'
        cases = [([], '.', (selected, 'TestSibling'), 2),
                 (['-run', '^' + selected + '$'], '^' + selected + '$', (selected,), 1),
                 (['-run=^' + selected + '$'], '^' + selected + '$', (selected,), 1),
                 (['-run', ''], '.', (selected, 'TestSibling'), 1)]
        for flags, pattern, names, repeats in cases:
            with self.subTest(flags=flags), tempfile.TemporaryDirectory() as directory:
                action = dict(kind='go-tests', argv=['/pinned/go', 'test', '-tags', 'test_dep',
                              '-count=' + str(repeats), *flags, './model/...'], cwd='/source', repeats=repeats)
                producer = static.Execution({}, dict(root=directory), 'validate')
                calls = []
                def call(argv, cwd, env):
                    calls.append((argv, cwd, env))
                    producer.rows.append(dict(name='command-' + str(len(calls))))
                    if '-list' in argv:
                        actual_names = names if argv[argv.index('-list') + 1] == pattern else (selected, 'TestSibling')
                        return listing(actual_names)
                    return execution(names, repeats=repeats)
                extra = dict(GOMAD_ARCH_ALL_PLATFORMS='1')
                with patch.object(producer, 'call', side_effect=call):
                    result = producer.go(action, extra)
                self.assertEqual(calls, [([*action['argv'][:2], '-json', '-list', pattern, *action['argv'][2:]], '/source', extra),
                                         ([*action['argv'][:2], '-json', *action['argv'][2:]], '/source', extra)])
                self.assertEqual(result['roots'], [dict(package='model', name=name, repetitions=repeats) for name in sorted(names)])

    def test_go_reader_selection_and_raw_command_integrity(self):
        selected = 'TestReplayPlanConformance'
        for flags, pattern, names in [([], '.', (selected, 'TestSibling')),
                                      (['-run', '^' + selected + '$'], '^' + selected + '$', (selected,))]:
            with self.subTest(flags=flags):
                action = dict(kind='go-tests', argv=['/pinned/go', 'test', '-tags=test_dep', '-count=2',
                              *flags, './model/...'], cwd='/source', repeats=2, all_platforms=True)
                root, evidence, command, report, check = self.reader_fixture('architecture', action)
                extra = dict(GOMAD_ARCH_ALL_PLATFORMS='1')
                listed = command([*action['argv'][:2], '-json', '-list', pattern, *action['argv'][2:]], listing(names), extra)
                run = command([*action['argv'][:2], '-json', *action['argv'][2:]], execution(names, repeats=2), extra)
                report['actions'][0]['results'] = [dict(kind='go-tests', listing=listed, run=run, repetitions=2,
                    scope_exclusions=[], roots=[dict(package='model', name=name, repetitions=2) for name in sorted(names)])]
                with patch.object(static, 'ignored_inventory', return_value=[]):
                    self.assertEqual(len(check()['testcases']), len(names) * 2)
                    mutations = [(listed, listing(names + (names[0],)), 'duplicate listed Go root'),
                                 (run, execution(names, repeats=1), 'missing/duplicate Go root'),
                                 (run, execution(names, repeats=3), 'missing/duplicate Go root')]
                    for name, raw, message in mutations:
                        with self.subTest(mutation=message):
                            row = next(row for row in report['commands'] if row['name'] == name)
                            path = evidence/(name + '.stdout')
                            original = path.read_bytes()
                            path.write_bytes(raw)
                            row['stdout_sha256'] = driver.sha(path)
                            with self.assertRaisesRegex(driver.Refusal, message):
                                check()
                            path.write_bytes(original)
                            row['stdout_sha256'] = driver.sha(path)
                    for name in (listed, run):
                        row = next(row for row in report['commands'] if row['name'] == name)
                        path = evidence/(name + '.command.json')
                        original = driver.read(path)
                        forged = dict(original, argv=original['argv'] + ['-run=TestSibling'])
                        driver.write(path, forged)
                        row['argv'] = forged['argv']
                        row['command_sha256'] = driver.sha(path)
                        with self.assertRaisesRegex(driver.Refusal, 'changed command argv'):
                            check()
                        driver.write(path, original)
                        row['argv'] = original['argv']
                        row['command_sha256'] = driver.sha(path)

    def test_go_subtest_selection_refuses_before_any_command(self):
        for flags in (['-run', '^TestA$/child'], ['-run=^TestA$/child']):
            with self.subTest(flags=flags), tempfile.TemporaryDirectory() as directory:
                producer = static.Execution({}, dict(root=directory), 'validate')
                action = dict(argv=['/pinned/go', 'test', *flags, './model'], cwd='/source')
                with patch.object(producer, 'call', side_effect=AssertionError('unsupported selector started a command')) as call, self.assertRaisesRegex(driver.Refusal, 'subtest selector'):
                    producer.go(action)
                call.assert_not_called()

    def test_actual_bounded_command_receipt_binds_regeneration_environment(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            config = dict(repo=str(root), environment={'GOMAD_HELPER_CORPUS_REGENERATE':'1'},
                          temporary_paths={'TMPDIR':str(root)})
            raw = driver.command(config, root, 'probe', [sys.executable, '-c',
                                 'import os; print(os.environ["GOMAD_HELPER_CORPUS_REGENERATE"])'], timeout=5)
            self.assertEqual(raw, b'1\n')
            receipt = driver.read(root/'probe.command.json')
            self.assertEqual(receipt['environment'], config['environment'])
            self.assertEqual(receipt['temporary_paths'], config['temporary_paths'])

    def test_exact_known_ignored_inventory(self):
        rows = static.ignored_inventory(driver.REPO)
        self.assertEqual(len(rows), 30)
        root34 = [row for row in rows if row['name'].startswith(('final_network_', 'joined_storage_'))]
        self.assertEqual(len(root34), 2)
        self.assertTrue(all(row['final_id'] == 'transferred-live/TestClusterBrokerDurableRestartParity' for row in root34))
        self.assertEqual(sum(row['final_id'] is None for row in rows), 5)
        publication = [row for row in rows if row['name'] == 'terminal_publication_retained_replay']
        self.assertEqual(len(publication), 1)
        self.assertEqual(publication[0]['disposition'], 'supporting-only-nonqualification')
        self.assertIsNone(publication[0]['final_id'])
        self.assertNotIn('broad_routes', publication[0])
        counterfactual = [row for row in rows if row['source'] == 'gomad/src/campaign/counterfactual.rs']
        self.assertEqual({row['name'] for row in counterfactual}, {
            'prepare_the_four_original_counterfactual_requests',
            'produce_the_four_original_counterfactual_artifact_pairs'})
        self.assertTrue(all(row['final_id'] == 'test-wasm/complete-behavioral-reach'
                            and row['disposition'] == 'separate-actual-proof-required'
                            for row in counterfactual))
        publication = next(row for row in rows if row['name'] == 'terminal_controls_publication_child')
        self.assertEqual(publication['required_parent_test'],
                         'measurement::warm::terminal_controls::terminal_controls_actual_child_publication_uses_original_terminal_to_reap_bound')
        with tempfile.TemporaryDirectory() as directory:
            repo = Path(directory)
            (repo / 'tools/gomad/rust/gomad/src').mkdir(parents=True)
            with self.assertRaises(driver.Refusal):
                static.ignored_inventory(repo)

    def test_ignored_duplicate_bare_names_resolve_only_within_actual_cargo_owner(self):
        known = [dict(crate=crate, source=crate + '/tests/allocation_receiving.rs', name='allocation_receiving')
                 for crate in ('gomad', 'wasmhost')]
        for crate in ('gomad', 'wasmhost'):
            artifact = dict(manifest_path=str(driver.REPO / 'tools/gomad/rust' / crate / 'Cargo.toml'),
                            target=dict(src_path=str(driver.REPO / 'tools/gomad/rust' / crate / 'tests/allocation_receiving.rs')))
            owner = static.rust_test_owner(artifact)
            self.assertEqual(static.ignored_test_owner(known, 'allocation_receiving', owner)['crate'], crate)
            with self.assertRaisesRegex(driver.Refusal, 'within Cargo owner'):
                static.ignored_test_owner(known + [copy.deepcopy(known[0 if crate == 'gomad' else 1])], 'allocation_receiving', owner)
            with self.assertRaisesRegex(driver.Refusal, 'within Cargo owner'):
                static.ignored_test_owner(known, 'allocation_receiving', dict(crate=crate, source=crate + '/tests/foreign.rs'))
        with self.assertRaisesRegex(driver.Refusal, 'foreign crate'):
            static.rust_test_owner(dict(manifest_path='/foreign/Cargo.toml', target=dict(src_path='/foreign/src/lib.rs')))

    def test_ignored_inventory_excludes_rust_literal_fixtures_and_comments(self):
        text = '\n'.join([
            '// #[ignore = "comment"] fn fake() {}',
            '/* nested /* #[ignore = "nested"] */ fn fake() {} */',
            'let fixture = r##"#[ignore = "raw"] fn fake() {}"##;',
            'let normal = "#[ignore = \\"normal\\"] fn fake() {}";',
            "let quote = b'\"'; let lifetime: &'static str = \"literal\";",
            '#[test] #[ignore = "actual reason"] fn actual() {}',
        ])
        self.assertEqual(static.ignored_attributes(text), [('actual', 'actual reason')])
        with self.assertRaises(driver.Refusal): static.ignored_attributes('#[ignore] fn missing_reason() {}')
        with self.assertRaises(driver.Refusal): static.ignored_attributes('/* unclosed')

    def test_ignored_broad_routes_bind_fixed_final_selectors_and_features(self):
        original = static.read(static.HERE / 'static-ignored.json')
        read = static.read
        for field, value in (('selector', 'caller::selector'), ('features', ['caller-feature']), ('stage', 'ordinary')):
            forged = copy.deepcopy(original)
            row = next(row for row in forged['tests'] if 'broad_routes' in row)
            row['broad_routes'][0][field] = value
            def inputs(path):
                return forged if path == static.HERE / 'static-ignored.json' else read(path)
            with self.subTest(field=field), patch.object(static, 'read', side_effect=inputs), self.assertRaisesRegex(driver.Refusal, 'fixed executable'):
                static.ignored_inventory(driver.REPO)

    def test_changed_missing_duplicate_and_unknown_ignored_routes_refuse(self):
        original = static.read(static.HERE / 'static-ignored.json')
        read = static.read
        mutations = []
        missing = copy.deepcopy(original)
        missing['tests'].pop()
        mutations.append(missing)
        changed = copy.deepcopy(original)
        changed['tests'][-1]['reason'] += ' changed'
        mutations.append(changed)
        duplicate = copy.deepcopy(original)
        duplicate['tests'].append(copy.deepcopy(duplicate['tests'][-1]))
        mutations.append(duplicate)
        unknown = copy.deepcopy(original)
        unknown['tests'][-1]['name'] = 'unlisted_ignored_test'
        mutations.append(unknown)
        for declared in mutations:
            with self.subTest(declared=declared['tests'][-1]['name']):
                def inputs(path):
                    return declared if path == static.HERE / 'static-ignored.json' else read(path)
                with patch.object(static, 'read', side_effect=inputs), self.assertRaisesRegex(
                        driver.Refusal, 'unknown/changed/missing ignored'):
                    static.ignored_inventory(driver.REPO)

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
            plan = dict(driver_config=str(config_path), stages={'portable': [action]},
                        tools={key: '/pinned/'+key for key in static.TOOLS},
                        caches={key: '/cache/'+key for key in static.CACHES})
            evidence = root / 'stages/portable/static'
            evidence.mkdir(parents=True)
            commands = []
            environment = static.command_environment(plan, config)
            def command(name, argv, raw):
                driver.write(evidence / (name+'.command.json'), dict(argv=argv, cwd='/source', exit=0, observation='completed', pid=123, start_ns=1, end_ns=2,
                             environment=environment, temporary_paths={}))
                driver.write(evidence / (name+'.process.json'), dict(pid=123, active=False))
                (evidence / (name+'.stdout')).write_bytes(raw)
                commands.append(dict(name=name, argv=argv, cwd='/source', environment=environment, command_sha256=driver.sha(evidence/(name+'.command.json')),
                                     process_sha256=driver.sha(evidence/(name+'.process.json')), stdout_sha256=driver.sha(evidence/(name+'.stdout'))))
            command('command-0000', ['/pinned/go', 'test', '-json', '-list', '.', './model'], listing())
            command('command-0001', ['/pinned/go', 'test', '-json', './model'], execution())
            result = dict(kind='go-tests', scope_exclusions=[], listing='command-0000', run='command-0001', repetitions=1,
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
