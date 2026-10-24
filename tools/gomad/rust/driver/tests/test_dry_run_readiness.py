"""Dry-run reader counterexamples; subprocess fixtures earn no qualification evidence."""
import copy
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import time
import unittest
from unittest.mock import patch

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
sys.path.insert(0, str(Path(__file__).resolve().parent))
import driver
import dry_run
import executed_divergence
import main_cli_build
import native_roles
import test_native_roles

CONTEXT = dict(compiler_utf8='rustc 1.99.1 (reader fixture)\n',
               bindings=dict(PROFILE='release', TARGET='aarch64-unknown-linux-gnu'))


def record(root, config, role):
    output = root / role
    output.mkdir()
    binary = output / main_cli_build.binary_name(role)
    binary.write_bytes(b'#!/bin/sh\nexit 0\n# ' + role.encode())
    binary.chmod(0o755)
    identity, runtime, _ = main_cli_build.identities(dict(config, main_role=role), CONTEXT)
    helper = role in main_cli_build.HELPERS
    generated = output / 'helper-out'
    generated.mkdir()
    (generated / 'identity-compiler.txt').write_text(CONTEXT['compiler_utf8'])
    (generated / 'identity-bindings.tsv').write_text(''.join(k+'\t'+v+'\n' for k,v in sorted(
        native_roles.feature_bindings(main_cli_build.helper_features(role), CONTEXT['bindings']['TARGET']).items())))
    frames = [dict(reason='build-script-executed', package_id='path+file:///helper#gomad-wasmhost@0.1.0',
                   out_dir=str(generated), env=[['GOMAD_LINKED_IMPLEMENTATION_SHA256',identity['helper']]])]
    if not helper:
        frames.append(dict(reason='build-script-executed', package_id='path+file:///host#gomad@0.1.0',
            env=[['GOMAD_HOST_IMPLEMENTATION_SHA256',identity['host']], ['GOMAD_RUNTIME_IMPLEMENTATION_SHA256',runtime]]))
    frames += [dict(reason='compiler-artifact', executable=str(binary),
        manifest_path=str(Path(config['repo'])/'tools/gomad/rust'/('wasmhost' if helper else 'gomad')/'Cargo.toml'),
        target=dict(name=main_cli_build.EXAMPLES.get(role, 'gomad_wasmhost' if helper and role in main_cli_build.LIBTESTS else 'gomad-wasmhost' if helper else 'gomad')),
        features=main_cli_build.ROLES[role], profile=dict(opt_level='3',debug_assertions=False,test=role in main_cli_build.LIBTESTS)),
        dict(reason='build-finished',success=True)]
    driver.write(output/'frames.json',frames)
    (output/'cargo.stdout').write_text('\n'.join(json.dumps(row) for row in frames))
    path = output/'dry-build.json'
    driver.write(path, dict(schema=dry_run.SCHEMA, role=role, binary=str(binary), binary_sha256=driver.sha(binary),
        build_head='a'*40, build_dirty=False, features=main_cli_build.ROLES[role],
        helper_features=main_cli_build.helper_features(role),host_sha256=identity['host'],helper_sha256=identity['helper'],
        runtime_implementation_sha256=runtime,argv=dry_run.cargo_command('cargo',role),scope='reader fixture'))
    return path, binary


