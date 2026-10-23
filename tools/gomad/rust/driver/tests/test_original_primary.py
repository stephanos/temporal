"""Pure original-stage recipe and raw replay refusal controls; no guests run."""
import copy
import json
import os
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import clusters_receiving as clusters
import ordinary_receiving as ordinary
import primary_stages as primary
import receiving
from driver import Refusal, sha, write


class OriginalClusterReplay(unittest.TestCase):
    def fixture(self, root):
        artifact, output = root / 'artifact', root / 'replay'
        summary, observed, incarnations, retained, files = [], [], [], [], {}
        for node, number in sorted(clusters.HANDLES):
            handle = dict(node=node, incarnation=number)
            terminal = 'crash' if number == 1 else 'exit'
            streams = {}
            for name in ('stdout', 'choice', 'transcript', 'storage', 'storage_image'):
                raw = f'{node}/{number}/{name}'.encode()
                source = artifact / f'evidence/cluster/{node}/{number}/{name}'
                target = output / f'incarnations/{node}/{number}/{name}'
                source.parent.mkdir(parents=True, exist_ok=True);target.parent.mkdir(parents=True, exist_ok=True)
                source.write_bytes(raw);target.write_bytes(raw);target.chmod(0o600)
                streams[name] = dict(bytes=len(raw), sha256=sha(source))
                files[str(source.relative_to(artifact))] = dict(streams[name], module=False)
            summary.append(dict(handle=handle, termination=terminal, exit_code=None if number == 1 else 0,
                                interrupted=number == 1, reason='', streams=streams))
            observed.append(dict(handle=handle, outcome=dict(termination=terminal, exit_code=None if number == 1 else 0, message=''),
                                 interrupted=number == 1, authorization_to_terminal_nanos=1, streams=streams))
            choice = f'evidence/cluster/{node}/{number}/choice'
            retained.append(dict(handle=handle, interrupted=number == 1,
                                 evidence=dict(choice=dict(file=choice, limit=1000, **files[choice]))))
            incarnations.append(dict(handle=handle, module=dict(file='target/module.wasm', bytes=1, sha256='', limit=1000),
                                     choice_identity=dict(owner=node, incarnation=number)))
        module = artifact / 'target/module.wasm';module.parent.mkdir();module.write_bytes(b'x')
        files['target/module.wasm'] = dict(bytes=1, sha256=sha(module), module=True)
        for item in incarnations:item['module']['sha256'] = sha(module)
        state = dict(incarnations=summary,original_outcome=dict(termination='exit',exit_code=0,reason=''))
        for name in ('world', 'network'):
            source, target = artifact / ('evidence/' + name), output / ('streams/' + name)
            target.parent.mkdir(parents=True, exist_ok=True)
            source.write_bytes(name.encode());target.write_bytes(name.encode());target.chmod(0o600)
            state[name + '_sha256'] = sha(source)
        (artifact / 'record.json').write_text('{}')
        write(output / 'result.json', dict(schema='gomad.live-cluster-observation/v1',execution_returned=True,
              wall_elapsed_nanos=1,incarnations=observed,outcome=dict(termination='exit',exit_code=0,message='')))
        (output / 'result.json').chmod(0o600)
        (output.parent / 'observe.stdout').write_bytes((output / 'result.json').read_bytes())
        return artifact, output, state, dict(configuration=dict(incarnations=incarnations), retained=retained), dict(files=files)

    def test_both_roles_execute_interrupted_prefix_then_full_reopened_tapes(self):
        with tempfile.TemporaryDirectory() as temporary:
            artifact, output, summary, inputs, manifest = self.fixture(Path(temporary))
            payload = clusters.replay_input(artifact, inputs, manifest)
            self.assertEqual({key:row['mode'] for key,row in payload['forcing'].items()},
                             {'active/1':3,'standby/1':3,'active/2':2,'standby/2':2})
            proof = clusters.compare(artifact, summary, output)
            self.assertEqual(len(proof['phases']),4)
            for mutation in ('missing-prefix','full-only','foreign-payload'):
                changed = copy.deepcopy(inputs)
                if mutation == 'missing-prefix':changed['retained'].pop(0)
                if mutation == 'full-only':changed['retained'][0]['interrupted'] = False
                if mutation == 'foreign-payload':changed['retained'][0]['evidence']['choice']['file'] = '../outside'
                with self.subTest(mutation=mutation),self.assertRaises(Refusal):clusters.replay_input(artifact,changed,manifest)

    def test_every_phase_stream_and_shared_record_is_compared_as_actual_bytes(self):
        for relative in ['streams/world','streams/network',
                         *(f'incarnations/{node}/{number}/{name}' for node,number in clusters.HANDLES
                           for name in ('choice','transcript','stdout','storage','storage_image'))]:
            with self.subTest(stream=relative),tempfile.TemporaryDirectory() as temporary:
                artifact, output, summary, _, _ = self.fixture(Path(temporary))
                (output / relative).write_bytes(b'changed')
                with self.assertRaises(Refusal):clusters.compare(artifact,summary,output)

    def test_generic_success_or_missing_replay_phase_never_supplies_cluster_proof(self):
        with tempfile.TemporaryDirectory() as temporary:
            artifact, output, summary, _, _ = self.fixture(Path(temporary))
            row = json.loads((output / 'result.json').read_bytes());row['incarnations'].pop()
            (output / 'result.json').write_text(json.dumps(row))
            with self.assertRaises(Refusal):clusters.compare(artifact,summary,output)
            (output / 'result.json').write_text(json.dumps(dict(matched=True,target_success=True)))
            with self.assertRaises((Refusal,KeyError)):clusters.compare(artifact,summary,output)

    def test_replay_output_cannot_alias_the_baseline_bytes(self):
        for kind in ('symlink', 'hardlink'):
            with self.subTest(kind=kind),tempfile.TemporaryDirectory() as temporary:
                artifact, output, summary, _, _ = self.fixture(Path(temporary))
                destination = output / 'streams/world'
                destination.unlink()
                if kind == 'symlink':destination.symlink_to(artifact / 'evidence/world')
                else:
                    (artifact / 'evidence/world').chmod(0o600)
                    os.link(artifact / 'evidence/world', destination)
                with self.assertRaises(Refusal):clusters.compare(artifact,summary,output)

    def test_actual_observer_stdout_must_equal_private_result(self):
        with tempfile.TemporaryDirectory() as temporary:
            artifact, output, summary, _, _ = self.fixture(Path(temporary))
            (output.parent / 'observe.stdout').write_text('{}')
            with self.assertRaisesRegex(Refusal,'command output differs'):clusters.compare(artifact,summary,output)

    def test_ordinary_lists_exact_selectors_and_preserves_recursive_skip(self):
        config = dict(repo='/repo',python='/python')
        with patch.object(ordinary,'plan',return_value=dict(tools=dict(go='/go'))), \
             patch.object(ordinary.live,'sealed_input') as sealed, \
             patch.object(ordinary.static,'command_environment',return_value={}), \
             patch.object(ordinary.static,'reference_environment',return_value={}), \
             patch.object(ordinary.live,'role',side_effect=lambda _config,name:dict(binary='/' + name)):
            rows = ordinary.recipes(config)
        by_name = {row['name']:row for row in rows}
        for name in ('namespace','builder','recursive','recursive-builder','slim'):
            listing = by_name['ordinary-' + name + '-list']['argv']
            executed = by_name['ordinary-' + name + '-run']['argv']
            self.assertEqual(listing[listing.index('-list')+1],executed[executed.index('-run')+1])
        for suffix in ('list','run'):
            argv = by_name['ordinary-recursive-' + suffix]['argv']
            self.assertEqual(argv[argv.index('-skip')+1],'^TestNamespaceConformance')
        self.assertEqual(by_name['ordinary-namespace-run']['timeout_s'],600)
        self.assertEqual(by_name['ordinary-builder-run']['timeout_s'],900)
        sealed.assert_called_once_with(config,'/python')
        self.assertEqual(by_name[ordinary.VOLUME_REPEAT]['argv'],
                         ['/ordinary-helper-test','--exact',*ordinary.VOLUME_TESTS,'--test-threads=1'])
        self.assertEqual(by_name[ordinary.WORKLOAD_TOOLS]['argv'],
                         ['/python','-m','unittest','discover','-v','-s','/repo/tools/gomad/rust/workload_tools','-p','test_*.py'])

    def test_ordinary_workload_tool_cases_require_every_listed_pass(self):
        ok = (b'test_a (test_x.Case.test_a) ... ok\ntest_b (test_x.Case.test_b) ... ok\n\n'
              b'----------------------------------------------------------------------\nRan 2 tests in 0.010s\n\nOK\n')
        self.assertEqual(ordinary.python_cases(ok),['test_x.Case.test_a','test_x.Case.test_b'])
        for bad in (ok.replace(b'Ran 2',b'Ran 3'), ok.replace(b'test_b) ... ok',b'test_b) ... FAIL'),
                    ok.replace(b'\nOK\n',b'\nFAILED (failures=1)\n'), ok.replace(b'test_b (test_x.Case.test_b) ... ok\n',b''),
                    ok.replace(b'test_b (test_x.Case.test_b)',b'test_a (test_x.Case.test_a)'), b''):
            with self.subTest(bad=bad[-40:]),self.assertRaises(Refusal):ordinary.python_cases(bad)

    def test_original_mode_changes_only_repeat_and_rejects_canary_or_other_workload(self):
        workloads = receiving.workloads
        common = dict(repo=Path('/repo'),root=Path('/result'),workload='replication',source='/source')
        normal = workloads.qualification_spec(**common)
        original = workloads.qualification_spec(**common,original_clusters=True)
        self.assertEqual(normal['Manifest']['Repeat'],2)
        self.assertEqual(original['Manifest']['Repeat'],1)
        normal['Manifest']['Repeat'] = 1
        self.assertEqual(normal,original)
        for kwargs in (dict(common,workload='recovery'),dict(common,canary=True)):
            with self.assertRaises(ValueError):workloads.qualification_spec(**kwargs,original_clusters=True)

    def test_primary_routes_are_closed_and_original_stage_bounds_remain(self):
        self.assertTrue({'ordinary','clusters'} <= primary.OWNED)
        self.assertEqual(primary.MISSING,{})
        self.assertEqual(primary.TIMEOUTS['clusters'],5400)
        self.assertEqual(primary.TIMEOUTS['ordinary'],3600)
        self.assertEqual(set(ordinary.ROLES),{'host','helper'})


if __name__ == '__main__':unittest.main()
