"""Fixed release CLI build receipt counterexamples; no compiler or CLI is executed."""
import copy
import json
import itertools
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch

sys.path.insert(0,str(Path(__file__).resolve().parents[1]))
sys.path.insert(0,str(Path(__file__).resolve().parent))
import main_cli_build as gate
from driver import Refusal,sha
import test_native_roles as role_fixture


def fixture(base,role='release'):
    config,_,_,_=role_fixture.NativeRoles().fixture(base)
    repo=Path(config['repo']);work=base/'main-cli';work.mkdir(mode=0o700)
    for path in (repo/'tools/gomad/rust/gomad/src/main.rs',repo/'tools/gomad/rust/gomad/src/lib.rs'):
        path.write_text('fixed main CLI source '+path.name);config['files'][str(path)]=sha(path)
    plan=base/'plan.json';plan.write_text(json.dumps(dict(schema='gomad.driver-static-plan/v1',repo=str(repo),driver_config=str(base/'future.json'),
        tools=dict(python=sys.executable,rustc=str(base/'tools/rustc'),cargo=str(base/'tools/cargo')))))
    config.update(python=sys.executable,sent_epoch=1,_config_path=str(base/'future.json'),main_role=role,source_sha256='s'*64,head='a'*40,
        symlinks={},stages=dict(validate=dict(argv=[sys.executable,'static','--plan',str(plan)])),identity_harness=dict(toolchain_plan=str(base/'gnu-plan.json')))
    config['files'][str(plan)]=sha(plan)
    version='rustc 1.99.1 (synthetic)\n';context=dict(compiler_utf8=version,bindings=dict(PROFILE='release',TARGET='aarch64-unknown-linux-gnu'))
    target='runtime-qualification-replay' if role=='runtime-harness' else 'gomad'
    executable=work/('target/release/examples/'+target if role=='runtime-harness' else 'target/release/gomad');executable.parent.mkdir(parents=True);executable.write_bytes(b'synthetic compiled CLI');executable.chmod(0o755)
    identity,runtime,inventories=gate.identities(config,context);frames=[]
    for kind,name,package in (('host','gomad','gomad'),('helper','wasmhost','gomad-wasmhost')):
        crate=repo/'tools/gomad/rust'/name;pkg='path+'+crate.as_uri()+('#0.1.0' if kind=='host' else '#'+package+'@0.1.0')
        generated=work/('target/release/build/'+package+'-abcd/out');generated.mkdir(parents=True)
        (generated/'measurement-source-inputs.tsv').write_text(gate.native_roles.source_manifest(inventories[kind][1]))
        if kind=='helper':
            (generated/'identity-compiler.txt').write_text(version)
            (generated/'identity-bindings.tsv').write_text('PROFILE\trelease\nTARGET\taarch64-unknown-linux-gnu\n')
        env=[['GOMAD_RUNTIME_IMPLEMENTATION_SHA256',runtime],['GOMAD_HOST_IMPLEMENTATION_SHA256',identity['host']]] if kind=='host' else [['GOMAD_LINKED_IMPLEMENTATION_SHA256',identity['helper']]]
        frames.append(dict(reason='build-script-executed',package_id=pkg,out_dir=str(generated),env=env))
        frames.append(dict(reason='compiler-artifact',package_id=pkg,target=dict(kind=['lib'],src_path=str(crate/'src/lib.rs')),features=gate.ROLES[role] if kind=='host' else []))
        if kind=='host':frames.append(dict(reason='compiler-artifact',package_id=pkg,manifest_path=str(crate/'Cargo.toml'),
            target=dict(name=target,kind=['example'] if role=='runtime-harness' else ['bin'],crate_types=['bin'],
                src_path=str(crate/('examples/runtime-qualification-replay.rs' if role=='runtime-harness' else 'src/main.rs'))),features=gate.ROLES[role],fresh=False,
            profile=dict(opt_level='3',debug_assertions=False,test=False),executable=str(executable)))
    frames.append(dict(reason='build-finished',success=True))
    def observed(name,argv,output,pid):
        env=environment(config,work,context['bindings']['TARGET'] if name=='cargo-build' else None)
        (work/(name+'.stdout')).write_text(output);(work/(name+'.stderr')).write_text('')
        value=dict(argv=argv,cwd=config['repo'],pid=pid,exit=0,observation='completed',start_ns=1_100_000_000,end_ns=1_200_000_000,
            environment=env,effective_environment=env,environment_mode='exact',temporary_paths={},clock='CLOCK_MONOTONIC',
            parent_pid=300,started_monotonic_nanos=20,reaped_monotonic_nanos=80)
        (work/(name+'.command.json')).write_text(json.dumps(value))
        (work/(name+'.process.json')).write_text(json.dumps(dict(value,stage_pid=300,active=False)))
    tools=json.loads(plan.read_text())['tools']
    for index,(name,argv,output) in enumerate((('rustc-version',[tools['rustc'],'--version'],version),
        ('rustc-verbose',[tools['rustc'],'-vV'],version+'host: aarch64-unknown-linux-gnu\nrelease: 1.99.1\n'),
        ('cargo-version',[tools['cargo'],'--version'],'cargo 1.99.1 (synthetic)\n'),
        ('cargo-build',gate.build_command(config),'\n'.join(json.dumps(row) for row in frames)))):observed(name,argv,output,201+index)
    (work/'producer.json').write_text(json.dumps(dict(pid=300,start_monotonic_nanos=10,end_monotonic_nanos=100,started_epoch_ns=1_000_000_000)))
    (work/'admitted').mkdir();binary=work/'admitted'/target;binary.write_bytes(executable.read_bytes());binary.chmod(0o755)
    receipt=gate.evidence(config,work);path=work/'main-build.json';path.write_text(json.dumps(receipt))
    return config,path


