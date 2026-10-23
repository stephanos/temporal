"""Adversarial raw-reader checks; these synthetic inputs carry no original-root credit."""
import base64
import copy
from pathlib import Path
import sys
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import core_retained_matrix as gate
from driver import Refusal


def encoded(raw):
    return base64.b64encode(raw).decode()


def fixture():
    handles = [dict(node='a', incarnation=1), dict(node='b', incarnation=1), dict(node='a', incarnation=2)]
    inputs = dict(schema='gomad.cluster-replay-inputs/v1', runtime_implementation_sha256='a' * 64,
        configuration=dict(steps=[dict(handle=handle, action='wait') for handle in handles],
            broker=dict(limits=dict(max_transitions=5)), world_encoding='compressed', world_profile='compressed',
            incarnations=[dict(handle=handle, module=dict(bytes=8, sha256='a'*64), namespace=dict(epoch_nanos=0),
                limits=dict(wall_millis=1000), model=dict(limit=10), choice_identity=dict(ToolchainBuildKey='b'*64)) for handle in handles]))
    payloads = {gate.FILE: gate.canonical(inputs), 'evidence/world': bytes(range(128)),
                'target/module.wasm': b'\0asm\1\0\0\0'}
    identity = dict(target=gate.digest(payloads['target/module.wasm']), toolchain='b'*64,
                    configuration=gate.digest(gate.canonical(inputs['configuration'])), platform='test', implementation='c'*64)
    record = dict(schema='gomad.execution-record/v2', identity=identity,
        execution_identity=gate.digest(gate.canonical(identity)), profile='test', seed=7, arguments=[], environment={},
        limits={}, virtual_time_nanos=0, outcome=dict(termination='exit', exit_code=0),
        streams={name:dict(bytes=len(raw), sha256=gate.digest(raw), records=0)
                 for name, raw in payloads.items() if name != 'target/module.wasm'})
    payloads['record.json'] = gate.canonical(record)
    manifest = dict(schema='gomad.artifact-manifest/v2', record=record, record_sha256=gate.digest(payloads['record.json']),
        files={name:dict(bytes=len(raw), sha256=gate.digest(raw), module=name=='target/module.wasm') for name,raw in payloads.items()})
    return dict(manifest_base64=encoded(gate.canonical(manifest)), record_base64=encoded(payloads['record.json']),
                payloads_base64={name:encoded(raw) for name,raw in payloads.items()})


def snapshots(actual):
    raw = dict(actual['payloads'], **{'manifest.json':gate.canonical(actual['manifest']),
                                    'storage.json':b'{"format":"gomad.artifact-storage/v2"}'})
    return {'artifact/'+name:dict(bytes=len(value), sha256=gate.digest(value)) for name,value in raw.items()}


def boundaries(actual):
    entries = snapshots(actual)
    values = dict(verify=dict(execution_identity=actual['record']['execution_identity']),
                  **{'storage-projection':dict(selection_entries=0, selection_bytes_base64=encoded(b'[]')),
                     'prepare-reopen':dict(exact=True, available=True),
                     'public-cli-replay':dict(status=0, output_base64=encoded(b'gomad: reproduced=true\n'))})
    return {name:dict(error=None, value=value, entries_before=copy.deepcopy(entries), entries_after=copy.deepcopy(entries),
                     entry_count_before=len(entries), entry_count_after=len(entries)) for name,value in values.items()}


