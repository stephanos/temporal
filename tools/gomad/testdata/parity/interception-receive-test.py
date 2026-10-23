#!/usr/bin/env python3
"""Offline causality tests; none count as a real Go receiving proof."""
import copy
import json
import os
from pathlib import Path
import sys
import tempfile
import time
import unittest
from unittest.mock import patch

sys.dont_write_bytecode = True
from importlib.machinery import SourceFileLoader
from importlib.util import module_from_spec, spec_from_loader

HERE = Path(__file__).resolve().parent
loader = SourceFileLoader('receiving_tests', str(HERE / 'interception-receive.py'))
check = module_from_spec(spec_from_loader(loader.name, loader))
loader.exec_module(check)
hidden = check.shim.load('receiving_hidden_tests', HERE / 'interception-receive-hidden.py')


class SourceContract(unittest.TestCase):
    def test_exact_original_81_definitions_and_21_real_names_unchanged(self):
        corpus = check.read(HERE / 'interception-shim-corpus.json')['cases']
        self.assertEqual(len(corpus), 81)
        self.assertEqual([row['name'] for row in corpus if row.get('requires_real_inputs')], [
            'guest-test', 'caller-count', 'boundary-args', 'boundary-double-args', 'boundary-terminator',
            'guest-lookalikes', 'goflags-quoted', 'repeated-effective-flags', 'empty-effective-exec',
            'external-launcher', 'exec-true-count0', 'multi-package-json', 'guest-run', 'guest-build',
            'explicit-excluded-tag', 'auto-prime', 'auto-reuse', 'auto-schema-mutation',
            'auto-schema-missing', 'auto-changed-executable', 'concurrent-identities'])
        for row in corpus:
            if row['name'].startswith('auto-'):
                with self.assertRaisesRegex(ValueError, 'compile-only'):
                    check.shim.validate_real(row, dict(limits=check.LIMITS, argv=['test'],
                        accepted_selections=[dict(package='x', selector='x', receipt=__file__,
                                                 receipt_sha256=check.sha(__file__))],
                        campaigns=0, status=2 if row['name'] == 'auto-schema-missing' else 0))

    def test_new_auto_route_runs_real_selected_test_without_install_or_exec(self):
        inputs = dict(binary='/binary', fixture_binary='/fixture', installation='/install', checkout='/source')
        argv = check.command(inputs, 'auto-first', Path('/evidence/auto-first'))
        self.assertNotIn('--installation', argv)
        self.assertNotIn('-exec=true', argv)
        self.assertIn('-count=1', argv)
        self.assertIn('-run=^TestFrontendSystemInfo$', argv)
        self.assertEqual(argv[-1], check.FRONTEND)
        self.assertEqual(check.command(inputs, 'auto-reuse', Path('/evidence/auto-reuse'))[7:], argv[7:])


class Output(unittest.TestCase):
    def test_guest_bytes_plus_go_summary_only(self):
        stdout = b'=== RUN   TestFrontendSystemInfo\n--- PASS: TestFrontendSystemInfo (0.00s)\nPASS\n'
        summary = b'ok  \tgo.temporal.io/server/tests/gomadfunctional\t1.0s\n'
        packages = [('go.temporal.io/server/tests/gomadfunctional', 'TestFrontendSystemInfo', stdout, b'')]
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / 'stdout').write_bytes(stdout + summary)
            (root / 'stderr').write_bytes(b'')
            check.authenticate_output(root, packages)
            for bad in (b'compiler noise\n' + stdout + summary, stdout, stdout + summary + summary):
                with self.subTest(raw=bad):
                    (root / 'stdout').write_bytes(bad)
                    with self.assertRaises(ValueError):
                        check.authenticate_output(root, packages)

    def test_named_pass_cannot_replace_another_guest_output(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / 'stdout').write_bytes(b'ok  \tpkg\t1s\n')
            (root / 'stderr').write_bytes(b'')
            with self.assertRaisesRegex(ValueError, 'exact_named_guest_stdout'):
                check.authenticate_output(root, [('pkg', 'TestFrontendSystemInfo', b'PASS\n', b'')])


