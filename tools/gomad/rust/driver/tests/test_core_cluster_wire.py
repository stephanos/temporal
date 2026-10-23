"""Synthetic adversarial receiving controls; never original Go or native run proof."""
import copy
import io
import json
from pathlib import Path
import struct
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import broad_stages
import cluster_wire_reference as reference
import core_cluster_wire as gate
import core_codec
from driver import Refusal


def events():
    package = reference.terminal_controls.PACKAGE
    rows = [dict(Package=package, Action='start')]
    for name in (*reference.ROOTS, *reference.CASES):
        rows.extend(dict(Package=package, Test=name, Action=action) for action in ('run', 'pass'))
    rows.append(dict(Package=package, Action='pass'))
    return rows


def encoded(rows):
    return b'\n'.join(json.dumps(row).encode() for row in rows)


def stale_restart():
    def number(value): return struct.pack('<Q', value)
    def blob(value): return number(len(value)) + value
    volumes = [dict(ID='data', Path='/workspace', CapacityBytes=4096)]
    model = dict(volumes=volumes, limits=dict(PendingOperations=100, Transitions=1000),
                 faults=None, sqlite_lock_databases=None)
    identity = 'sha256:' + core_codec.digest(core_codec.struct_json(dict(volumes=volumes, limits=model['limits'])))
    obj = number(3) + number(0o700) + number(2) + number(0) + blob(b'')
    volume = blob(b'data') + blob(b'/workspace') + number(4096) + number(1) + obj + number(1) + blob(b'/') + number(3)
    volume += number(1) + number(3) + number(0) + number(0) + number(1) + number(0)
    sections = [blob(identity.encode()), number(0), number(4) + number(1) + number(1) + volume]
    image = b'GMVOLIMG' + struct.pack('<II', 1, 3) + b''.join(
        struct.pack('<I', index) + blob(section) for index, section in enumerate(sections, 1))
    before = dict(descriptor=dict(model=model, model_identity=identity, image=None), image=None, context=None)
    retry = copy.deepcopy(before)
    retry['image'] = list(image)
    retry['descriptor']['image'] = dict(bytes=len(image), chunks=1, sha256=core_codec.digest(image))
    return dict(refusal='WASI invalid: broker restart lifecycle', before_incarnation=1, before_slot=1,
        after_refusal_incarnation=1, after_refusal_slot=1, peer_incarnation=1, peer_slot=2,
        refused_storage_unchanged=True, retry_incarnation=2, retry_slot=3, recovery_input_unchanged=True,
        storage_before=before, storage_after_refusal=copy.deepcopy(before), retry_storage=retry,
        recovery_image=list(image), activity=dict(launched_stores=0, remote_callback_attempts=0))


def guards():
    errors = {'missing-namespace': 'cluster requires sealed namespace', 'unknown-step': 'unknown cluster lifecycle handle',
        'zero-incarnation': 'invalid or duplicate cluster handle', 'duplicate-handle': 'invalid or duplicate cluster handle',
        'model-seed': 'cluster model seed mismatch'}
    rows = [dict(schema=gate.lifecycle.SCHEMA, original_root=gate.lifecycle.ATTRIBUTION,
                 variant=name, observed=dict(admission_error=error)) for name, error in errors.items()]
    rows.extend(dict(schema=gate.lifecycle.SCHEMA, original_root=gate.lifecycle.CALLBACK, variant=name,
        observed=dict(error_kind='unsupported', message='linked cluster forbids transport callbacks',
                      activity=dict(launched_stores=0, remote_callback_attempts=1))) for name in ('call', 'sync'))
    rows.append(dict(schema=gate.lifecycle.SCHEMA, original_root=gate.lifecycle.RESTART,
                     variant='stale-incarnation', observed=stale_restart()))
    return rows


def contract(root):
    return dict(schema=gate.SCHEMA, original_root=root, original_pass=False,
        historical_reference=dict(schema=reference.SCHEMA, testcases=list((*reference.ROOTS, *reference.CASES))),
        retired_assertions=[dict(id=name, applicability='retired-framed-helper-boundary', historical_original_test=root)
                            for name in gate.ASSERTIONS[root]], native_guards=guards(),
        boundary='typed-native-ownership; no framed JSON peer or tag codec',
        source_bindings=dict.fromkeys(gate.SOURCES, 'synthetic'))