def environment(config,root,target=None):
    return dict(PATH='/fixed/tools',HOME=str(root/'home'),CARGO_TARGET_DIR=str(root/'target'),TARGET=target or '')


class MainCliBuild(unittest.TestCase):
    def setUp(self):
        selected=patch.object(gate.harness,'environment',side_effect=environment);selected.start();self.addCleanup(selected.stop)

    def test_tracked_symlink_literal_is_git_authenticated_separately_from_regular_digest(self):
        import os
        import shutil
        import subprocess
        git=shutil.which('git')
        with tempfile.TemporaryDirectory() as directory:
            repo=Path(directory);(repo/'regular').write_text('fixed source');(repo/'other').write_text('other source')
            (repo/'tracked-link').symlink_to('regular')
            for args in (['init','--quiet'],['add','regular','other','tracked-link'],
                         ['-c','user.name=unit','-c','user.email=unit@example.invalid','commit','--quiet','-m','source fixture']):
                subprocess.run([git,'-C',str(repo),*args],check=True,stdout=subprocess.PIPE,stderr=subprocess.PIPE)
            files,digest,head,links=gate.source_state(repo,git)
            self.assertNotIn(str(repo/'tracked-link'),files)
            self.assertEqual(links,{str(repo/'tracked-link'):'regular'})
            (repo/'tracked-link').unlink();(repo/'tracked-link').symlink_to('other')
            # A caller can rewrite its table and keep the regular-file digest unchanged;
            # the unchanged Git blob still refuses the coherent literal substitution.
            with self.assertRaisesRegex(Refusal,'actual Git HEAD blob'):gate.source_state(repo,git)
            original=subprocess.check_output([git,'-C',str(repo),'rev-parse','HEAD:tracked-link'],text=True).strip()
            replacement=subprocess.check_output([git,'-C',str(repo),'hash-object','-w','--stdin'],input=b'other').decode().strip()
            subprocess.run([git,'-C',str(repo),'replace',original,replacement],check=True,stdout=subprocess.PIPE,stderr=subprocess.PIPE)
            with self.assertRaisesRegex(Refusal,'actual Git HEAD blob'):gate.source_state(repo,git)
            (repo/'tracked-link').unlink();(repo/'tracked-link').write_text('regular')
            with self.assertRaisesRegex(Refusal,'changed type'):gate.source_state(repo,git)

    def test_full_raw_chain_and_closed_distinct_fixture_role(self):
        for role in gate.ROLES:
            with self.subTest(role=role),tempfile.TemporaryDirectory() as temporary:
                config,path=fixture(Path(temporary),role)
                with patch.object(gate,'provision_config',return_value=config):
                    receipt=gate.read_build(path,expected_role=role)
                self.assertEqual(receipt['role'],role);self.assertEqual(receipt['features'],gate.ROLES[role])
                self.assertNotIn('--identity',receipt['command'])
                self.assertEqual(receipt['helper_features'],[])
                frozen=dict(config,binary=receipt['binary'],binary_sha256=receipt['binary_sha256'],identity=dict(host=receipt['host_sha256'],helper=receipt['helper_sha256']),
                    files=dict(receipt['inputs'],**{str(path):sha(path)}))
                with patch.object(gate,'provision_config',return_value=config):
                    if role=='release':gate.read_build(path,frozen)
                    elif role=='runtime-harness':
                        with self.assertRaises(Refusal):gate.read_build(path,frozen)
                        frozen['stages']={f'runtime-{index}':dict(harness=receipt['binary'],harness_build_receipt=str(path)) for index in range(32)}
                        gate.read_build(path,frozen)
                        with self.assertRaises(Refusal):gate.read_build(path,frozen,expected_role='release')
                    else:
                        with self.assertRaises(Refusal):gate.read_build(path,frozen)
                        frozen['live_corpus']=dict(roles={'host-fixture':dict(binary=receipt['binary'],build_receipt=str(path))})
                        gate.read_build(path,frozen)
                        with self.assertRaises(Refusal):gate.read_build(path,frozen,expected_role='release')

    def test_runtime_harness_is_a_fixed_example_with_ordinary_libraries(self):
        self.assertIn('runtime-harness',gate.ROLES)
        with tempfile.TemporaryDirectory() as temporary:
            config,path=fixture(Path(temporary),'runtime-harness')
            with patch.object(gate,'provision_config',return_value=config):
                receipt=gate.read_build(path,expected_role='runtime-harness')
            self.assertEqual(receipt['features'],[])
            self.assertEqual(receipt['helper_features'],[])
            self.assertEqual(receipt['command'][receipt['command'].index('--example')+1],'runtime-qualification-replay')
            self.assertNotIn('--bin',receipt['command'])
            self.assertTrue(receipt['binary'].endswith('/admitted/runtime-qualification-replay'))
            self.assertTrue(receipt['cargo_artifact'].endswith('/target/release/examples/runtime-qualification-replay'))
            root=path.parent;raw=root/'cargo-build.stdout';frames=[json.loads(line) for line in raw.read_text().splitlines()]
            artifact=next(row for row in frames if row.get('executable'))
            artifact['target']['kind']=['bin']
            raw.write_text('\n'.join(json.dumps(row) for row in frames))
            with patch.object(gate,'provision_config',return_value=config),self.assertRaises(Refusal):
                gate.read_build(path)

    def test_freezer_includes_complete_one_harness_custody_for_all_original_shards(self):
        import configure
        with tempfile.TemporaryDirectory() as temporary:
            config,path=fixture(Path(temporary),'runtime-harness')
            receipt=json.loads(path.read_text())
            config['stages'].update({f'runtime-{index}':dict(adapter='runtime',harness=receipt['binary'],harness_build_receipt=str(path)) for index in range(32)})
            with patch.object(gate,'provision_config',return_value=config):
                self.assertEqual(configure.runtime_harness_inputs(config),set(receipt['inputs'])|{str(path)})
                config['stages']['runtime-31']['harness']='/foreign/harness'
                with self.assertRaises(Refusal):configure.runtime_harness_inputs(config)
                config['stages']['runtime-31']['harness']=receipt['binary']
                config['stages']['runtime-31']['harness_build_receipt']='/foreign/receipt'
                with self.assertRaises(Refusal):configure.runtime_harness_inputs(config)

    def test_artifact_compiler_features_source_and_identity_tampering(self):
        for role,mutation in itertools.product(gate.ROLES,('features','helper-features','source','profile','package','executable','duplicate','completion','compiled-id','compiler','capture','name','fresh')):
            with self.subTest(role=role,mutation=mutation),tempfile.TemporaryDirectory() as temporary:
                config,path=fixture(Path(temporary),role);root=path.parent;raw=root/'cargo-build.stdout';frames=[json.loads(line) for line in raw.read_text().splitlines()]
                artifact=next(row for row in frames if row.get('executable'))
                if mutation=='features':artifact['features']=['foreign-feature']
                if mutation=='helper-features':frames[-2]['features']=['diagnostic-profile']
                if mutation=='name':artifact['target']['name']='foreign'
                if mutation=='fresh':artifact['fresh']=True
                if mutation=='source':artifact['target']['src_path']='/caller/main.rs'
                if mutation=='profile':artifact['profile']['test']=True
                if mutation=='package':artifact['package_id']='foreign#gomad@0.1.0'
                if mutation=='executable':artifact['executable']='/caller/gomad'
                if mutation=='duplicate':frames.insert(0,copy.deepcopy(artifact))
                if mutation=='completion':frames[-1]['success']=False
                if mutation=='compiled-id':frames[0]['env'][-1][1]='0'*64
                if mutation=='compiler':(root/'rustc-version.stdout').write_text('rustc 2.0.0\n')
                if mutation=='capture':
                    generated=Path(frames[0]['out_dir']);(generated/'measurement-source-inputs.tsv').write_text('caller source')
                raw.write_text('\n'.join(json.dumps(row) for row in frames))
                with patch.object(gate,'provision_config',return_value=config),self.assertRaises(Refusal):gate.read_build(path)

    def test_resealed_parent_clock_environment_publication_and_wrapper_refuse(self):
        for mutation in ('parent','clock','environment','binary','legacy-wrapper','receipt-exit'):
            with self.subTest(mutation=mutation),tempfile.TemporaryDirectory() as temporary:
                config,path=fixture(Path(temporary));root=path.parent
                if mutation in ('parent','clock','environment'):
                    for suffix in ('command','process'):
                        raw=root/('cargo-build.'+suffix+'.json');row=json.loads(raw.read_text())
                        if mutation=='parent':row.update(parent_pid=777,stage_pid=777)
                        if mutation=='clock':row['clock']='CLOCK_REALTIME'
                        if mutation=='environment':row['effective_environment']['LD_PRELOAD']='/caller/hook.so'
                        raw.write_text(json.dumps(row))
                if mutation=='binary':(root/'admitted/gomad').write_bytes(b'caller executable')
                if mutation=='legacy-wrapper':path.write_text(json.dumps(dict(profile='release',features=[],exit=0,binary_sha256='0'*64)))
                if mutation=='receipt-exit':
                    row=json.loads(path.read_text());row['exit']=False;path.write_text(json.dumps(row))
                with patch.object(gate,'provision_config',return_value=config),self.assertRaises(Refusal):gate.read_build(path)


if __name__=='__main__':unittest.main()