class RetainedMatrixRawReader(unittest.TestCase):
    def test_actual_manifest_record_payload_and_stream_preimages(self):
        raw = fixture()
        actual = gate.artifact(raw)
        gate.snapshot(snapshots(actual), actual)
        for name in ('manifest_base64', 'record_base64'):
            changed = copy.deepcopy(raw)
            changed[name] = encoded(gate.b64(changed[name]) + b'\n')
            with self.subTest(name=name), self.assertRaises(Refusal):
                gate.artifact(changed)
        changed = copy.deepcopy(raw)
        changed['payloads_base64'][gate.FILE] = encoded(b'{}')
        with self.assertRaises(Refusal): gate.artifact(changed)

    def test_resealed_manifest_cannot_replace_record_stream_or_module_kind(self):
        raw = fixture()
        for forged in ('stream', 'module-kind'):
            changed = copy.deepcopy(raw)
            manifest = gate.decode(gate.b64(changed['manifest_base64']))
            if forged == 'stream':
                changed['payloads_base64'][gate.FILE] = encoded(b'{}')
                manifest['files'][gate.FILE].update(bytes=2, sha256=gate.digest(b'{}'))
            else:
                manifest['files']['target/module.wasm']['module'] = False
            changed['manifest_base64'] = encoded(gate.canonical(manifest))
            with self.subTest(forged=forged), self.assertRaises(Refusal): gate.artifact(changed)

    def test_original_record_and_root_mutations_remain_authenticated_only_at_outer_layer(self):
        before = gate.artifact(fixture())
        for case, failure in [('changed-original-record', 'noncanonical-record'), ('changed-root-identity', 'execution-identity')]:
            payloads, manifest = gate.expected_mutation(before, case)
            raw = dict(manifest_base64=encoded(gate.canonical(manifest)), record_base64=encoded(payloads['record.json']),
                       payloads_base64={name:encoded(value) for name,value in payloads.items()})
            with self.subTest(case=case):
                self.assertIn(failure, gate.artifact(raw, authenticate=False)['violations'])
                with self.assertRaises(Refusal): gate.artifact(raw)

    def test_every_exact_mutation_preserves_original_configuration_identity_and_reseals_real_bytes(self):
        before = gate.artifact(fixture())
        self.assertEqual(len(gate.MUTATIONS), 21)
        self.assertEqual(len(set(gate.MUTATIONS)), 21)
        for case in gate.MUTATIONS:
            with self.subTest(case=case):
                payloads, manifest = gate.expected_mutation(before, case)
                self.assertNotEqual(payloads, before['payloads'])
                self.assertEqual(manifest['record']['identity'], before['record']['identity'])
                for name, raw in payloads.items():
                    self.assertEqual(manifest['files'][name]['bytes'], len(raw))
                    self.assertEqual(manifest['files'][name]['sha256'], gate.digest(raw))
        for case, path in [('foreign-module', ('configuration','incarnations',0,'module','sha256')),
                           ('boot-module-differs', ('configuration','incarnations',1,'module','sha256')),
                           ('changed-model-implementation', ('runtime_implementation_sha256',))]:
            payloads, _ = gate.expected_mutation(before, case)
            value = gate.decode(payloads[gate.FILE])
            for key in path: value = value[key]
            self.assertEqual(value, '0'*64)

    def test_receiving_boundaries_require_raw_snapshot_no_write_and_actual_positive_values(self):
        actual = gate.artifact(fixture())
        original = boundaries(actual)
        gate.boundary_facts(original, admitted=True, actual=actual)
        for forged in ('write', 'missing-file', 'count', 'identity', 'selection', 'replay', 'missing-boundary'):
            changed = copy.deepcopy(original)
            if forged == 'write': changed['verify']['entries_after']['new'] = dict(directory=True)
            elif forged == 'missing-file':
                for entry in ('entries_before','entries_after'): changed['verify'][entry].pop('artifact/'+gate.FILE)
                for entry in ('entry_count_before','entry_count_after'): changed['verify'][entry] -= 1
            elif forged == 'count': changed['verify']['entry_count_after'] += 1
            elif forged == 'identity': changed['verify']['value']['execution_identity'] = '0'*64
            elif forged == 'selection': changed['storage-projection']['value']['selection_bytes_base64'] = encoded(b'{}')
            elif forged == 'replay': changed['public-cli-replay']['value']['output_base64'] = encoded(b'reproduced=false')
            else: changed.pop('verify')
            with self.subTest(forged=forged), self.assertRaises(Refusal):
                gate.boundary_facts(changed, admitted=True, actual=actual)
        rejected = copy.deepcopy(original)
        for row in rejected.values(): row.update(error='actual refusal', value=None)
        gate.boundary_facts(rejected, admitted=False, actual=actual)
        rejected['verify']['error'] = ''
        with self.assertRaises(Refusal): gate.boundary_facts(rejected, admitted=False, actual=actual)

    def test_missing_duplicate_and_reordered_case_inventories_are_closed(self):
        cases = ['valid', *gate.MUTATIONS, 'module-nonembedding']
        for changed in (cases[:-1], cases+['valid'], [cases[1],cases[0],*cases[2:]]):
            stdout = '\n'.join(gate.MATRIX_MARKER+gate.canonical(dict(case=case)).decode() for case in changed)
            with self.assertRaises(Refusal): gate.lifecycle_matrix(stdout)

    def test_large_module_requires_real_bounded_custom_section_padding(self):
        def leb(value):
            output = bytearray()
            while value >= 128:
                output.append((value & 127) | 128)
                value >>= 7
            output.append(value)
            return bytes(output)
        name = b'gomad-retention-padding-v1'
        custom = leb(len(name))+name+b'\0'*((1<<20)-27)
        module = b'\0asm\1\0\0\0'+b'\0'+leb(len(custom))+custom
        self.assertTrue(gate.wasm_custom_padding(module))
        self.assertFalse(gate.wasm_custom_padding(module[:-1]+b'X'))
        with self.assertRaises(Refusal): gate.wasm_custom_padding(module[:-1])
        with self.assertRaises(Refusal): gate.wasm_custom_padding(b'wrong'+module)


if __name__ == '__main__': unittest.main()