class ClusterWire(unittest.TestCase):
    def test_cli_binds_the_exact_config_path_and_hash_before_execution(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / 'config.json'
            path.write_bytes(b'{"synthetic":"CLI binding control"}')
            def collect(config, produce):
                self.assertEqual(config['_config_path'], str(path.resolve()))
                self.assertEqual(config['_config_sha256'], reference.sha(path))
                self.assertTrue(produce)
                return dict(synthetic='not execution proof')
            with patch.object(sys, 'argv', ['cluster_wire_reference.py', 'produce', '--config', str(path)]), \
                    patch.object(reference, 'configured', return_value={}), \
                    patch.object(reference, 'collect', side_effect=collect), \
                    patch.object(sys, 'stdout', io.StringIO()), patch.object(sys, 'stderr', io.StringIO()):
                self.assertEqual(reference.main(), 0)

    def test_input_closure_adds_genuine_original_source_archive_and_complete_compiler(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            derived, goroot = root / 'derived', root / 'go'
            files = [derived / name for name in ('source.json', 'original.tar', 'source/original.go', 'source/glue.go')]
            files += [root / 'git', goroot / 'bin/go', goroot / 'pkg/tool/compiler']
            for path in files:
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_bytes(b'synthetic input closure control')
            source = dict(original_files={'original.go': dict(kind='file'), 'original-link': dict(kind='symlink')},
                          glue={'glue.go': 'synthetic'})
            entry = dict(derived=str(derived), go_root=str(goroot), go=str(goroot / 'bin/go'), git=str(root / 'git'))
            with patch('reference_profile.verify', return_value=source) as verify:
                paths = reference.input_paths(entry)
                verify.assert_called_once_with(derived)
                self.assertEqual(paths, {str(path) for path in files} | {str(reference.Path(reference.__file__).parent / 'reference/main.go.txt')})
                with self.assertRaises(Refusal):
                    reference.input_paths(dict(entry, go=str(root / 'foreign-go')))
                (derived / 'source/original.go').unlink()
                with self.assertRaises(Refusal):
                    reference.input_paths(entry)

    def test_exact_six_roots_and_twelve_mutation_events(self):
        self.assertEqual(len(reference.ROOTS), 6)
        self.assertEqual(len(reference.CASES), 12)
        self.assertEqual(reference.events(encoded(events())), list((*reference.ROOTS, *reference.CASES)))
        for mutation in ('missing', 'duplicate', 'skip', 'fail', 'package', 'foreign'):
            rows = events()
            if mutation == 'missing': rows.pop(2)
            if mutation == 'duplicate': rows.insert(2, rows[2])
            if mutation in ('skip', 'fail'): rows[2]['Action'] = mutation
            if mutation == 'package': rows[2]['Package'] = 'foreign'
            if mutation == 'foreign': rows[2]['Test'] = 'TestTerminalExitMeasurementControls'
            with self.subTest(mutation=mutation), self.assertRaises(Refusal):
                reference.events(encoded(rows))

    def test_retired_assertions_and_actual_native_guard_preimages_are_required(self):
        for root in reference.ROOTS:
            self.assertTrue(gate.applicable(root, contract(root)))
            for mutation in ('historical', 'assertion', 'original-pass', 'guard', 'launched', 'restart', 'source'):
                value = contract(root)
                if mutation == 'historical': value['historical_reference']['testcases'].pop()
                if mutation == 'assertion': value['retired_assertions'].pop()
                if mutation == 'original-pass': value['original_pass'] = True
                if mutation == 'guard': value['native_guards'][0] = value['native_guards'][1]
                if mutation == 'launched': value['native_guards'][5]['observed']['activity']['launched_stores'] = 1
                if mutation == 'restart': value['native_guards'][-1]['observed']['after_refusal_incarnation'] = 2
                if mutation == 'source': value['source_bindings'].pop(gate.SOURCES[0])
                with self.subTest(root=root, mutation=mutation):
                    self.assertFalse(gate.applicable(root, value))

    def test_only_dedicated_complete_contract_can_receive_each_unchanged_root(self):
        roots = list(reference.ROOTS)
        facts = {root: contract(root) for root in roots}
        rows, complete = broad_stages.closure(roots, {'cluster-wire': dict(roots=facts)}, {})
        self.assertTrue(complete)
        self.assertEqual([row['id'] for row in rows], roots)
        _, complete = broad_stages.closure(roots, {'generic-pass': dict(roots=facts)}, {})
        self.assertFalse(complete)
        facts[roots[0]]['historical_reference']['testcases'].pop()
        rows, complete = broad_stages.closure(roots, {'cluster-wire': dict(roots=facts)}, {})
        self.assertFalse(complete)
        self.assertEqual(rows[0]['status'], 'REFUSED')

    def test_missing_or_duplicate_native_row_cannot_be_hidden_by_historical_pass(self):
        stdout = encoded(guards()).decode()
        with patch.object(gate.lifecycle, 'parse_rows') as parse:
            self.assertEqual(gate.native_guards(stdout), guards())
            parse.assert_called_once_with(stdout)
            with self.assertRaises(Refusal):
                gate.native_guards(encoded(guards()[:-1]).decode())
            changed = guards()
            changed[-1] = changed[0]
            with self.assertRaises(Refusal):
                gate.native_guards(encoded(changed).decode())


if __name__ == '__main__':
    unittest.main()