class Overlap(unittest.TestCase):
    def fixture(self):
        origins = [dict(launch=dict(process_id=pid, argv=check.encoded(['/fixture', 'run-guest', '/module' + str(pid)])))
                   for pid in (10, 11)]
        launches = [dict(pid=row['launch']['process_id'], argv=row['launch']['argv']) for row in origins]
        return origins, [dict(monotonic_ns=1, go_pid=9, launchers=launches)]

    def test_two_actual_children_with_distinct_argv_overlap(self):
        origins, samples = self.fixture()
        check.authenticate_overlap(samples, origins, 9)

    def test_sequential_launches_do_not_prove_overlap(self):
        origins, samples = self.fixture()
        launches = samples[0]['launchers']
        samples = [dict(monotonic_ns=i + 1, go_pid=9, launchers=[value]) for i, value in enumerate(launches)]
        with self.assertRaisesRegex(ValueError, 'no observed concurrent'):
            check.authenticate_overlap(samples, origins, 9)

    def test_wrong_pid_argv_and_duplicate_are_refused(self):
        origins, original = self.fixture()
        for field, value in [('go_pid', 12), ('pid', 12), ('argv', ['ff'])]:
            samples = copy.deepcopy(original)
            if field == 'go_pid':
                samples[0][field] = value
            else:
                samples[0]['launchers'][0][field] = value
            with self.subTest(field=field), self.assertRaises(ValueError):
                check.authenticate_overlap(samples, origins, 9)
        samples = copy.deepcopy(original)
        samples[0]['launchers'][1] = samples[0]['launchers'][0]
        with self.assertRaisesRegex(ValueError, 'duplicate'):
            check.authenticate_overlap(samples, origins, 9)

    def test_observer_scans_every_thread_and_preserves_non_utf8_argv(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            for thread, child in [('1', '10'), ('2', '11')]:
                path = root / '9/task' / thread
                path.mkdir(parents=True)
                (path / 'children').write_text(child)
                path = root / child
                path.mkdir()
                (path / 'cmdline').write_bytes(b'/fixture\0run-guest\0/\xff\0')
            actual = check.Path
            def fake_path(*parts):
                if parts[0] == '/proc':
                    return root.joinpath(*parts[1:])
                return actual(*parts)
            with patch.object(check, 'Path', side_effect=fake_path):
                result = check.observe(9)
            self.assertEqual([row['pid'] for row in result['launchers']], [10, 11])
            self.assertEqual(result['launchers'][0]['argv'][-1], '2fff')


class WrapperChild(unittest.TestCase):
    def test_only_actual_observed_go_driver_can_parent_launcher(self):
        inputs = dict(compiler='/compiler', files={'/compiler': 'a' * 64})
        argv = check.encoded(['/compiler', 'test', './selected'])
        origins = [dict(invocation=dict(driver_argv=argv), launch=dict(parent_process_id=10))]
        samples = [dict(go_pid=9, drivers=[dict(pid=10, argv=argv,
                       executable=os.fsencode('/compiler').hex(), executable_sha256='a' * 64)])]
        check.authenticate_wrapper(samples, origins, 9, inputs)
        for parent in (None, 9, 11):
            bad = copy.deepcopy(origins)
            bad[0]['launch']['parent_process_id'] = parent
            with self.subTest(parent=parent), self.assertRaises(ValueError):
                check.authenticate_wrapper(samples, bad, 9, inputs)
        for key, value in [('argv', ['ff']), ('executable_sha256', 'b' * 64)]:
            bad = copy.deepcopy(samples)
            bad[0]['drivers'][0][key] = value
            with self.subTest(key=key), self.assertRaises(ValueError):
                check.authenticate_wrapper(bad, origins, 9, inputs)


class Cache(unittest.TestCase):
    def test_first_publication_and_exact_reuse(self):
        state = {'manifest': dict(sha256='a' * 64, mtime_ns=12)}
        check.authenticate_cache_pair({}, state)
        check.authenticate_cache_pair(state, state, state)
        for before, after, baseline in [(state, state, None), ({}, {}, None),
                                         (state, {'manifest': dict(sha256='a' * 64, mtime_ns=13)}, state),
                                         ({}, state, state)]:
            with self.subTest(before=before, after=after), self.assertRaises(ValueError):
                check.authenticate_cache_pair(before, after, baseline)


class CommandCustody(unittest.TestCase):
    def test_actual_small_python_command_and_raw_tampering(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory) / 'command'
            inputs = dict(checkout=directory, compiler=sys.executable, cache=directory, environment={})
            deadline = time.monotonic_ns() + 5_000_000_000
            argv = [sys.executable, '-c', "import sys;sys.stdout.buffer.write(b'raw\\xff')"]
            self.assertEqual(check.run(argv, inputs, root, deadline), 0)
            self.assertGreater(check.authenticate_command(root, argv, inputs, deadline), 0)
            (root / 'stdout').write_bytes(b'normalized')
            with self.assertRaisesRegex(ValueError, 'raw_stream_digest'):
                check.authenticate_command(root, argv, inputs, deadline)

    def test_expired_admission_never_spawns(self):
        with tempfile.TemporaryDirectory() as directory:
            inputs = dict(checkout=directory, compiler=sys.executable, cache=directory, environment={})
            with patch.object(check.subprocess, 'Popen') as spawn:
                with self.assertRaisesRegex(ValueError, 'before spawn'):
                    check.run(['/no-command'], inputs, Path(directory) / 'command', 1)
                spawn.assert_not_called()


class InspectBinding(unittest.TestCase):
    def test_exit_zero_without_exact_record_hash_and_path_does_not_pass(self):
        record = dict(schema='actual-record', streams={'target/intercepted-origin.json': dict(sha256='a' * 64)})
        inspected = dict(schema='gomad.inspect/v9', kind='artifact', path='/artifact',
                         artifact=dict(record=record, record_sha256='b' * 64))
        with patch.object(check.check.campaign, 'authenticate_artifact', return_value=record), \
             patch.object(check, 'sha', return_value='b' * 64):
            check.authenticate_inspect(json.dumps(inspected).encode(), Path('/artifact'))
            for field in ('path', 'record', 'record_sha256'):
                bad = copy.deepcopy(inspected)
                if field == 'path':
                    bad[field] = '/other'
                else:
                    bad['artifact'][field] = {} if field == 'record' else 'c' * 64
                with self.subTest(field=field), self.assertRaises(ValueError):
                    check.authenticate_inspect(json.dumps(bad).encode(), Path('/artifact'))


class HiddenInputs(unittest.TestCase):
    def test_retained_only_commands_have_no_installation_selector(self):
        self.assertEqual(hidden.commands(dict(fixture_binary='/fixture'), ['/a', '/b']),
                         [['/fixture', 'replay', '/a'], ['/fixture', 'replay', '/b']])

    def test_namespace_requires_all_original_source_and_compiler_paths_masked(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            source, compiler = root / 'source', root / 'compiler'
            source.mkdir(); compiler.mkdir()
            installation = root / 'install.json'
            installation.write_text(json.dumps(dict(cache_root=str(source), runtime_root=str(source),
                overlay_module_root=str(source), compiler_path=str(compiler / 'go'),
                admission=dict(compiler_inputs={str(compiler / 'go'): 'a'}))))
            inputs = dict(hidden_directories=[str(source), str(compiler)], hidden_installations=[str(installation)],
                          installation=str(installation), checkout=str(source), cache=str(source), compiler=str(compiler / 'go'),
                          files={str(installation): check.sha(installation)},
                          binary='/bin/ordinary', fixture_binary='/bin/fixture', bwrap='/bin/bwrap', python='/bin/python')
            provenance = dict(inputs=dict(compiler_path=str(compiler / 'go'), compiler={str(compiler / 'go'): 'a'}, sources={str(source / 'file'): 'b'}))
            with patch.object(hidden.base.check.campaign, 'canonical_file', return_value=provenance):
                argv = hidden.namespace_argv(inputs, root / 'spec', root / 'proof', ['/evidence/artifact'])
                self.assertIn('--unshare-net', argv)
                self.assertEqual(argv.count('--tmpfs'), 2)
                bad = copy.deepcopy(inputs)
                bad['hidden_directories'] = [str(source)]
                with self.assertRaisesRegex(ValueError, 'cover original compiler'):
                    hidden.namespace_argv(bad, root / 'spec', root / 'proof', ['/evidence/artifact'])
                changed = copy.deepcopy(provenance)
                changed['inputs']['compiler_path'] = '/unhidden/bin/go'
                with patch.object(hidden.base.check.campaign, 'canonical_file', return_value=changed):
                    with self.assertRaisesRegex(ValueError, 'compiler executable'):
                        hidden.namespace_argv(inputs, root / 'spec', root / 'proof', ['/evidence/artifact'])
                bad = copy.deepcopy(inputs)
                bad['fixture_binary'] = str(source / 'fixture')
                with self.assertRaisesRegex(ValueError, 'cover replay executable'):
                    hidden.namespace_argv(bad, root / 'spec', root / 'proof', ['/evidence/artifact'])

    def test_standalone_inner_program_has_no_source_checkout_dependency(self):
        compile(hidden.INNER_SOURCE, '<retained-hidden-child>', 'exec')
        self.assertNotIn('receiving_hidden_base', hidden.INNER_SOURCE)
        self.assertNotIn('import base', hidden.INNER_SOURCE)
        self.assertIn('spec[\'deadline_ns\'] - time.monotonic_ns()', hidden.INNER_SOURCE)


if __name__ == '__main__':
    unittest.main()