class DryRunReadiness(unittest.TestCase):
    def tearDown(self):
        dry_run.enable({})

    def fixture(self, root):
        config, _, _, _ = test_native_roles.NativeRoles().fixture(root)
        files = {name:driver.sha(name) for name in config['files'] if Path(name).is_relative_to(config['repo'])}
        source = ''.join(value+'  '+name+'\n' for name,value in sorted(files.items()))
        config.update(source_sha256=hashlib.sha256(source.encode()).hexdigest(),head='b'*40,symlinks={},files=files,dry_run=True)
        return config, patch.object(main_cli_build,'source_state',return_value=(files,config['source_sha256'],config['head'],{}))

    def test_feature_role_rejects_stale_host_helper_features_and_compiler_context(self):
        with tempfile.TemporaryDirectory() as raw:
            root=Path(raw);config,source=self.fixture(root)
            path,_=record(root,config,'host-fixture')
            original=driver.read(path)
            config['identity']=dict(host=original['host_sha256'],helper=original['helper_sha256'])
            dry_run.enable(config)
            with source:
                for field in ('host_sha256','helper_sha256','features','context'):
                    changed=copy.deepcopy(original)
                    if field=='context':
                        context=path.parent/'helper-out/identity-compiler.txt'
                        context.write_text('rustc 1.99.1 (other compiler)\n')
                    else: changed[field]=[] if field=='features' else 'f'*64
                    path.write_text(json.dumps(changed))
                    with self.subTest(field=field),self.assertRaises(driver.Refusal):
                        main_cli_build.read_build(path,config,expected_role='host-fixture')
                    (path.parent/'helper-out/identity-compiler.txt').write_text(CONTEXT['compiler_utf8'])
                path.write_text(json.dumps(original))
                # Mutating actual feature-bearing host source invalidates an otherwise unchanged role.
                host=Path(config['repo'])/'tools/gomad/rust/gomad/src/bin/measure.rs'
                host.write_bytes(host.read_bytes()+b'changed')
                config['files'][str(host)]=driver.sha(host)
                with self.assertRaises(driver.Refusal): main_cli_build.read_build(path,config)

    def test_counterfactual_roles_share_actual_compiled_context(self):
        with tempfile.TemporaryDirectory() as raw:
            root=Path(raw);config,source=self.fixture(root)
            cli,_=record(root,config,'counterfactual-fixture')
            harness,_=record(root,config,'counterfactual-harness')
            dry_run.enable(config)
            with source:
                first,second=executed_divergence.roles(cli,harness)
                self.assertEqual(first['context'],CONTEXT)
                self.assertEqual(second['context'],CONTEXT)
                self.assertEqual(first['source_sha256'],config['source_sha256'])

    def test_clean_child_loads_context_and_reaches_private_capture_campaign_admission(self):
        import host_admission
        with tempfile.TemporaryDirectory() as raw:
            root=Path(raw)
            files,digest,head,links=main_cli_build.source_state(driver.REPO,'/usr/bin/git')
            config=dict(repo=str(driver.REPO),files=files,source_sha256=digest,head=head,symlinks=links)
            build,binary=record(root,config,'release')
            identity=driver.read(build)
            path=root/'config.json'
            config.update(schema='gomad.rust-qualification-driver/v1',dry_run=True,
                stages={label:dict(adapter='primary') for label in driver.mapping()},final_reader=['unexecuted'],final_sections={'unexecuted':{}},
                identity=dict(host=identity['host_sha256'],helper=identity['helper_sha256']),cores=1,mem_gib=1,sent_epoch=time.time(),deadline_s=3600,
                binary=str(binary),binary_sha256=driver.sha(binary),root=str(root/'cycle'),scratch=str(root/'scratch'),
                temporary_paths=dict(TMPDIR=str(root/'scratch/tmp'),GOTMPDIR=str(root/'scratch/go')))
            config['files'].update({str(p):driver.sha(p) for p in root.rglob('*') if p.is_file()})
            config['files'][str(host_admission.STAT)]=driver.sha(host_admission.STAT)
            driver.write(path,config)
            child='''import sys
from pathlib import Path
sys.path.insert(0,sys.argv[1])
import driver,main_cli_build,installed_cluster_failure as installed,native_observations
config=driver.configured(sys.argv[2])
build=main_cli_build.read_build(sys.argv[3],expected_role='release')
root=Path(sys.argv[4]);root.mkdir()
for name in ('compiler','source','runtime','module-cache'):
    (root/name).mkdir();(root/name/'input').write_bytes(b'fixture')
(root/'source-inputs.tsv').write_bytes(installed.source_manifest(build,Path(config['repo'])))
driver.write(root/'producer.json',dict(pid=1,start_ns=1,started_monotonic_nanos=1))
run=installed.Run(root,dict(repo=config['repo']),build,True)
try:native_observations.capture(run.config,2,[])
except ValueError as error:
    assert str(error)=='empty/duplicate private campaign contexts',str(error)
else:raise AssertionError('missing campaign was accepted')
assert not any(name.startswith('GOMAD') for name in installed.environment(root))
'''
            observed=subprocess.run([sys.executable,'-c',child,str(driver.HERE),str(path),str(build),str(root/'private')],env=driver.clean_env(config),capture_output=True,text=True)
            self.assertEqual(observed.returncode,0,observed.stderr)
            # Exercise both actual installed entrypoints with their sealed configuration and no ambient opt-in.
            for script in ('installed_cluster_failure.py','installed_lifecycle_families.py'):
                argv=[sys.executable,str(driver.HERE/script),'produce','--config',str(path),'--main-build',str(build),
                    '--compiler-root',str(root/'absent-compiler'),'--guest-config',str(root/'guest'),'--module-cache',str(root/'cache'),'--root',str(root/script)]
                if script=='installed_lifecycle_families.py':argv+=['--family','typed-detached']
                observed=subprocess.run(argv,env=driver.clean_env(config),capture_output=True,text=True)
                self.assertNotEqual(observed.returncode,0)
                self.assertIn('stock compiler root required',observed.stderr)
            parity_child = '''import sys
from pathlib import Path
sys.path.insert(0,sys.argv[1])
import driver,main_cli_build,final_public_verbs
lane=final_public_verbs.orchestrator()
build=Path(sys.argv[3])
def binding(path,digest,frozen=None):
    assert frozen is not None,'lost driver configuration at parity boundary'
    main_cli_build.read_build(build,frozen,expected_role='release')
    raise ValueError('fixture stopped after authenticated public role')
lane.load_bindings=binding
sys.argv=[str(lane.SCRIPT) if hasattr(lane,'SCRIPT') else 'all-verb-parity','--config',sys.argv[2],
          '--bindings',str(build),'--bindings-sha256',driver.sha(build),'--work',sys.argv[4]]
assert lane.main()==1
'''
            observed=subprocess.run([sys.executable,'-c',parity_child,str(driver.HERE),str(path),str(build),str(root/'parity')],env=driver.clean_env(config),capture_output=True,text=True)
            self.assertEqual(observed.returncode,0,observed.stderr)
            self.assertIn('fixture stopped after authenticated public role',observed.stderr)
