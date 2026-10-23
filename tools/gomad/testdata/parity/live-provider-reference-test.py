#!/usr/bin/env python3
"""Pure provider custody controls; synthetic rows earn no original-Go acceptance."""
import copy
import contextlib
import io
import importlib.util
import json
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.dont_write_bytecode = True
HERE = Path(__file__).resolve().parent
sys.path.insert(0,str(HERE.parents[1]/'rust/driver'))
import main_cli_build


def load(name):
    spec = importlib.util.spec_from_file_location(name,HERE/(name+'.py'))
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


check = load('live-cluster-check')
observe = load('live-broker-observe')


class ProviderReferenceCustody(unittest.TestCase):
    def fixture(self, root):
        binary = root/'reference-helper'
        binary.write_bytes(b'actual publication fixture')
        receipt = root/'helper-build.json'
        receipt.write_bytes(b'{}')
        build = dict(role='reference-transport',features=['reference-helper'],helper_features=['reference-helper'],
            binary=str(binary),binary_sha256=check.digest(binary.read_bytes()),host_sha256=None,
            runtime_implementation_sha256=None)
        binding = dict(path=str(receipt),sha256=check.digest(receipt.read_bytes()))
        return binary,receipt,build,binding

    def test_exact_typed_role_publication_and_frozen_binding_are_required(self):
        with tempfile.TemporaryDirectory() as temporary:
            binary,receipt,build,binding = self.fixture(Path(temporary))
            frozen = {'frozen':'current candidate'}
            with patch.object(main_cli_build,'read_build',return_value=build) as reader:
                self.assertEqual(check.reference_helper(binding,build['binary_sha256'],frozen),build)
                reader.assert_called_once_with(receipt,frozen,expected_role='reference-transport')
                for mutation in [dict(role='conformance-reference'),dict(features=['conformance','reference-helper']),
                    dict(helper_features=['conformance','reference-helper']),dict(host_sha256='invented-host'),
                    dict(runtime_implementation_sha256='invented-runtime'),dict(binary_sha256='wrong')]:
                    reader.return_value = dict(build,**mutation)
                    with self.subTest(mutation=mutation),self.assertRaises(AssertionError):
                        check.reference_helper(binding,build['binary_sha256'],frozen)
                reader.return_value = build
                binary.write_bytes(b'changed publication')
                with self.assertRaises(AssertionError):check.reference_helper(binding,build['binary_sha256'])
            receipt.write_text(json.dumps(dict(profile='release',features=[],exit=0,binary_sha256='historical',role='release')))
            legacy = dict(path=str(receipt),sha256=check.digest(receipt.read_bytes()))
            with self.assertRaises(ValueError):check.reference_helper(legacy,'historical')

    def test_actual_provider_requires_typed_receipt_before_any_tool_or_worktree(self):
        argv = ['live-broker-observe.py','--provider-roots','19']
        for name in ('source','derived','work','compiler','helper'):
            argv.extend(['--'+name,'/not-provisioned/'+name])
        with patch.object(sys,'argv',argv), patch.object(observe.subprocess,'run',side_effect=AssertionError('tool ran')), \
             contextlib.redirect_stderr(io.StringIO()),self.assertRaises(SystemExit) as refused:
            observe.main()
        self.assertEqual(refused.exception.code,2)

    def test_observer_propagates_full_receipt_binding_and_exact_published_path(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            binary,receipt,build,binding = self.fixture(root)
            with patch.object(main_cli_build,'read_build',return_value=build):
                self.assertEqual(observe.provider_helper_receipt(receipt,binary),binding)
                other = root/'same-bytes-another-helper'
                other.write_bytes(binary.read_bytes())
                with self.assertRaisesRegex(ValueError,'publication'):
                    observe.provider_helper_receipt(receipt,other)

    def test_actual_open_boots_cannot_cross_bind_a_helper_after_resealing_index(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            binary,receipt,build,binding = self.fixture(root)
            name = 'TestProviderClusterStartsOnlyRequestedIncarnations'
            module = root/'module.wasm'
            module.write_bytes(b'\0asm\x01\0\0\0')
            request = dict(HelperPath=str(binary),HelperSHA256=build['binary_sha256'],
                ModulePath=str(module),ModuleSHA256=check.digest(module.read_bytes()))
            opens = [dict(kind='open',role='original-private-go-broker-reference',admitted=True,error='',
                go_broker_reference=True,spec=dict(fidelity='hard_isolation')),
                dict(kind='open',role='original-one-helper-provider',admitted=False,error='hard isolation',
                    spec=dict(fidelity='hard_isolation')),
                dict(kind='open',role='original-one-helper-provider',admitted=True,error='',helper_started_at_open=False,
                    lazy_cluster_request_at_open=True,spec=dict(nodes=[dict(id='a'),dict(id='b'*128)]))]
            for row in opens:row['boots']={'guest':copy.deepcopy(request)}
            rows = opens+[dict(kind='finish',error='',evidence=dict(Execution=dict(incarnations=[dict(handle=check.handle('a'))])),
                frames=dict(Results=1,GuestCallbacks=0))]
            observations = root/'observations'
            observations.mkdir()
            for ordinal,row in enumerate(rows,1):
                row.update(test=name,ordinal=ordinal)
                (observations/f'provider-{ordinal:06}.json').write_bytes(check.canonical(row))
            (root/'test-only-helper.go').write_bytes(b'helper')
            (root/'test-only.diff').write_bytes(b'diff')
            derivation = dict(roots=[19],oracle_role='original-private-go-broker-reference-and-original-one-helper-provider',
                original_test_sha256=check.digest(check.provider_source()),helper_sha256=check.digest(b'helper'),
                diff_sha256=check.digest(b'diff'),helper_build_receipt=binding,oracle_helper_sha256=build['binary_sha256'])
            (root/'derivation.json').write_bytes(check.canonical(derivation))
            (root/'go-test.jsonl').write_bytes(check.canonical(dict(Test=name,Action='pass'))+b'\n')
            (root/'command.json').write_bytes(b'{"exit":0}')
            def reseal():
                index = dict(schema='gomad.live-broker-observation-index/v1',
                    derivation_sha256=check.digest((root/'derivation.json').read_bytes()),
                    go_test_sha256=check.digest((root/'go-test.jsonl').read_bytes()),files={
                        path.name:dict(bytes=path.stat().st_size,sha256=check.digest(path.read_bytes()))
                        for path in observations.iterdir()})
                (root/'observation-index.json').write_bytes(check.canonical(index))
            reseal()
            with patch.object(main_cli_build,'read_build',return_value=build):
                self.assertEqual(check.provider_reference(root)['helper_role'],'reference-transport')
                path = observations/'provider-000003.json'
                original_row = path.read_bytes()
                path.write_bytes(check.canonical(dict(rows[2],boots={})))
                reseal()
                with self.assertRaises(AssertionError):check.provider_reference(root)
                path.write_bytes(original_row)
                for mutation in [dict(HelperPath='/same-hash/another-path'),dict(HelperSHA256='historical')]:
                    changed = dict(rows[2],boots={'guest':dict(request,**mutation)})
                    path.write_bytes(check.canonical(changed))
                    reseal()
                    with self.subTest(mutation=mutation),self.assertRaises(AssertionError):check.provider_reference(root)


if __name__ == '__main__':unittest.main()
